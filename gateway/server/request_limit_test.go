package server

import (
	"Hamburger/internal/config/core_config"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (r *trackingReadCloser) Close() error {
	r.closed = true
	return nil
}

func TestEffectiveMaxQuerySize(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		got, err := core_config.EffectiveMaxQuerySize(0)
		if err != nil {
			t.Fatalf("resolve default: %v", err)
		}
		if got != core_config.DefaultMaxQuerySize {
			t.Fatalf("default max query size = %d, want %d", got, core_config.DefaultMaxQuerySize)
		}
	})

	t.Run("custom", func(t *testing.T) {
		got, err := core_config.EffectiveMaxQuerySize(4096)
		if err != nil {
			t.Fatalf("resolve custom value: %v", err)
		}
		if got != 4096 {
			t.Fatalf("custom max query size = %d, want 4096", got)
		}
	})

	t.Run("negative", func(t *testing.T) {
		if _, err := core_config.EffectiveMaxQuerySize(-1); err == nil {
			t.Fatal("negative max query size should be rejected")
		}
	})
}

func TestWrapHandlerWithMaxQuerySize(t *testing.T) {
	logger := zerolog.Nop()
	called := 0
	handler, err := WrapHandlerWithMaxQuerySize(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusNoContent)
	}), 0, &logger)
	if err != nil {
		t.Fatalf("create request target limiter: %v", err)
	}

	limit := int(core_config.DefaultMaxQuerySize)
	for _, tc := range []struct {
		name       string
		method     string
		requestURI string
		wantStatus int
		wantCalled int
	}{
		{
			name:       "boundary path and query",
			method:     http.MethodGet,
			requestURI: "/" + strings.Repeat("a", limit-1),
			wantStatus: http.StatusNoContent,
			wantCalled: 1,
		},
		{
			name:       "over limit post with body",
			method:     http.MethodPost,
			requestURI: "/" + strings.Repeat("a", limit),
			wantStatus: http.StatusRequestURITooLong,
			wantCalled: 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called = 0
			body := &trackingReadCloser{Reader: strings.NewReader("request body")}
			req := httptest.NewRequest(tc.method, "http://example.test/", body)
			req.RequestURI = tc.requestURI
			resp := httptest.NewRecorder()

			handler.ServeHTTP(resp, req)

			if resp.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", resp.Code, tc.wantStatus)
			}
			if called != tc.wantCalled {
				t.Fatalf("downstream call count = %d, want %d", called, tc.wantCalled)
			}
			if tc.wantStatus == http.StatusRequestURITooLong && !body.closed {
				t.Fatal("over-limit request body was not closed")
			}
		})
	}
}

func TestWrapHandlerWithMaxQuerySizeAcceptsAsteriskRequestTarget(t *testing.T) {
	logger := zerolog.Nop()
	called := false
	handler, err := WrapHandlerWithMaxQuerySize(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}), 1, &logger)
	if err != nil {
		t.Fatalf("create request target limiter: %v", err)
	}

	req := httptest.NewRequest(http.MethodOptions, "http://example.test/", nil)
	req.RequestURI = "*"
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if !called || resp.Code != http.StatusOK {
		t.Fatalf("asterisk request target was rejected: called=%v status=%d", called, resp.Code)
	}
}

func TestCommonHttpServerAppliesRequestTargetLimitBeforeOtherHandlers(t *testing.T) {
	logger := zerolog.Nop()
	called := false
	instance, err := CommonHttpServer(core_config.ServerConfig{
		Name:           "request-target-limit-test",
		Host:           "127.0.0.1",
		Port:           0,
		Protocol:       "http",
		MaxRequestBody: 1,
		DomainConfig: []core_config.DomainConfig{{
			Domains:      []string{"example.test"},
			AutoRedirect: true,
		}},
	}, &logger, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}), nil, false, 8)
	if err != nil {
		t.Fatalf("create test HTTP server: %v", err)
	}
	defer instance.Listener.Close()

	httpServer, ok := instance.Server.(*http.Server)
	if !ok {
		t.Fatalf("server type = %T, want *http.Server", instance.Server)
	}
	req := httptest.NewRequest(http.MethodPost, "http://example.test/", strings.NewReader("request body"))
	req.RequestURI = "/123456789"
	resp := httptest.NewRecorder()
	httpServer.Handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusRequestURITooLong {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusRequestURITooLong)
	}
	if called {
		t.Fatal("over-limit request reached the downstream handler")
	}
}

func TestWrapHandlerWithMaxQuerySizeCountsRawRequestTargetBytes(t *testing.T) {
	logger := zerolog.Nop()
	called := false
	handler, err := WrapHandlerWithMaxQuerySize(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}), int64(len("/path?value=%C3%A9")), &logger)
	if err != nil {
		t.Fatalf("create request target limiter: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	req.RequestURI = "/path?value=%C3%A9"
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if !called || resp.Code != http.StatusOK {
		t.Fatalf("encoded request target at byte boundary was rejected: called=%v status=%d", called, resp.Code)
	}

	called = false
	req.RequestURI = "/path?value=%C3%A9x"
	resp = httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if called || resp.Code != http.StatusRequestURITooLong {
		t.Fatalf("encoded request target over byte limit was accepted: called=%v status=%d", called, resp.Code)
	}
}
