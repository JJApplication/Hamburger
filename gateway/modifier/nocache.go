package modifier

import (
	"Hamburger/gateway/resolver"
	"Hamburger/internal/config"
	"net/http"
)

type NoCache struct {
	enabled bool
}

func NewNoCache() *NoCache {
	cfg := config.Get()
	mod := new(NoCache)
	mod.enabled = cfg.Middleware.NoCache
	return mod
}

func (n NoCache) Use(response *http.Response) bool {
	if !n.enabled {
		return true
	}
	// 首先判断请求头中的cache
	cacheHeader := response.Header.Get("Cache-Control")
	if cacheHeader != "" {
		response.Header.Add("Cache-Control", cacheHeader)
	} else {
		if resolver.IsBackend(response.Request) {
			response.Header.Add("Cache-Control", "no-cache")
		}
	}
	return true
}

func (n NoCache) ModifyResponse(response *http.Response) error {
	if !n.Use(response) {
		return nil
	}
	return nil
}

func (n NoCache) IsEnabled() bool {
	return n.enabled
}

func (n NoCache) UpdateConfig() {
	//TODO implement me
	panic("implement me")
}

func (n NoCache) GetName() string {
	return "nocache"
}
