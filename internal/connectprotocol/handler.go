package connectprotocol

import (
	connectpb "Hamburger/app/connect"
	"Hamburger/gateway/api/route"
	"Hamburger/internal/config"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	"unicode"

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const DefaultBaseRoute = config.DefaultConnectProtocolBaseRoute

// ConfigProvider allows gateway restarts and in-place config reloads to
// change the Connect mount without creating another listener.
type ConfigProvider func() config.ConnectProtocolConfig

// Handler mounts the generated Connect service in front of an existing
// gateway handler. Requests outside the configured base route are delegated
// unchanged, which preserves all existing proxy behavior.
type Handler struct {
	next    http.Handler
	connect http.Handler
	config  ConfigProvider
	logger  *zerolog.Logger
}

func NewHandler(next http.Handler, apiHandler connectpb.ServiceHandler, provider ConfigProvider, logger *zerolog.Logger) (*Handler, error) {
	if next == nil {
		next = http.NotFoundHandler()
	}
	if provider == nil {
		provider = func() config.ConnectProtocolConfig {
			return config.ConnectProtocolConfig{}
		}
	}
	cfg := provider()
	if _, err := BaseRoute(cfg.BaseRoute); err != nil {
		return nil, err
	}
	if err := validateEndpointCatalog(); err != nil {
		return nil, err
	}
	if apiHandler == nil {
		// Keep a generated handler even when Connect is initially disabled. A
		// later in-place config reload can enable the mount; the service will
		// then return a normal Unavailable error instead of panicking or
		// bypassing the handler.
		apiHandler = NewService(nil)
	}
	_, generated := connectpb.NewServiceHandler(apiHandler)
	return &Handler{
		next:    next,
		connect: generated,
		config:  provider,
		logger:  logger,
	}, nil
}

func validateEndpointCatalog() error {
	methods := connectpb.File_hamburger_proto.Services().ByName("service").Methods()
	seen := make(map[string]struct{}, len(route.Endpoints()))
	for _, endpoint := range route.Endpoints() {
		if endpoint.ConnectMethod == "" || endpoint.RequestAdapter == nil || endpoint.ResponseAdapter == nil {
			return fmt.Errorf("invalid API endpoint catalog entry for %q", endpoint.Path)
		}
		if _, ok := seen[endpoint.ConnectMethod]; ok {
			return fmt.Errorf("duplicate Connect method %q in API endpoint catalog", endpoint.ConnectMethod)
		}
		seen[endpoint.ConnectMethod] = struct{}{}
		if methods.ByName(protoreflect.Name(endpoint.ConnectMethod)) == nil {
			return fmt.Errorf("Connect method %q is missing from hamburger.proto", endpoint.ConnectMethod)
		}
		if methods.ByName(protoreflect.Name(endpoint.ConnectMethod+"Stream")) == nil {
			return fmt.Errorf("Connect stream method %q is missing from hamburger.proto", endpoint.ConnectMethod+"Stream")
		}
	}
	for index := 0; index < methods.Len(); index++ {
		method := string(methods.Get(index).Name())
		if strings.HasSuffix(method, "Stream") {
			continue
		}
		if _, ok := seen[method]; !ok {
			return fmt.Errorf("Connect method %q is missing from API endpoint catalog", method)
		}
	}
	return nil
}

// New is the typed constructor used by production code.
func New(next http.Handler, apiHandler connectpb.ServiceHandler, provider ConfigProvider, logger *zerolog.Logger) (*Handler, error) {
	return NewHandler(next, apiHandler, provider, logger)
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cfg := h.config()
	if !cfg.Enabled {
		h.next.ServeHTTP(w, r)
		return
	}
	base, err := BaseRoute(cfg.BaseRoute)
	if err != nil {
		if h.logger != nil {
			h.logger.Error().Err(err).Msg("invalid ConnectProtocol base route")
		}
		h.next.ServeHTTP(w, r)
		return
	}
	if !pathUnderBase(r.URL.Path, base) {
		h.next.ServeHTTP(w, r)
		return
	}

	method := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, base), "/")
	if method == "" {
		h.connect.ServeHTTP(w, r)
		return
	}
	streamMethod := isStreamMethod(method)
	if streamMethod && !cfg.EnableBidiStream {
		// The stream methods are intentionally not mounted when disabled. Let
		// the normal gateway handler decide how an otherwise unrelated path is
		// served.
		h.next.ServeHTTP(w, r)
		return
	}

	if r.Method == http.MethodOptions {
		writeCORSPreflight(w)
		return
	}
	writeCORSHeaders(w)
	if streamMethod && r.ProtoMajor != 2 {
		// Connect's bidi wire format requires an HTTP/2 transport. Returning
		// 505 here is the standard HTTP protocol error and avoids allowing an
		// HTTP/1.1 or HTTP/3 listener to accidentally advertise a stream.
		if r.ProtoMajor < 2 {
			// Do not drain an open streaming request body before returning the
			// error: the client may be waiting for our response before sending.
			w.Header().Set("Connection", "close")
		}
		w.WriteHeader(http.StatusHTTPVersionNotSupported)
		return
	}

	// Generated handlers use the canonical protobuf procedure prefix. Rewrite
	// only the path on a shallow request copy so a custom base route remains an
	// implementation detail of this mount and never leaks to the next handler.
	canonicalPath := "/hamburger.service/" + method
	if !strings.HasPrefix(canonicalPath, "/") {
		canonicalPath = "/" + canonicalPath
	}
	request := r.Clone(r.Context())
	request.URL.Path = canonicalPath
	request.URL.RawPath = ""
	if request.Header == nil {
		request.Header = make(http.Header)
	}
	request.Header.Set("Host", r.Host)
	request.RequestURI = request.URL.RequestURI()
	h.connect.ServeHTTP(w, request)
}

