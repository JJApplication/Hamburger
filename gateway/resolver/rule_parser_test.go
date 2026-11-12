package resolver

import (
	"Hamburger/gateway/runtime"
	"Hamburger/internal/config"
	"Hamburger/internal/config/frontproxy_config"
	"Hamburger/internal/constant"
	"Hamburger/internal/structure"
	"net/http"
	"testing"
)

func TestNewRulerIndexesAPIRulesByServiceName(t *testing.T) {
	cfg := &config.Config{}
	cfg.PxyFrontend.Servers = []frontproxy_config.FrontServerConfig{
		{
			Name: "App",
			Backends: []frontproxy_config.BackendConfig{
				{
					API:        "/api",
					Service:    "Backend",
					UseRewrite: true,
					Rewrite:    "/v2",
					ProxyDirect: frontproxy_config.ProxyDirect{
						ProxyHost: "127.0.0.1",
						ProxyPort: 9000,
					},
				},
			},
		},
	}
	ruler := NewRuler(cfg, nil)
	if _, ok := ruler.apiRules["App"]; !ok {
		t.Fatal("frontend API rules were not indexed by service name")
	}
	if _, ok := ruler.apiRules["app.example.com"]; ok {
		t.Fatal("API rules still use a domain key")
	}
	req, _ := http.NewRequest(http.MethodGet, "http://app.example.com/api/users", nil)
	result, ok := ruler.MatchAPIRule(req, ruler.apiRules["App"])
	if !ok {
		t.Fatal("service-indexed API rule did not match")
	}
	if result.ProxyPath != "/v2/users" || result.ProxyHost != "127.0.0.1" || result.ProxyPort != 9000 {
		t.Fatalf("unexpected API result: %#v", result)
	}
}

func TestParseFrontendStaticDoesNotLoopToHelios(t *testing.T) {
	const host = "static.example.com"
	service := config.Service{
		ServiceName:   "Homeland",
		ServiceType:   constant.FrontendType,
		ServiceDomain: host,
	}
	installRuntimeService(t, service)

	cfg := &config.Config{}
	cfg.PxyFrontend.InternalFlag = "X-Proxy-Internal-Front"
	cfg.PxyFrontend.Host = "127.0.0.1"
	cfg.PxyFrontend.Port = 7777
	cfg.PxyFrontend.Servers = []frontproxy_config.FrontServerConfig{{
		Name: "Homeland",
		Backends: []frontproxy_config.BackendConfig{{
			API:     "/api",
			Service: "Blog",
			ProxyDirect: frontproxy_config.ProxyDirect{
				ProxyHost: "10.1.2.3",
				ProxyPort: 9000,
			},
		}},
	}}
	ruler := NewRuler(cfg, nil)

	staticReq, _ := http.NewRequest(http.MethodGet, "http://"+host+"/index.html", nil)
	staticReq.Host = host
	staticReq.Header.Set("X-Proxy-Internal-Front", "forged")
	staticResult := ruler.Parse(staticReq)
	if staticResult.ProxyError == nil || staticResult.ProxyHost != "" || staticResult.ProxyPort != 0 {
		t.Fatalf("static frontend was still proxied: %#v", staticResult)
	}
	if got := staticReq.Header.Get("X-Proxy-Internal-Front"); got != "forged" {
		t.Fatalf("Parse rewrote internal header to %q", got)
	}
	if site, ok := ruler.FrontendStaticSite(staticReq); !ok || site != "Homeland" {
		t.Fatalf("FrontendStaticSite = %q, %v", site, ok)
	}

	apiReq, _ := http.NewRequest(http.MethodGet, "http://"+host+"/api/posts", nil)
	apiReq.Host = host
	apiResult := ruler.Parse(apiReq)
	if apiResult.ProxyError != nil || apiResult.ProxyHost != "10.1.2.3" || apiResult.ProxyPort != 9000 {
		t.Fatalf("API rule was not preserved: %#v", apiResult)
	}
	if _, ok := ruler.FrontendStaticSite(apiReq); ok {
		t.Fatal("API path was treated as in-process static")
	}
}

func installRuntimeService(t *testing.T, service config.Service) {
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
