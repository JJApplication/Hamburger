package frontend_proxy

import (
	"Hamburger/internal/config"
	"Hamburger/internal/config/frontproxy_config"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestRoutingMiddlewareUsesContextWithoutHeader(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("from-context"), 0o644); err != nil {
		t.Fatal(err)
	}
	logger := zerolog.New(io.Discard)
	cfg := &frontproxy_config.PxyFrontConfig{
		InternalFlag: "X-Proxy-Internal-Front",
		Servers: []frontproxy_config.FrontServerConfig{{
			Name:  "Homeland",
			Root:  dir,
			Index: "index.html",
		}},
	}
	server := testHelios(cfg, &logger)
	router := gin.New()
	router.Use(RoutingMiddleware(server))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request = request.WithContext(WithSiteName(request.Context(), "Homeland"))
	request.Header.Set("X-Proxy-Internal-Front", "forged-site")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if !strings.Contains(recorder.Body.String(), "from-context") {
		t.Fatalf("context site was not served: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestRoutingMiddlewareHeaderFallbackStillServesDirectListener(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("from-header"), 0o644); err != nil {
		t.Fatal(err)
	}
	logger := zerolog.New(io.Discard)
	cfg := &frontproxy_config.PxyFrontConfig{
		InternalFlag: "X-Proxy-Internal-Front",
		Servers: []frontproxy_config.FrontServerConfig{{
			Name:  "Homeland",
			Root:  dir,
			Index: "index.html",
		}},
	}
	router := gin.New()
	router.Use(RoutingMiddleware(testHelios(cfg, &logger)))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Proxy-Internal-Front", "Homeland")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if !strings.Contains(recorder.Body.String(), "from-header") {
		t.Fatalf("header fallback did not serve the site: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestRoutingMiddlewareMissingSiteIsNotFound(t *testing.T) {
	logger := zerolog.New(io.Discard)
	cfg := &frontproxy_config.PxyFrontConfig{InternalFlag: "X-Proxy-Internal-Front"}
	router := gin.New()
	router.Use(RoutingMiddleware(testHelios(cfg, &logger)))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if !strings.Contains(recorder.Body.String(), "Page Not Found") {
		t.Fatalf("missing site was served: %d %s", recorder.Code, recorder.Body.String())
	}
}

func testHelios(cfg *frontproxy_config.PxyFrontConfig, logger *zerolog.Logger) *HeliosServer {
	gin.SetMode(gin.TestMode)
	index := make(map[string]frontproxy_config.FrontServerConfig, len(cfg.Servers))
	for _, server := range cfg.Servers {
		index[server.Name] = server
	}
	return &HeliosServer{
		config:       cfg,
		logger:       logger,
		cacheManager: &CacheManager{config: cfg, logger: logger, memory: newMemoryFileCache(1, 1)},
		serverIndex:  index,
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