func pathUnderBase(requestPath, base string) bool {
	return requestPath == base || strings.HasPrefix(requestPath, base+"/")
}

func isStreamMethod(method string) bool {
	if !strings.HasSuffix(method, "Stream") {
		return false
	}
	base := strings.TrimSuffix(method, "Stream")
	_, ok := route.EndpointForConnectMethod(base)
	return ok
}

// BaseRoute normalizes and validates the configured mount path.
func BaseRoute(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = DefaultBaseRoute
	}
	if !strings.HasPrefix(value, "/") || strings.ContainsAny(value, `\?#%`) || strings.IndexFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}) >= 0 {
		return "", ErrInvalidBaseRoute
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "." || segment == ".." {
			return "", ErrInvalidBaseRoute
		}
	}
	if strings.Contains(value, "..") {
		return "", ErrInvalidBaseRoute
	}
	normalized := path.Clean(value)
	if normalized == "/" || normalized == "." {
		return "", ErrInvalidBaseRoute
	}
	return strings.TrimRight(normalized, "/"), nil
}

// NormalizeBaseRoute is an explicit alias for callers that prefer a verb
// describing the normalization performed by BaseRoute.
func NormalizeBaseRoute(value string) (string, error) {
	return BaseRoute(value)
}

func writeCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Connect-Protocol-Version, Connect-Timeout-Ms, Authorization, X-Hamburger-Token, Grpc-Timeout, X-Grpc-Web, X-User-Agent")
	w.Header().Set("Access-Control-Expose-Headers", "Connect-Status, Connect-Message, Connect-Error-Detail-Bin")
}

func writeCORSPreflight(w http.ResponseWriter) {
	writeCORSHeaders(w)
	w.WriteHeader(http.StatusNoContent)
}

var (
	ErrInvalidBaseRoute      = errors.New("invalid ConnectProtocol base route")
	ErrAPIServiceUnavailable = errors.New("api service unavailable")
)
