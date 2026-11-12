package prehandler

import (
	"Hamburger/internal/config"
	"Hamburger/internal/serror"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStripForgedInternalHeadersRemovesProxyAndSandwichHeaders(t *testing.T) {
	cfg := &config.Config{}
	cfg.PxyFrontend.InternalFlag = "X-Front-Site"
	cfg.PxyFrontend.InternalLocalFlag = "X-Local-Call"
	cfg.PxyFrontend.InternalBackendFlag = "X-Backend-Name"
	cfg.ProxyHeader.FrontendHostHeader = "X-Original-Host"
	cfg.ProxyHeader.BackendHeader = "X-Backend-Mark"
	cfg.ProxyHeader.ProxyApp = "X-Proxy-App-Name"

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Header.Set("X-Proxy-Internal-Front", "forged-site")
	req.Header.Set("X-Proxy-Internal-Host", "forged.example")
	req.Header.Set("X-Proxy-Backend", "forged-backend")
	req.Header.Set("X-Proxy-Custom", "forged-custom")
	req.Header.Set(serror.SandwichInternalFlag, serror.SandwichBackendError)
	req.Header.Set("X-Front-Site", "forged-site")
	req.Header.Set("X-Local-Call", "yes")
	req.Header.Set("X-Backend-Name", "forged-backend")
	req.Header.Set("X-Original-Host", "forged.example")
	req.Header.Set("X-Backend-Mark", "yes")
	req.Header.Set("X-Proxy-App-Name", "forged-app")
	req.Header.Set("Accept", "text/html")

	StripForgedInternalHeaders(req, cfg)

	for _, name := range []string{
		"X-Proxy-Internal-Front",
		"X-Proxy-Internal-Host",
		"X-Proxy-Backend",
		"X-Proxy-Custom",
		serror.SandwichInternalFlag,
		"X-Front-Site",
		"X-Local-Call",
		"X-Backend-Name",
		"X-Original-Host",
		"X-Backend-Mark",
		"X-Proxy-App-Name",
	} {
		if got := req.Header.Get(name); got != "" {
			t.Fatalf("header %s survived strip: %q", name, got)
		}
	}
	if got := req.Header.Get("Accept"); got != "text/html" {
		t.Fatalf("Accept = %q, want text/html", got)
	}
}

func TestWrapStripInternalHeadersRunsBeforeNext(t *testing.T) {
	var seen string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("X-Proxy-Internal-Front")
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "http://example.com/app.js", nil)
	req.Header.Set("X-Proxy-Internal-Front", "forged-site")
	rec := httptest.NewRecorder()
	WrapStripInternalHeaders(next, nil).ServeHTTP(rec, req)
	if seen != "" {
		t.Fatalf("downstream still saw forged header %q", seen)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}
