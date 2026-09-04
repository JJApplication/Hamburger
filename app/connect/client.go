package connect

import (
	"errors"
	"net/http"
	"path"
	"strings"
	"unicode"

	connectrpc "connectrpc.com/connect"
)

// ErrInvalidBaseRoute indicates that a configured Connect mount is unsafe or
// cannot be represented as an absolute URL path.
var ErrInvalidBaseRoute = errors.New("invalid ConnectProtocol base route")

const DefaultBaseRoute = "/hamburger.service"

// NewServiceClientAtBaseRoute keeps the generated protobuf client types while
// routing every generated procedure through the server's configurable mount.
// The ordinary NewServiceClient constructor remains available for the default
// /hamburger.service route.
func NewServiceClientAtBaseRoute(httpClient connectrpc.HTTPClient, baseURL, baseRoute string, opts ...connectrpc.ClientOption) (ServiceClient, error) {
	route, err := normalizeBaseRoute(baseRoute)
	if err != nil {
		return nil, err
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	return NewServiceClient(routeHTTPClient{delegate: httpClient, route: route}, baseURL, opts...), nil
}

// NormalizeBaseRoute applies the same route rules as the server-side mount.
// It is exported so configuration-aware clients can validate a route before
// constructing a client.
func NormalizeBaseRoute(value string) (string, error) {
	return normalizeBaseRoute(value)
}

type routeHTTPClient struct {
	delegate connectrpc.HTTPClient
	route    string
}

func (c routeHTTPClient) Do(request *http.Request) (*http.Response, error) {
	if c.delegate == nil {
		return nil, errors.New("nil Connect HTTP client")
	}
	clone := request.Clone(request.Context())
	if clone.URL != nil {
		const canonical = "/hamburger.service"
		if index := strings.LastIndex(clone.URL.Path, canonical); index >= 0 {
			originalPath := clone.URL.Path
			clone.URL.Path = originalPath[:index] + c.route + originalPath[index+len(canonical):]
			clone.URL.RawPath = ""
		}
	}
	// RequestURI must be empty for an outbound client request; the transport
	// reconstructs it from URL.
	clone.RequestURI = ""
	return c.delegate.Do(clone)
}

func normalizeBaseRoute(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = DefaultBaseRoute
	}
	if !strings.HasPrefix(value, "/") || strings.ContainsAny(value, `\?#%`) || strings.IndexFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}) >= 0 || strings.Contains(value, "..") {
		return "", ErrInvalidBaseRoute
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "." || segment == ".." {
			return "", ErrInvalidBaseRoute
		}
	}
	value = path.Clean(value)
	if value == "/" || value == "." {
		return "", ErrInvalidBaseRoute
	}
	return strings.TrimRight(value, "/"), nil
}
