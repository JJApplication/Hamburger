package resolver

import (
	"Hamburger/internal/config"
	"Hamburger/internal/config/frontproxy_config"
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
