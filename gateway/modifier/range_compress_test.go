package modifier

import (
	"Hamburger/internal/config"
	"Hamburger/internal/config/loader"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func TestGzipSkipsPartialAndRangeResponses(t *testing.T) {
	modifier := newTestGzipModifier(t)
	body := bytes.Repeat([]byte("apk-bytes-"), 400)

	partial := newCompressibleResponse(t, http.StatusPartialContent, body, "bytes=0-3999")
	partial.Header.Set("Content-Range", "bytes 0-3999/20000")
	if err := modifier.ModifyResponse(partial); err != nil {
		t.Fatal(err)
	}
	assertUncompressed(t, partial, body, "bytes 0-3999/20000")

	ranged := newCompressibleResponse(t, http.StatusOK, body, "bytes=0-100")
	if err := modifier.ModifyResponse(ranged); err != nil {
		t.Fatal(err)
	}
	assertUncompressed(t, ranged, body, "")
}

func TestGzipStillCompressesFullResponse(t *testing.T) {
	modifier := newTestGzipModifier(t)
	body := bytes.Repeat([]byte("apk-bytes-"), 400)
	resp := newCompressibleResponse(t, http.StatusOK, body, "")

	if err := modifier.ModifyResponse(resp); err != nil {
		t.Fatal(err)
	}
	if resp.Header.Get("Content-Encoding") != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", resp.Header.Get("Content-Encoding"))
	}
	if resp.ContentLength != int64(lenHeader(resp, "Content-Length")) {
		t.Fatalf("ContentLength = %d, header = %s", resp.ContentLength, resp.Header.Get("Content-Length"))
	}
	reader, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("decoded length = %d, want %d", len(got), len(body))
	}
}

func TestZstdSkipsPartialResponse(t *testing.T) {
	modifier := newTestZstdModifier(t)
	body := bytes.Repeat([]byte("apk-bytes-"), 400)
	resp := newCompressibleResponse(t, http.StatusPartialContent, body, "bytes=0-3999")
	resp.Header.Set("Accept-Encoding", "")
	resp.Request.Header.Set("Accept-Encoding", "zstd")
	resp.Header.Set("Content-Range", "bytes 0-3999/20000")

	if err := modifier.ModifyResponse(resp); err != nil {
		t.Fatal(err)
	}
	assertUncompressed(t, resp, body, "bytes 0-3999/20000")
	if resp.Header.Get("Content-Encoding") == "zstd" {
		t.Fatal("partial response was zstd compressed")
	}
}

func TestZstdStillCompressesFullResponse(t *testing.T) {
	modifier := newTestZstdModifier(t)
	body := bytes.Repeat([]byte("apk-bytes-"), 400)
	resp := newCompressibleResponse(t, http.StatusOK, body, "")
	resp.Request.Header.Set("Accept-Encoding", "zstd")

	if err := modifier.ModifyResponse(resp); err != nil {
		t.Fatal(err)
	}
	if resp.Header.Get("Content-Encoding") != "zstd" {
		t.Fatalf("Content-Encoding = %q, want zstd", resp.Header.Get("Content-Encoding"))
	}
	if resp.ContentLength <= 0 || resp.Header.Get("Content-Length") != strconv.FormatInt(resp.ContentLength, 10) {
		t.Fatalf("ContentLength = %d, header = %s", resp.ContentLength, resp.Header.Get("Content-Length"))
	}
	decoder, err := zstd.NewReader(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer decoder.Close()
	got, err := io.ReadAll(decoder)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("decoded length = %d, want %d", len(got), len(body))
	}
}

func newTestGzipModifier(t *testing.T) *GzipModifier {
	t.Helper()
	cfg := config.GetDefaultConfig()
	cfg.Middleware.Gzip.Enabled = true
	cfg.Middleware.Gzip.Level = gzip.BestSpeed
	cfg.Middleware.Gzip.Types = []string{"text/plain"}
	cfg.Middleware.Gzip.Threshold = 32
	cfg.Middleware.Zstd.Enabled = false
	loader.Set(loader.Merge(cfg))
	return NewGzipModifier()
}

func newTestZstdModifier(t *testing.T) *ZstdModifier {
	t.Helper()
	cfg := config.GetDefaultConfig()
	cfg.Middleware.Gzip.Enabled = false
	cfg.Middleware.Zstd.Enabled = true
	cfg.Middleware.Zstd.Level = 1
	cfg.Middleware.Zstd.Types = []string{"text/plain"}
	cfg.Middleware.Zstd.Threshold = 32
	loader.Set(loader.Merge(cfg))
	return NewZstdModifier()
}

func newCompressibleResponse(t *testing.T, status int, body []byte, rangeHeader string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, "http://pkg.example/app.apk", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "gzip")
	if rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}
	return &http.Response{
		StatusCode:    status,
		Header:        http.Header{"Content-Type": []string{"text/plain"}, "Content-Length": []string{strconv.Itoa(len(body))}},
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       req,
	}
}

func assertUncompressed(t *testing.T, resp *http.Response, want []byte, contentRange string) {
	t.Helper()
	if resp.Header.Get("Content-Encoding") != "" {
		t.Fatalf("Content-Encoding = %q, want empty", resp.Header.Get("Content-Encoding"))
	}
	if resp.Header.Get("Content-Range") != contentRange {
		t.Fatalf("Content-Range = %q, want %q", resp.Header.Get("Content-Range"), contentRange)
	}
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("body length = %d, want %d", len(got), len(want))
	}
}

func lenHeader(resp *http.Response, key string) int {
	n, _ := strconv.Atoi(resp.Header.Get(key))
	return n
}
