package prehandler

import (
	"Hamburger/internal/config"
	"Hamburger/internal/serror"
	"net/http"
	"strings"
)

// StripForgedInternalHeaders removes client-supplied internal routing headers
// before any gateway decision treats them as trusted. It always runs and is
// independent of the optional HeaderSanitizer.
func StripForgedInternalHeaders(r *http.Request, cfg *config.Config) {
	if r == nil || r.Header == nil {
		return
	}

	for _, name := range internalHeaderNames(cfg) {
		if name != "" {
			r.Header.Del(name)
		}
	}

	var prefixed []string
	for name := range r.Header {
		if strings.HasPrefix(strings.ToLower(name), "x-proxy-") {
			prefixed = append(prefixed, name)
		}
	}
	for _, name := range prefixed {
		r.Header.Del(name)
	}
}

// WrapStripInternalHeaders strips forged internal headers on every request
// before the rest of the gateway handler chain.
func WrapStripInternalHeaders(next http.Handler, cfg *config.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		StripForgedInternalHeaders(r, cfg)
		if next != nil {
			next.ServeHTTP(w, r)
		}
	})
}

func internalHeaderNames(cfg *config.Config) []string {
	names := []string{
		serror.SandwichInternalFlag,
		"X-Proxy-Internal-Front",
		"X-Proxy-Internal-Host",
		"X-Proxy-Internal-Local",
		"X-Proxy-Backend",
	}
	if cfg == nil {
		return names
	}
	names = append(names,
		cfg.PxyFrontend.InternalFlag,
		cfg.PxyFrontend.InternalLocalFlag,
		cfg.PxyFrontend.InternalBackendFlag,
		cfg.ProxyHeader.FrontendHostHeader,
		cfg.ProxyHeader.BackendHeader,
		cfg.ProxyHeader.ProxyApp,
	)
	return names
}
