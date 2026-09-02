package frontend_proxy

import (
	"Hamburger/internal/config"
	"Hamburger/internal/config/frontproxy_config"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"net/http"
	"net/http/httptest"
)

func testCacheManager(t *testing.T) (*CacheManager, string) {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "app.js")
	if err := os.WriteFile(source, []byte("console.log('cached');"), 0644); err != nil {
		t.Fatal(err)
	}
	logger := zerolog.New(io.Discard)
	cfg := &config.Config{}
	memoryEnable := true
	cfg.PxyFrontend.Cache = frontproxy_config.FrontCacheConfig{
		Enable:                  true,
		Dir:                     filepath.Join(root, "cache"),
		Matcher:                 []string{"*.js"},
		MemoryEnable:            &memoryEnable,
		MemoryMaxMB:             1,
		MemoryMaxFileMB:         1,
		MemoryMaxEntries:        4,
		MemoryRevalidateSeconds: 60,
	}
	cfg.PxyFrontend.CacheHeader = "X-Helios-Cache"
	return NewCacheManager(cfg, &logger), source
}

func testContext(method, path string, headers map[string]string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(method, path, nil)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	ctx.Request = req
	return ctx, recorder
}

func TestMemoryCacheCascadeAndRange(t *testing.T) {
	cm, source := testCacheManager(t)
	internalFlag := "App"
	requestPath := "/app.js"
	cm.CacheFile(internalFlag, requestPath, source)

	key := cm.cacheKey(internalFlag, requestPath)
	if _, ok := cm.memory.get(key); !ok {
		t.Fatal("expected source miss to populate memory cache")
	}
	if _, err := os.Stat(cm.cachedFilePath(internalFlag, requestPath)); err != nil {
		t.Fatalf("expected source miss to populate disk cache: %v", err)
	}

	ctx, recorder := testContext(http.MethodGet, requestPath, map[string]string{
		"Range": "bytes=0-6",
	})
	if !cm.ServeCached(ctx, internalFlag, requestPath, source) {
		t.Fatal("expected memory cache hit")
	}
	if recorder.Code != http.StatusPartialContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusPartialContent)
	}
	if got := recorder.Header().Get("X-Helios-Cache"); got != "True" {
		t.Fatalf("cache header = %q, want True", got)
	}
	if !strings.Contains(recorder.Body.String(), "console") {
		t.Fatalf("unexpected range body: %q", recorder.Body.String())
	}

	cm.Reset()
	ctx, recorder = testContext(http.MethodHead, requestPath, nil)
	if !cm.ServeCached(ctx, internalFlag, requestPath, source) {
		t.Fatal("expected disk fallback to populate and serve memory cache")
	}
	if recorder.Code != http.StatusOK || recorder.Body.Len() != 0 {
		t.Fatalf("HEAD response = status %d, body %d bytes", recorder.Code, recorder.Body.Len())
	}
}

func TestMemoryCacheInvalidatesChangedSource(t *testing.T) {
	cm, source := testCacheManager(t)
	internalFlag := "App"
	requestPath := "/app.js"
	cm.CacheFile(internalFlag, requestPath, source)
	key := cm.cacheKey(internalFlag, requestPath)

	entry, ok := cm.memory.get(key)
	if !ok {
		t.Fatal("expected memory entry")
	}
	cm.memory.mu.Lock()
	for _, element := range cm.memory.items {
		cached := element.Value.(memoryFileEntry)
		cached.validatedAt = time.Now().Add(-2 * time.Minute)
		element.Value = cached
	}
	cm.memory.mu.Unlock()
	if err := os.WriteFile(source, []byte("changed content"), 0644); err != nil {
		t.Fatal(err)
	}
	if cm.ServeCached(func() *gin.Context {
		ctx, _ := testContext(http.MethodGet, requestPath, nil)
		return ctx
	}(), internalFlag, requestPath, source) {
		t.Fatal("changed source must not be served from stale cache")
	}
	if _, ok := cm.memory.get(entry.key); ok {
		t.Fatal("stale memory entry was not evicted")
	}
}

func TestMemoryCacheLRUByteAndEntryLimits(t *testing.T) {
	cache := newMemoryFileCache(3, 2)
	cache.put(memoryFileEntry{key: "a", data: []byte("aa"), size: 2})
	cache.put(memoryFileEntry{key: "b", data: []byte("b"), size: 1})
	if _, ok := cache.get("a"); !ok {
		t.Fatal("expected a in cache")
	}
	cache.put(memoryFileEntry{key: "c", data: []byte("c"), size: 1})
	if _, ok := cache.get("b"); ok {
		t.Fatal("least recently used entry b was not evicted")
	}
	if _, ok := cache.get("a"); !ok {
		t.Fatal("recently used entry a was evicted")
	}
	if _, ok := cache.get("c"); !ok {
		t.Fatal("new entry c was not inserted")
	}
}

func BenchmarkMemoryCacheGet(b *testing.B) {
	cache := newMemoryFileCache(1024*1024, 4096)
	cache.put(memoryFileEntry{key: "app\x00/app.js", data: []byte("content"), size: 7})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := cache.get("app\x00/app.js"); !ok {
			b.Fatal("memory cache entry disappeared")
		}
	}
}

func BenchmarkDiskCacheLookup(b *testing.B) {
	root := b.TempDir()
	source := filepath.Join(root, "app.js")
	if err := os.WriteFile(source, []byte("console.log('cached');"), 0644); err != nil {
		b.Fatal(err)
	}
	logger := zerolog.New(io.Discard)
	memoryEnable := false
	cfg := &config.Config{}
	cfg.PxyFrontend.Cache = frontproxy_config.FrontCacheConfig{
		Enable:                  true,
		Dir:                     filepath.Join(root, "cache"),
		Matcher:                 []string{"*.js"},
		MemoryEnable:            &memoryEnable,
		MemoryMaxMB:             1,
		MemoryMaxFileMB:         1,
		MemoryMaxEntries:        4,
		MemoryRevalidateSeconds: 60,
	}
	cm := NewCacheManager(cfg, &logger)
	cm.CacheFile("App", "/app.js", source)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, ok := cm.diskEntry("App", "/app.js", source); !ok {
			b.Fatal("disk cache entry disappeared")
		}
	}
}

func TestCacheFileKeepsMemoryWhenDiskWriteFails(t *testing.T) {
	cm, source := testCacheManager(t)
	badDir := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(badDir, []byte("file"), 0644); err != nil {
		t.Fatal(err)
	}
	cm.config.Cache.Dir = badDir
	cm.CacheFile("App", "/app.js", source)
	if _, ok := cm.memory.get(cm.cacheKey("App", "/app.js")); !ok {
		t.Fatal("disk write failure should not prevent memory population")
	}
}

func TestMemoryCacheExplicitDisable(t *testing.T) {
	cm, source := testCacheManager(t)
	enabled := false
	cm.config.Cache.MemoryEnable = &enabled
	cm.CacheFile("App", "/app.js", source)
	if _, ok := cm.memory.get(cm.cacheKey("App", "/app.js")); ok {
		t.Fatal("explicit memory_enable=false populated memory cache")
	}
	if _, err := os.Stat(cm.cachedFilePath("App", "/app.js")); err != nil {
		t.Fatalf("disk cache should remain enabled: %v", err)
	}
}
