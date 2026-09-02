package frontend_proxy

import (
	"Hamburger/internal/config"
	"Hamburger/internal/config/frontproxy_config"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func TestRefreshIndexesBuildsServerLookup(t *testing.T) {
	cfg := &config.Config{}
	cfg.PxyFrontend.Servers = []frontproxy_config.FrontServerConfig{
		{Name: "First", Access: true},
		{Name: "Second", Access: false},
	}
	logger := zerolog.New(io.Discard)
	server := &HeliosServer{config: &cfg.PxyFrontend, logger: &logger}
	server.refreshIndexes()
	if got, ok := server.lookupServer("First"); !ok || !got.Access {
		t.Fatalf("First lookup = %#v, %v", got, ok)
	}
	if got, ok := server.lookupServer("Second"); !ok || got.Access {
		t.Fatalf("Second lookup = %#v, %v", got, ok)
	}
}

func TestLoggingMiddlewareSuppressesInfoForAccessDisabled(t *testing.T) {
	var output bytes.Buffer
	logger := zerolog.New(&output).Level(zerolog.InfoLevel)
	server := &HeliosServer{
		config: &frontproxy_config.PxyFrontConfig{InternalFlag: "X-Front"},
		logger: &logger,
		serverIndex: map[string]frontproxy_config.FrontServerConfig{
			"App": {Name: "App", Access: false},
		},
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(LoggingMiddleware(server))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Front", "App")
	router.ServeHTTP(httptest.NewRecorder(), request)
	if output.Len() != 0 {
		t.Fatalf("access=false emitted request log: %s", output.String())
	}
}

func TestLoggingMiddlewareKeepsDetailedInfoForAccessEnabled(t *testing.T) {
	var output bytes.Buffer
	logger := zerolog.New(&output).Level(zerolog.InfoLevel)
	server := &HeliosServer{
		config: &frontproxy_config.PxyFrontConfig{InternalFlag: "X-Front"},
		logger: &logger,
		serverIndex: map[string]frontproxy_config.FrontServerConfig{
			"App": {Name: "App", Access: true},
		},
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(LoggingMiddleware(server))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Front", "App")
	router.ServeHTTP(httptest.NewRecorder(), request)
	if !strings.Contains(output.String(), "access log") || !strings.Contains(output.String(), "response_time") {
		t.Fatalf("detailed access log missing: %s", output.String())
	}
}

func BenchmarkServerIndexLookup(b *testing.B) {
	server := &HeliosServer{
		serverIndex: make(map[string]frontproxy_config.FrontServerConfig, 4096),
	}
	for i := 0; i < 4096; i++ {
		server.serverIndex["site-"+strconv.Itoa(i)] = frontproxy_config.FrontServerConfig{Name: "site-" + strconv.Itoa(i)}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := server.lookupServer("site-2048"); !ok {
			b.Fatal("server index lookup failed")
		}
	}
}
