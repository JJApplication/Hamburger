package precheck

import (
	"Hamburger/gateway/runtime"
	"Hamburger/internal/config"
	"Hamburger/internal/structure"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

	req := httptest.NewRequest(http.MethodGet, "/__precheck?u=https://evil.com", nil)
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
