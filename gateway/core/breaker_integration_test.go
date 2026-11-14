package core

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"Hamburger/gateway/breaker"
	"Hamburger/gateway/error_page"
	"Hamburger/gateway/runtime"
	"Hamburger/internal/config"
	"Hamburger/internal/config/core_config"
	"Hamburger/internal/config/loader"
	"Hamburger/internal/constant"

	"github.com/rs/zerolog"
)

func TestProxyTransportsOpenBreakerAndRecover(t *testing.T) {
	for _, mode := range []string{constant.ProxyMode_HTTP, constant.ProxyMode_FastHTTP} {
		t.Run(mode, func(t *testing.T) {
			var hits atomic.Int32
			var fail atomic.Bool
			fail.Store(true)
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				if fail.Load() {
					http.Error(w, "down", http.StatusServiceUnavailable)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(upstream.Close)

			proxy := newBreakerProxy(t, mode, "status-"+mode, upstream.Listener.Addr().String(), 2, 1)
			host := "status-" + mode + ".example.com"

			for i := 0; i < 2; i++ {
				if status := doBreakerRequest(proxy, host); status != http.StatusServiceUnavailable {
					t.Fatalf("failure %d status = %d", i+1, status)
				}
			}
			if hits.Load() != 2 {
				t.Fatalf("upstream hits = %d before opening", hits.Load())
			}
			if status := doBreakerRequest(proxy, host); status != http.StatusServiceUnavailable {
				t.Fatalf("open circuit status = %d", status)
			}
			if hits.Load() != 2 {
				t.Fatalf("open circuit still reached upstream, hits = %d", hits.Load())
			}
			if state := breakerState("status-" + mode); state != breaker.StateOpen {
				t.Fatalf("state = %s", state)
			}

			time.Sleep(1500 * time.Millisecond)
			fail.Store(false)
			if status := doBreakerRequest(proxy, host); status != http.StatusOK {
				t.Fatalf("probe status = %d", status)
			}
			if status := doBreakerRequest(proxy, host); status != http.StatusOK {
				t.Fatalf("recovered status = %d", status)
			}
			if hits.Load() != 4 {
				t.Fatalf("recovered hits = %d, want 4", hits.Load())
			}
			if state := breakerState("status-" + mode); state != breaker.StateClosed {
				t.Fatalf("recovered state = %s", state)
			}
		})
	}
}

func TestProxyBreakerTripsOnConnectionFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()

	proxy := newBreakerProxy(t, constant.ProxyMode_HTTP, "dead-service", address, 2, 1)
	host := "dead-service.example.com"
	for i := 0; i < 2; i++ {
		if status := doBreakerRequest(proxy, host); status != http.StatusBadGateway {
			t.Fatalf("connect failure %d status = %d", i+1, status)
		}
	}
	if status := doBreakerRequest(proxy, host); status != http.StatusServiceUnavailable {
		t.Fatalf("open circuit status = %d", status)
	}
	if state := breakerState("dead-service"); state != breaker.StateOpen {
		t.Fatalf("state = %s", state)
	}
}

func TestProxyBreakerTripsWhenBackendPortsAreMissing(t *testing.T) {
	proxy := newBreakerProxy(t, constant.ProxyMode_HTTP, "missing-ports", "", 2, 1)
	host := "missing-ports.example.com"
	for i := 0; i < 2; i++ {
		if status := doBreakerRequest(proxy, host); status != http.StatusInternalServerError {
			t.Fatalf("route failure %d status = %d", i+1, status)
		}
	}
	if status := doBreakerRequest(proxy, host); status != http.StatusServiceUnavailable {
		t.Fatalf("open circuit status = %d", status)
	}
}

func TestClientCancelDoesNotTripBreaker(t *testing.T) {
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var releaseOnce sync.Once
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case started <- struct{}{}:
		default:
		}
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(release) })
		upstream.Close()
	})

	proxy := newBreakerProxy(t, constant.ProxyMode_HTTP, "cancel-service", upstream.Listener.Addr().String(), 1, 1)
	host := "cancel-service.example.com"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "http://"+host+"/slow", nil)
	req.Host = host
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		proxy.ServeHTTP(rec, req)
		close(done)
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("upstream was not reached")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("canceled request did not return")
	}
	var found bool
	for _, snapshot := range breaker.Snapshot() {
		if snapshot.Key != "cancel-service" {
			continue
		}
		found = true
		if snapshot.State != breaker.StateClosed || snapshot.ConsecutiveFailures != 0 {
			t.Fatalf("client cancel changed breaker: %+v", snapshot)
		}
	}
	if !found {
		t.Fatal("canceled request did not reach the breaker")
	}
}

