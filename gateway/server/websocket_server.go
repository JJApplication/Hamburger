package server

import (
	"Hamburger/internal/config"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
)

func wrapHandlerWithWebSocket(h http.Handler, logger *zerolog.Logger, serverConfig config.ServerConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isWebSocketUpgrade(r) {
			h.ServeHTTP(w, r)
			return
		}
		if !isWebSocketEnabled() {
			logger.Warn().Str("host", r.Host).Msg("websocket disabled")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("websocket disabled"))
			return
		}
		if !isWsDomainAllowed(r.Host, serverConfig) {
			logger.Warn().Str("host", r.Host).Msg("websocket domain not allowed")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("websocket domain not allowed"))
			return
		}

		h.ServeHTTP(w, r)
	})
}

func isWebSocketEnabled() bool {
	cfg := config.Get()
	if cfg == nil {
		return false
	}
	return cfg.Features.WebSocket.Enabled
}

func isWebSocketUpgrade(r *http.Request) bool {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return false
	}
	return headerHasToken(r.Header, "Connection", "upgrade")
}

func headerHasToken(header http.Header, key string, token string) bool {
	values := header.Values(key)
	for _, value := range values {
		parts := strings.Split(value, ",")
		for _, part := range parts {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}

func isWsDomainAllowed(host string, serverConfig config.ServerConfig) bool {
	host = stripHostPort(host)
	if host == "" {
		return false
	}
	for _, domainConfig := range serverConfig.DomainConfig {
		if !domainConfig.UseWebsocket {
			continue
		}
		for _, wsDomain := range domainConfig.WsDomains {
			if matchDomain(host, wsDomain) {
				return true
			}
		}
	}
	return false
}

func stripHostPort(host string) string {
	if colonIndex := strings.LastIndex(host, ":"); colonIndex != -1 {
		return host[:colonIndex]
	}
	return host
}

func matchDomain(host string, configuredDomain string) bool {
	if host == configuredDomain {
		return true
	}
	return strings.HasPrefix(configuredDomain, "*.") && strings.HasSuffix(host, configuredDomain[1:])
}
