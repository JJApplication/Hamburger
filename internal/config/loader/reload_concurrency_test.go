package loader_test

import (
	"Hamburger/gateway/api/service"
	"Hamburger/internal/config"
	"Hamburger/internal/config/loader"
	"Hamburger/internal/config/svr_config"
	"net/http"
	"sync"
	"testing"
)

func TestReloadConcurrentWithAuthenticationAndSnapshots(t *testing.T) {
	previous := loader.Get()
	t.Cleanup(func() { loader.Set(previous) })
	current := &config.Config{}
	loader.Set(current)
	api := service.NewAPIService(svr_config.ApiServerConfig{
		JWT: svr_config.JWTConfig{Enabled: true, Secret: "reload-test"},
	})
	var wg sync.WaitGroup
	for reader := 0; reader < 4; reader++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 2000; n++ {
				api.AuthorizeHeaders(http.Header{}, "example.com")
				for _, snapshot := range []*config.Config{loader.Snapshot(), loader.SnapshotOf(current)} {
					if snapshot.DevMode != snapshot.ConnectProtocol.Enabled {
						t.Error("read a partially published configuration")
						return
					}
				}
			}
		}()
	}
	for n := 0; n < 2000; n++ {
		enabled := n%2 == 0
		target := current
		if enabled {
			target = nil // Exercise both management reload target forms.
		}
		loader.ReplaceInPlace(target, &config.Config{
			DevMode: enabled, ConnectProtocol: config.ConnectProtocolConfig{Enabled: enabled},
		})
	}
	wg.Wait()
	loader.ReplaceInPlace(nil, &config.Config{DevMode: true})
	if !api.AuthorizeHeaders(http.Header{}, "example.com") {
		t.Fatal("development-mode reload was not visible to authentication")
	}
	loader.ReplaceInPlace(current, &config.Config{})
	if api.AuthorizeHeaders(http.Header{}, "example.com") {
		t.Fatal("authentication still bypassed after development mode was disabled")
	}
}
