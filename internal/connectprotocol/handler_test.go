package connectprotocol

import (
	connectpb "Hamburger/app/connect"
	"Hamburger/gateway/api/route"
	gwserver "Hamburger/gateway/server"
	"Hamburger/internal/config"
	coreconfig "Hamburger/internal/config/core_config"
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	connectrpc "connectrpc.com/connect"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestBaseRoute(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		valid bool
	}{
		{name: "default", want: DefaultBaseRoute, valid: true},
		{name: "trailing slash", input: "/custom/", want: "/custom", valid: true},
		{name: "nested", input: "/rpc/hamburger", want: "/rpc/hamburger", valid: true},
		{name: "root", input: "/", valid: false},
		{name: "query", input: "/rpc?x=1", valid: false},
		{name: "space", input: "/rpc service", valid: false},
		{name: "backslash", input: `\rpc`, valid: false},
		{name: "dot segment", input: "/rpc/./service", valid: false},
		{name: "escaped", input: "/rpc%2Fservice", valid: false},
		{name: "relative", input: "rpc", valid: false},
		{name: "parent", input: "/rpc/../service", valid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BaseRoute(tt.input)
			if tt.valid {
				if err != nil || got != tt.want {
					t.Fatalf("BaseRoute(%q) = %q, %v; want %q", tt.input, got, err, tt.want)
				}
				return
			}
			if !errors.Is(err, ErrInvalidBaseRoute) {
				t.Fatalf("BaseRoute(%q) error = %v, want ErrInvalidBaseRoute", tt.input, err)
			}
		})
	}
}

func TestEndpointCatalogMatchesProto(t *testing.T) {
	methods := connectpb.File_hamburger_proto.Services().ByName("service").Methods()
	for _, endpoint := range route.Endpoints() {
		method := methods.ByName(protoreflect.Name(endpoint.ConnectMethod))
		if method == nil {
			t.Fatalf("missing protobuf method %q", endpoint.ConnectMethod)
		}
		if got := endpoint.RequestAdapter().ProtoReflect().Descriptor().FullName(); got != method.Input().FullName() {
			t.Fatalf("%s request = %s, want %s", endpoint.ConnectMethod, got, method.Input().FullName())
		}
		if got := endpoint.ResponseAdapter().ProtoReflect().Descriptor().FullName(); got != method.Output().FullName() {
			t.Fatalf("%s response = %s, want %s", endpoint.ConnectMethod, got, method.Output().FullName())
		}
	}
}

