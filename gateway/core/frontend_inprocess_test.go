package core

import (
	"Hamburger/frontend_proxy"
	"Hamburger/gateway/runtime"
	"Hamburger/internal/config"
	"Hamburger/internal/config/frontproxy_config"
	"Hamburger/internal/config/loader"
	"Hamburger/internal/constant"
	"Hamburger/internal/structure"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
)

func TestFrontendStaticUsesContextAndStripsForgedHeaders(t *testing.T) {
	loader.Set(&config.Config{})
	const host = "blog.example.com"
	installCoreService(t, config.Service{
		ServiceName:   "BlogFront",
		ServiceType:   constant.FrontendType,
		ServiceDomain: host,
	})

	cfg := &config.Config{}
	cfg.PxyFrontend.InternalFlag = "X-Proxy-Internal-Front"
	cfg.PxyFrontend.Servers = []frontproxy_config.FrontServerConfig{{
		Name: "BlogFront",
		Backends: []frontproxy_config.BackendConfig{{
			API:     "/api",
			Service: "Blog",
			ProxyDirect: frontproxy_config.ProxyDirect{
				ProxyHost: "10.9.8.7",
				ProxyPort: 8080,
			},
		}},
	}}
	logger := zerolog.New(io.Discard)
	proxy := NewProxy(cfg, &logger)

	var frontCalls int
	var site string
	var forged string
	proxy.SetFrontHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		frontCalls++
		site, _ = frontend_proxy.SiteNameFrom(r.Context())
		forged = r.Header.Get("X-Proxy-Internal-Front")
		if r.URL.Host == "127.0.0.1:7777" {
			t.Errorf("static request was rewritten onto the Helios listener: %s", r.URL.Host)
		}
		w.WriteHeader(http.StatusOK)
	}))
	var nextCalls int
	proxy.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalls++
		w.WriteHeader(http.StatusAccepted)
	})

	staticReq := httptest.NewRequest(http.MethodGet, "http://"+host+"/app.js", nil)
	staticReq.Host = host
	staticReq.Header.Set("X-Proxy-Internal-Front", "forged-site")
	staticReq.Header.Set("X-Proxy-Internal-Host", "forged.example")
	staticReq.Header.Set("X-Sandwich-Request-Flag", "SandwichBackendError")
	staticRec := httptest.NewRecorder()
	proxy.proxy().ServeHTTP(staticRec, staticReq)
	if frontCalls != 1 || nextCalls != 0 {
		t.Fatalf("static routing front=%d next=%d", frontCalls, nextCalls)
	}
	if site != "BlogFront" || forged != "" {
		t.Fatalf("site=%q forged header=%q", site, forged)
	}
	if staticRec.Code != http.StatusOK {
		t.Fatalf("static status = %d", staticRec.Code)
	}

	apiReq := httptest.NewRequest(http.MethodGet, "http://"+host+"/api/posts", nil)
	apiReq.Host = host
	apiRec := httptest.NewRecorder()
	proxy.proxy().ServeHTTP(apiRec, apiReq)
	if frontCalls != 1 || nextCalls != 1 {
		t.Fatalf("API routing front=%d next=%d", frontCalls, nextCalls)
	}
	if apiRec.Code != http.StatusAccepted {
		t.Fatalf("API status = %d, want reverse-proxy continuation", apiRec.Code)
	}
}

func installCoreService(t *testing.T, service config.Service) {
	t.Helper()
	runtime.DomainLock.Lock()
	previous := runtime.DomainsRuntimeMap
	domains := structure.NewMap[config.Service]()
	services := structure.NewMap[config.Service]()
	domains.Put(service.ServiceDomain, service)
	services.Put(service.ServiceName, service)
	runtime.DomainsRuntimeMap.Domains = []string{service.ServiceDomain}
	runtime.DomainsRuntimeMap.DomainsMap = domains
	runtime.DomainsRuntimeMap.RegexDomains = nil
	runtime.DomainsRuntimeMap.DomainFrontMap = structure.NewMap[string]()
	runtime.DomainsRuntimeMap.ServiceMap = services
	runtime.DomainLock.Unlock()
	t.Cleanup(func() {
		runtime.DomainLock.Lock()
		runtime.DomainsRuntimeMap = previous
		runtime.DomainLock.Unlock()
	})
}
