package pre_auth

import (
	"net/http"
	"strings"
)

func extractBearerToken(req *http.Request, tokenHeader string) (string, bool) {
	if req == nil {
		return "", false
	}
	header := tokenHeader
	if header == "" {
		header = "Authorization"
	}
	value := strings.TrimSpace(req.Header.Get(header))
	if value == "" {
		return "", false
	}
	parts := strings.SplitN(value, " ", 2)
	if len(parts) != 2 {
		return "", false
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}
	return token, true
}