func TestHandlerFallsThroughWhenDisabled(t *testing.T) {
	handler, err := New(nil, NewService(nil), func() config.ConnectProtocolConfig {
		return config.ConnectProtocolConfig{Enabled: false}
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "http://example.com/hamburger.service/health", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("disabled handler status = %d, want 404", response.Code)
	}
}

func TestHandlerDoesNotMountStreamsByDefault(t *testing.T) {
	fallback := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	handler, err := New(fallback, NewService(nil), func() config.ConnectProtocolConfig {
		return config.ConnectProtocolConfig{Enabled: true, BaseRoute: DefaultBaseRoute}
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "http://example.com"+DefaultBaseRoute+"/healthStream", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusTeapot {
		t.Fatalf("disabled stream status = %d, want %d", response.Code, http.StatusTeapot)
	}
}

func TestHandlerRefreshesConfigInPlace(t *testing.T) {
	var cfg config.ConnectProtocolConfig
	fallback := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	handler, err := New(fallback, NewService(nil), func() config.ConnectProtocolConfig { return cfg }, nil)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	request, err := http.NewRequest(http.MethodPost, server.URL+DefaultBaseRoute+"/health", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusTeapot {
		response.Body.Close()
		t.Fatalf("disabled status = %d, want %d", response.StatusCode, http.StatusTeapot)
	}
	response.Body.Close()

	cfg.Enabled = true
	client := connectpb.NewServiceClient(server.Client(), server.URL)
	if _, err := client.Health(context.Background(), connectrpc.NewRequest(&connectpb.Empty{})); err != nil {
		t.Fatalf("enabled Connect request failed: %v", err)
	}
	cfg.Enabled = false
	response, err = server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusTeapot {
		t.Fatalf("re-disabled status = %d, want %d", response.StatusCode, http.StatusTeapot)
	}
}

func TestHandlerServesUnaryAtCustomRoute(t *testing.T) {
	provider := func() config.ConnectProtocolConfig {
		return config.ConnectProtocolConfig{Enabled: true, BaseRoute: "/custom"}
	}
	handler, err := New(http.NotFoundHandler(), NewService(nil), provider, nil)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	client, err := connectpb.NewServiceClientAtBaseRoute(server.Client(), server.URL+"/", "/custom/")
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Health(context.Background(), connectrpc.NewRequest(&connectpb.Empty{}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg == nil {
		t.Fatal("health response is nil")
	}
	request, err := http.NewRequest(http.MethodPost, server.URL+"/hamburger.service/health", nil)
	if err != nil {
		t.Fatal(err)
	}
	rawResponse, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer rawResponse.Body.Close()
	if rawResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("canonical route status = %d, want 404", rawResponse.StatusCode)
	}
}

func TestHandlerServesBidiStreamOverHTTP2(t *testing.T) {
	provider := func() config.ConnectProtocolConfig {
		return config.ConnectProtocolConfig{Enabled: true, BaseRoute: "/custom", EnableBidiStream: true}
	}
	handler, err := New(http.NotFoundHandler(), NewService(nil), provider, nil)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(handler)
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()
	client, err := connectpb.NewServiceClientAtBaseRoute(server.Client(), server.URL, "/custom")
	if err != nil {
		t.Fatal(err)
	}
	stream := client.HealthStream(context.Background())
	if err := stream.Send(&connectpb.Empty{}); err != nil {
		t.Fatal(err)
	}
	if _, err := stream.Receive(); err != nil {
		t.Fatal(err)
	}
	if err := stream.CloseRequest(); err != nil {
		t.Fatal(err)
	}
	if _, err := stream.Receive(); !errors.Is(err, io.EOF) {
		t.Fatalf("stream close error = %v, want EOF", err)
	}
}

func TestHandlerRejectsBidiStreamOverHTTP1(t *testing.T) {
	handler, err := New(http.NotFoundHandler(), NewService(nil), func() config.ConnectProtocolConfig {
		return config.ConnectProtocolConfig{Enabled: true, BaseRoute: DefaultBaseRoute, EnableBidiStream: true}
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	request, err := http.NewRequest(http.MethodPost, server.URL+DefaultBaseRoute+"/healthStream", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/connect+proto")
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusHTTPVersionNotSupported {
		t.Fatalf("HTTP/1 stream status = %d, want %d", response.StatusCode, http.StatusHTTPVersionNotSupported)
	}
}

func TestHTTP1BidiRejectionDoesNotWaitForRequestEOF(t *testing.T) {
	handler, err := New(nil, NewService(nil), func() config.ConnectProtocolConfig {
		return config.ConnectProtocolConfig{Enabled: true, EnableBidiStream: true}
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	conn, err := net.DialTimeout("tcp", server.Listener.Addr().String(), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	// Leave the chunked body open, as a duplex client waiting for a response
	// would. A finite request body would not expose a response/drain deadlock.
	_, err = io.WriteString(conn, "POST /hamburger.service/healthStream HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/connect+proto\r\nTransfer-Encoding: chunked\r\n\r\n2\r\n{}\r\n")
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("response must arrive before request EOF: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusHTTPVersionNotSupported || !response.Close {
		t.Fatalf("status=%d close=%v; want 505 and connection close", response.StatusCode, response.Close)
	}
}

func TestHandlerSharesGatewayListenerWithFallback(t *testing.T) {
	provider := func() config.ConnectProtocolConfig {
		return config.ConnectProtocolConfig{Enabled: true, BaseRoute: DefaultBaseRoute}
	}
	fallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("proxy"))
	})
	connectHandler, err := New(fallback, NewService(nil), provider, nil)
	if err != nil {
		t.Fatal(err)
	}
	logger := zerolog.New(io.Discard)
	instance, err := gwserver.CommonHttpServer(coreconfig.ServerConfig{
		Name:     "connect-test",
		Host:     "127.0.0.1",
		Protocol: "http",
	}, &logger, connectHandler, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = instance.Server.Serve(instance.Listener) }()
	defer func() { _ = instance.Server.Close() }()

	baseURL := "http://" + instance.Listener.Addr().String()
	client := connectpb.NewServiceClient(http.DefaultClient, baseURL)
	if _, err := client.Health(context.Background(), connectrpc.NewRequest(&connectpb.Empty{})); err != nil {
		t.Fatalf("Connect request failed on gateway listener: %v", err)
	}
	response, err := http.Get(baseURL + "/ordinary")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusTeapot {
		t.Fatalf("fallback status = %d, want %d", response.StatusCode, http.StatusTeapot)
	}
}
