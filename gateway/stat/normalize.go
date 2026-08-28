package stat

import (
	"net"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/idna"
)

// NormalizeDomain returns the canonical domain key used by all statistics.
// It accepts a Host header, an absolute URL, or a host with an optional port.
// IP literals are kept as literals; DNS names are converted to lower-case ASCII
// (IDNA) so HTTP and HTTPS requests share one bucket.
func NormalizeDomain(raw string) string {
	host := strings.TrimSpace(raw)
	if host == "" {
		return ""
	}

	if strings.Contains(host, "://") {
		if parsed, err := url.Parse(host); err == nil && parsed.Host != "" {
			host = parsed.Host
		}
	} else if strings.HasPrefix(host, "//") {
		if parsed, err := url.Parse(host); err == nil && parsed.Host != "" {
			host = parsed.Host
		}
	}
	if slash := strings.IndexAny(host, "/?#"); slash >= 0 {
		host = host[:slash]
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}

	if parsedHost, port, err := net.SplitHostPort(host); err == nil {
		if _, portErr := strconv.Atoi(port); portErr == nil || port == "" {
			host = parsedHost
		}
	} else if strings.Count(host, ":") == 1 {
		// net.SplitHostPort rejects a host without brackets in a few malformed
		// cases. Only strip the suffix when it is a numeric port.
		idx := strings.LastIndex(host, ":")
		if idx > 0 {
			if _, portErr := strconv.Atoi(host[idx+1:]); portErr == nil {
				host = host[:idx]
			}
		}
	}
	host = strings.Trim(host, "[]")
	host = strings.TrimSuffix(host, ".")
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return ""
	}
	if ascii, err := idna.Lookup.ToASCII(host); err == nil && ascii != "" {
		return strings.ToLower(ascii)
	}
	return host
}