func TestProxyConstructorsExposeHandlers(t *testing.T) {
	cfg := breakerTestConfig(2, 1)
	previous := loader.Get()
	loader.Set(cfg)
	t.Cleanup(func() { loader.Set(previous) })
	logger := zerolog.New(io.Discard)
	if NewHttpProxy(cfg, &logger) == nil || NewFastProxy(cfg, &logger) == nil {
		t.Fatal("proxy constructor returned nil")
	}
}

func newBreakerProxy(t *testing.T, mode, service, address string, maxError, successThreshold int) http.Handler {
	t.Helper()
	cfg := breakerTestConfig(maxError, successThreshold)
	previous := loader.Get()
	loader.Set(cfg)
	previousPage := error_page.EPM
	pageLogger := zerolog.New(io.Discard)
	error_page.InitErrorPageManager(cfg, &pageLogger)
	breaker.InitBreaker()
	breaker.ReloadConfig(cfg.Features.Break)
	breaker.ResetAll()
	t.Cleanup(func() {
		breaker.ResetAll()
		loader.Set(previous)
		error_page.EPM = previousPage
	})

	port := 0
	if address != "" {
		_, portText, err := net.SplitHostPort(address)
		if err != nil {
			t.Fatal(err)
		}
		port = atoiPort(t, portText)
	}
	installBreakerService(t, service, service+".example.com", port)

	logger := zerolog.New(io.Discard)
	return &httputil.ReverseProxy{
		Director:       ProxyDirector(cfg, &logger),
		Transport:      &myTransport{Transport: roundTripperForMode(mode), conf: cfg},
		ModifyResponse: ProxyModifyResponse(cfg, &logger),
		ErrorHandler:   ProxyErrorHandler(&logger),
	}
}

func roundTripperForMode(mode string) http.RoundTripper {
	if mode == constant.ProxyMode_FastHTTP {
		return NewFastRoundTripper()
	}
	return OriginRoundTrip()
}

func breakerTestConfig(maxError, successThreshold int) *config.Config {
	cfg := &config.Config{}
	cfg.ProxyHeader.TraceId = "X-Gateway-Trace-Id"
	cfg.ProxyHeader.FrontendHostHeader = "X-Proxy-Internal-Host"
	cfg.ErrorConfig.ErrorMode = error_page.ErrorModeText
	cfg.Features.Break = core_config.BreakConfig{
		MaxError:            maxError,
		Reset:               1,
		Bucket:              8,
		HalfOpenMaxRequests: 1,
		SuccessThreshold:    successThreshold,
		TripStatusCodes:     []int{http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout},
	}
	return cfg
}

func installBreakerService(t *testing.T, name, host string, port int) {
	t.Helper()
	installCoreService(t, config.Service{
		ServiceName:   name,
		ServiceType:   constant.BackendType,
		ServiceDomain: host,
	})
	if port > 0 {
		runtime.ServicePortsMap.Put(name, []int{port})
		t.Cleanup(func() {
			runtime.ServicePortsMap.Delete(name)
		})
		return
	}
	runtime.ServicePortsMap.Delete(name)
}

func doBreakerRequest(proxy http.Handler, host string) int {
	req := httptest.NewRequest(http.MethodGet, "http://"+host+"/api", nil)
	req.Host = host
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)
	return rec.Code
}

func breakerState(key string) breaker.State {
	for _, snapshot := range breaker.Snapshot() {
		if snapshot.Key == key {
			return snapshot.State
		}
	}
	return ""
}

func atoiPort(t *testing.T, text string) int {
	t.Helper()
	port := 0
	for _, ch := range text {
		if ch < '0' || ch > '9' {
			t.Fatalf("port %q is not numeric", text)
		}
		port = port*10 + int(ch-'0')
	}
	return port
}
