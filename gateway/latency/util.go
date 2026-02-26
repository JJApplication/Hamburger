package latency

import (
	"Hamburger/internal/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type LatencyResponse struct {
	Domain        string `json:"domain"`
	Normalized    string `json:"normalized"`
	StatusCode    int    `json:"status_code"`
	DurationMs    int64  `json:"duration_ms"`
	BodySize      int64  `json:"body_size"`
	ContentLength int64  `json:"content_length"`
	ErrorMessage  string `json:"error,omitempty"`
}

func writeLatencyResponse(w http.ResponseWriter, statusCode int, resp LatencyResponse) {
	w.WriteHeader(statusCode)
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_, _ = w.Write(data)
}

func normalizeDomain(domainAddress string) (string, string, error) {
	if strings.TrimSpace(domainAddress) == "" {
		return "", "", fmt.Errorf("domain is empty")
	}
	parsed, err := url.Parse(domainAddress)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		parsed, err = url.Parse("http://" + domainAddress)
		if err != nil {
			return "", "", fmt.Errorf("invalid domain")
		}
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", "", fmt.Errorf("invalid scheme")
	}
	host := parsed.Hostname()
	if host == "" || net.ParseIP(host) != nil || !isValidDomain(host) {
		return "", "", fmt.Errorf("invalid host")
	}
	normalized := fmt.Sprintf("%s://%s", parsed.Scheme, host)
	return normalized, host, nil
}

func doLatencyRequest(domainURL string) (LatencyResponse, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	req, err := http.NewRequest(http.MethodGet, domainURL, nil)
	if err != nil {
		return LatencyResponse{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return LatencyResponse{}, err
	}
	defer resp.Body.Close()
	bodySize, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		return LatencyResponse{}, err
	}
	return LatencyResponse{
		Domain:        req.URL.Hostname(),
		Normalized:    domainURL,
		StatusCode:    resp.StatusCode,
		BodySize:      bodySize,
		ContentLength: resp.ContentLength,
	}, nil
}

func isDomainInList(host string, normalized string, list []string) bool {
	for _, item := range list {
		if strings.EqualFold(item, host) || strings.EqualFold(item, normalized) {
			return true
		}
	}
	return false
}

func isValidDomain(host string) bool {
	if len(host) > 253 {
		return false
	}
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if part == "" || len(part) > 63 {
			return false
		}
		if part[0] == '-' || part[len(part)-1] == '-' {
			return false
		}
		for i := 0; i < len(part); i++ {
			ch := part[i]
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' {
				continue
			}
			return false
		}
	}
	return true
}

func getClientIPFromRequest(req *http.Request) string {
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xri := req.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	ip := req.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}
