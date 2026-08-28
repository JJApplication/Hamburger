package stat

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestObservingResponseWriterCountsActualPayloadBytes(t *testing.T) {
	started := time.Now().Add(-250 * time.Millisecond)
	observation := newRequestObservation(started)
	request := httptest.NewRequest(http.MethodPost, "https://Example.COM:443/upload", strings.NewReader("request-body"))
	request.Body = &countingReadCloser{ReadCloser: request.Body, count: &observation.requestBytes}
	if _, err := io.Copy(io.Discard, request.Body); err != nil {
		t.Fatalf("read request body: %v", err)
	}

	recorder := httptest.NewRecorder()
	writer := &observingResponseWriter{ResponseWriter: recorder, observation: observation}
	writer.WriteHeader(http.StatusCreated)
	if _, err := io.Copy(writer, strings.NewReader("response-body")); err != nil {
		t.Fatalf("write response body: %v", err)
	}
	request = request.WithContext(withObservation(request.Context(), observation))
	MarkRoute(request, RouteBackend)

	snapshot := observation.snapshot(request, writer.statusCode)
	if snapshot.Domain != "example.com" || snapshot.Route != RouteBackend || snapshot.StatusCode != http.StatusCreated || snapshot.RequestBytes != int64(len("request-body")) || snapshot.ResponseBytes != int64(len("response-body")) {
		t.Fatalf("snapshot = %+v", snapshot)
	}
}

func TestObservationRouteAndCounterReset(t *testing.T) {
	observation := newRequestObservation(time.Now())
	request := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	request = request.WithContext(withObservation(request.Context(), observation))
	MarkRoute(request, RouteFrontend)
	if got := codeRoute(observation.route.Load()); got != RouteFrontend {
		t.Fatalf("route = %q, want frontend", got)
	}
	if got := counterDelta(500, 20); got != 0 {
		t.Fatalf("counterDelta reset = %d, want 0", got)
	}
	if got := counterDeltaValue(500, 20, true); got != 0 {
		t.Fatalf("counterDeltaValue reset = %d, want 0", got)
	}
	if got := counterDeltaValue(500, 520, true); got != 20 {
		t.Fatalf("counterDeltaValue = %d, want 20", got)
	}
}

func withObservation(ctx context.Context, observation *RequestObservation) context.Context {
	return context.WithValue(ctx, observationContextKey{}, observation)
}

type hijackTestWriter struct {
	header http.Header
}

func (w *hijackTestWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *hijackTestWriter) Write(p []byte) (int, error) { return len(p), nil }

func (w *hijackTestWriter) WriteHeader(int) {}

func (w *hijackTestWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, nil
}

func TestObservingResponseWriterPreservesHijackMarker(t *testing.T) {
	observation := newRequestObservation(time.Now())
	writer := &observingResponseWriter{ResponseWriter: &hijackTestWriter{}, observation: observation}
	if _, _, err := writer.Hijack(); err != nil {
		t.Fatalf("hijack: %v", err)
	}
	if !observation.hijacked.Load() || writer.statusCode != http.StatusSwitchingProtocols {
		t.Fatalf("hijack marker = hijacked %v status %d", observation.hijacked.Load(), writer.statusCode)
	}
}
