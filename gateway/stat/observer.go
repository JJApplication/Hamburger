package stat

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

// RouteType is the final gateway route classification used by historical
// statistics. Unknown is intentionally retained for requests rejected before
// the resolver can make a frontend/backend decision.
type RouteType string

const (
	RouteUnknown  RouteType = "unknown"
	RouteFrontend RouteType = "frontend"
	RouteBackend  RouteType = "backend"
)

type observationContextKey struct{}

// RequestSnapshot is the immutable result of observing one HTTP request.
type RequestSnapshot struct {
	StartedAt     time.Time
	Duration      time.Duration
	Domain        string
	Route         RouteType
	StatusCode    int
	RequestBytes  int64
	ResponseBytes int64
	Hijacked      bool
}

// RequestObservation is stored in the request context and updated by the
// outer response/request wrappers and by the resolver.
type RequestObservation struct {
	startedAt     time.Time
	route         atomic.Int32
	statusCode    atomic.Int64
	requestBytes  atomic.Int64
	responseBytes atomic.Int64
	hijacked      atomic.Bool
}

func newRequestObservation(now time.Time) *RequestObservation {
	obs := &RequestObservation{startedAt: now}
	obs.route.Store(routeCode(RouteUnknown))
	return obs
}

func routeCode(route RouteType) int32 {
	switch route {
	case RouteFrontend:
		return 1
	case RouteBackend:
		return 2
	default:
		return 0
	}
}

func codeRoute(code int32) RouteType {
	switch code {
	case 1:
		return RouteFrontend
	case 2:
		return RouteBackend
	default:
		return RouteUnknown
	}
}

// ObservationFromContext returns the mutable observation attached to ctx.
func ObservationFromContext(ctx context.Context) *RequestObservation {
	if ctx == nil {
		return nil
	}
	obs, _ := ctx.Value(observationContextKey{}).(*RequestObservation)
	return obs
}

// ObservationFromRequest returns the observation attached by WrapHTTPHandler.
func ObservationFromRequest(req *http.Request) *RequestObservation {
	if req == nil {
		return nil
	}
	return ObservationFromContext(req.Context())
}

// MarkRoute records the final route selected by the resolver or a static
// alias handler.
func MarkRoute(req *http.Request, route RouteType) {
	if obs := ObservationFromRequest(req); obs != nil {
		obs.route.Store(routeCode(route))
	}
}

func (o *RequestObservation) snapshot(req *http.Request, statusCode int) RequestSnapshot {
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	domain := ""
	if req != nil {
		domain = NormalizeDomain(req.Host)
	}
	return RequestSnapshot{
		StartedAt:     o.startedAt,
		Duration:      time.Since(o.startedAt),
		Domain:        domain,
		Route:         codeRoute(o.route.Load()),
		StatusCode:    statusCode,
		RequestBytes:  o.requestBytes.Load(),
		ResponseBytes: o.responseBytes.Load(),
		Hijacked:      o.hijacked.Load(),
	}
}

// WrapHTTPHandler installs the gateway's outermost request observation. It
// counts bytes actually read/written instead of trusting Content-Length and
// forwards the optional ResponseWriter interfaces used by streaming and
// protocol upgrades.
func WrapHTTPHandler(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		obs := newRequestObservation(time.Now().UTC())
		ctx := context.WithValue(req.Context(), observationContextKey{}, obs)
		req = req.WithContext(ctx)
		if req.Body != nil {
			req.Body = &countingReadCloser{ReadCloser: req.Body, count: &obs.requestBytes}
		}

		writer := &observingResponseWriter{ResponseWriter: w, observation: obs}
		defer func() {
			statusCode := writer.statusCode
			if statusCode == 0 {
				statusCode = http.StatusOK
			}
			GetManager().RecordObservation(obs.snapshot(req, statusCode))
		}()
		next.ServeHTTP(writer, req)
	})
}

type countingReadCloser struct {
	io.ReadCloser
	count *atomic.Int64
}

func (r *countingReadCloser) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n > 0 {
		r.count.Add(int64(n))
	}
	return n, err
}

type observingResponseWriter struct {
	http.ResponseWriter
	observation *RequestObservation
	statusCode  int
}

func (w *observingResponseWriter) WriteHeader(statusCode int) {
	if w.statusCode != 0 {
		return
	}
	w.statusCode = statusCode
	w.observation.statusCode.Store(int64(statusCode))
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *observingResponseWriter) Write(p []byte) (int, error) {
	if w.statusCode == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(p)
	if n > 0 {
		w.observation.responseBytes.Add(int64(n))
	}
	return n, err
}

func (w *observingResponseWriter) WriteString(value string) (int, error) {
	if writer, ok := w.ResponseWriter.(io.StringWriter); ok {
		if w.statusCode == 0 {
			w.WriteHeader(http.StatusOK)
		}
		n, err := writer.WriteString(value)
		if n > 0 {
			w.observation.responseBytes.Add(int64(n))
		}
		return n, err
	}
	return w.Write([]byte(value))
}

func (w *observingResponseWriter) Flush() {
	if w.statusCode == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *observingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("http hijacking is not supported")
	}
	w.observation.hijacked.Store(true)
	if w.statusCode == 0 {
		// A hijacked HTTP connection has no later ResponseWriter status. 101 is
		// the useful classification for dashboards while preserving the raw
		// connection bytes outside HTTP payload accounting.
		w.statusCode = http.StatusSwitchingProtocols
		w.observation.statusCode.Store(int64(w.statusCode))
	}
	return hijacker.Hijack()
}

func (w *observingResponseWriter) Push(target string, opts *http.PushOptions) error {
	pusher, ok := w.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

func (w *observingResponseWriter) ReadFrom(src io.Reader) (int64, error) {
	if w.statusCode == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if readerFrom, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		n, err := readerFrom.ReadFrom(src)
		if n > 0 {
			w.observation.responseBytes.Add(n)
		}
		return n, err
	}
	var total int64
	buf := make([]byte, 32*1024)
	for {
		n, readErr := src.Read(buf)
		if n > 0 {
			written, writeErr := w.Write(buf[:n])
			total += int64(written)
			if writeErr != nil {
				return total, writeErr
			}
			if written != n {
				return total, io.ErrShortWrite
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				return total, nil
			}
			return total, readErr
		}
	}
}

// Unwrap lets http.ResponseController reach newer optional capabilities that
// are not expressible as the legacy interfaces above.
func (w *observingResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// CloseNotify preserves the legacy interface used by older handlers.
func (w *observingResponseWriter) CloseNotify() <-chan bool {
	if notifier, ok := w.ResponseWriter.(http.CloseNotifier); ok {
		return notifier.CloseNotify()
	}
	return make(chan bool)
}
