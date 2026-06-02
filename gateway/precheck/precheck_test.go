package precheck

import (
	"Hamburger/gateway/runtime"
	"Hamburger/internal/config"
	"Hamburger/internal/config/loader"
	"Hamburger/internal/structure"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEffectivePrecheckConfig_GlobalFallback(t *testing.T) {
	loader.Set(&config.Config{
		GlobalPreCheck: config.PreCheckConfig{
			TTLSeconds:       123,
			CacheMaxMB:       77,
			PathPrefix:       "/__guard",
			ExcludePaths:     []string{"/health"},
			ExcludeExtensions: []string{".jpg"},
			VerifyTimeout:    9,
		},
	})

	pc := effectivePreCheckConfig(config.PreCheckConfig{Enabled: true})
	if !pc.Enabled {
		t.Fatal("expected enabled")
	}
	if pc.TTLSeconds != 123 || pc.CacheMaxMB != 77 || pc.PathPrefix != "/__guard" {
		t.Fatalf("unexpected merged config: %+v", pc)
	}
	if len(pc.ExcludePaths) != 1 || pc.ExcludePaths[0] != "/health" {
		t.Fatalf("unexpected exclude paths: %+v", pc.ExcludePaths)
	}
	if len(pc.ExcludeExtensions) != 1 || pc.ExcludeExtensions[0] != ".jpg" {
		t.Fatalf("unexpected exclude extensions: %+v", pc.ExcludeExtensions)
	}
	if pc.VerifyTimeout != 9 {
		t.Fatalf("unexpected verify timeout: %d", pc.VerifyTimeout)
	}
}

func TestShouldSkipPrecheck(t *testing.T) {
	pc := config.PreCheckConfig{
		ExcludePaths:      []string{"/health", "/api/public"},
		ExcludeExtensions: []string{".jpg", "png", ".webp"},
	}
	pc = normalizePreCheckConfig(pc)

	if !shouldSkipPrecheck("/health", pc) {
		t.Fatal("expected /health skipped")
	}
	if !shouldSkipPrecheck("/api/public/foo", pc) {
		t.Fatal("expected /api/public/foo skipped")
	}
	if !shouldSkipPrecheck("/static/a.jpg", pc) {
		t.Fatal("expected .jpg skipped")
	}
	if !shouldSkipPrecheck("/static/a.PNG", pc) {
		t.Fatal("expected .png skipped")
	}
	if shouldSkipPrecheck("/index.html", pc) {
		t.Fatal("html should not be skipped")
	}
}

func TestSanitizeReturnURL(t *testing.T) {
	if got := sanitizeReturnURL("https://evil.com"); got != "/" {
		t.Fatalf("got %q want /", got)
	}
	if got := sanitizeReturnURL("foo"); got != "/foo" {
		t.Fatalf("got %q want /foo", got)
	}
	// 修复 JS 误传 u=%22%2F%22（即 "/"）时路径不断累积 %22 的问题
	if got := sanitizeReturnURL(`"/"`); got != "/" {
		t.Fatalf("got %q want /", got)
	}
	if got := sanitizeReturnURL("/%22/%22"); got != "/" {
		t.Fatalf("got %q want /", got)
	}
}

func TestPrecheck_PageRendersWithIPFallback(t *testing.T) {
	loader.Set(&config.Config{})
	runtime.DomainLock.Lock()
	runtime.DomainsRuntimeMap.ServiceMap = structure.NewMap[config.Service]()
	runtime.DomainsRuntimeMap.DomainsMap = structure.NewMap[config.Service]()
	runtime.DomainsRuntimeMap.ServiceMap.Put("HomelandV2", config.Service{
		ServiceName:   "HomelandV2",
		ServiceDomain: "renj.io",
		PreCheck:      config.PreCheckConfig{Enabled: true},
	})
	runtime.DomainLock.Unlock()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	h := NewRouter(next).Handler()

	pc := effectivePreCheckConfig(config.PreCheckConfig{Enabled: true})
	req := httptest.NewRequest(http.MethodGet, pc.PathPrefix+"?u=https://evil.com", nil)
	req.Host = "192.168.1.1:88"
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "正在进行安全检查") {
		t.Fatalf("expected challenge page html")
	}
	if strings.Contains(body, "https://evil.com") {
		t.Fatalf("absolute return url should be sanitized")
	}
}
