package core

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

const e2eAPKSize = 19*1024*1024 + 12345

type e2eEnv struct {
	data      []byte
	gateway   *httptest.Server
	origin    *httptest.Server
	served    atomic.Int64
	slowAfter atomic.Int64
	slowDelay atomic.Int64

	stallBeforeReply atomic.Int64
}

func newE2EEnv(t *testing.T) *e2eEnv {
	return newE2EEnvWithIdle(t, defaultUpstreamIdleTimeout)
}

func newE2EEnvWithIdle(t *testing.T, idle time.Duration) *e2eEnv {
	t.Helper()
	env := &e2eEnv{data: make([]byte, e2eAPKSize)}
	if _, err := rand.Read(env.data); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "app.apk")
	if err := os.WriteFile(path, env.data, 0o644); err != nil {
		t.Fatal(err)
	}

	env.origin = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			got, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if stall := env.stallBeforeReply.Load(); stall > 0 {
				time.Sleep(time.Duration(stall))
			}
			sum := sha256.Sum256(got)
			w.Header().Set("X-Upload-Size", strconv.Itoa(len(got)))
			w.Header().Set("X-Upload-Sha256", fmt.Sprintf("%x", sum))
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.android.package-archive")
		http.ServeFile(&countingWriter{ResponseWriter: w, env: env}, r, path)
	}))
	t.Cleanup(env.origin.Close)

	target, _ := url.Parse(env.origin.URL)
	proxy := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.URL.Scheme = target.Scheme
			r.URL.Host = target.Host
		},
		Transport:     newFastRoundTripper(idle),
		FlushInterval: 100 * time.Millisecond,
		ErrorLog:      nil,
	}
	env.gateway = httptest.NewServer(proxy)
	t.Cleanup(env.gateway.Close)
	return env
}

type countingWriter struct {
	http.ResponseWriter
	env *e2eEnv
}

func (w *countingWriter) Write(p []byte) (int, error) {
	if after := w.env.slowAfter.Load(); after > 0 && w.env.served.Load() >= after {
		time.Sleep(time.Duration(w.env.slowDelay.Load()))
	}
	n, err := w.ResponseWriter.Write(p)
	w.env.served.Add(int64(n))
	return n, err
}

func (w *countingWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func TestE2EFullDownloadMatchesFile(t *testing.T) {
	env := newE2EEnv(t)
	resp, err := http.Get(env.gateway.URL + "/app.apk")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if resp.ContentLength != int64(len(env.data)) {
		t.Fatalf("Content-Length = %d, want %d", resp.ContentLength, len(env.data))
	}
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(got) != sha256.Sum256(env.data) {
		t.Fatalf("downloaded %d bytes, hash mismatch", len(got))
	}
}

func TestE2EPauseResumeNeverExceedsFileSize(t *testing.T) {
	env := newE2EEnv(t)
	client := &http.Client{}

	// 第一段：下到约 7MB 时“暂停”（主动断开）。
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, env.gateway.URL+"/app.apk", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	part1 := make([]byte, 7*1024*1024)
	if _, err := io.ReadFull(resp.Body, part1); err != nil {
		t.Fatal(err)
	}
	cancel()
	resp.Body.Close()

	// 续传：Range 从已下载位置开始。
	req2, _ := http.NewRequest(http.MethodGet, env.gateway.URL+"/app.apk", nil)
	req2.Header.Set("Range", fmt.Sprintf("bytes=%d-", len(part1)))
	req2.Header.Set("Accept-Encoding", "gzip, zstd")
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusPartialContent {
		t.Fatalf("resume status = %d, want 206", resp2.StatusCode)
	}
	wantRange := fmt.Sprintf("bytes %d-%d/%d", len(part1), len(env.data)-1, len(env.data))
	if got := resp2.Header.Get("Content-Range"); got != wantRange {
		t.Fatalf("Content-Range = %q, want %q", got, wantRange)
	}
	if enc := resp2.Header.Get("Content-Encoding"); enc != "" {
		t.Fatalf("resume response must not be compressed, got %q", enc)
	}
	part2, err := io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatal(err)
	}
	total := len(part1) + len(part2)
	if total != len(env.data) {
		t.Fatalf("resumed total = %d, file size = %d", total, len(env.data))
	}
	if !bytes.Equal(append(part1, part2...), env.data) {
		t.Fatal("resumed file content mismatch")
	}
}

func TestE2EStopDownloadReleasesUpstream(t *testing.T) {
	env := newE2EEnv(t)
	// 上游在发出 2MB 后变慢，模拟慢链路；客户端停止后上游写入必须很快报错停止。
	env.slowAfter.Store(2 * 1024 * 1024)
	env.slowDelay.Store(int64(20 * time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, env.gateway.URL+"/app.apk", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 1024*1024)
	if _, err := io.ReadFull(resp.Body, buf); err != nil {
		t.Fatal(err)
	}
	cancel()
	resp.Body.Close()

	time.Sleep(500 * time.Millisecond)
	before := env.served.Load()
	time.Sleep(1 * time.Second)
	after := env.served.Load()
	if after != before {
		t.Fatalf("upstream still sending after stop: %d -> %d bytes", before, after)
	}
	if after >= int64(len(env.data)) {
		t.Fatalf("upstream sent whole file (%d) despite stop", after)
	}
}

const e2eIdle = 300 * time.Millisecond

func TestE2EKeepAliveSlowDownloadOutlivesIdleTimeout(t *testing.T) {
	env := newE2EEnvWithIdle(t, e2eIdle)
	client := &http.Client{}

	// 第一次请求让网关到上游的连接进入连接池。
	warm, err := client.Get(env.gateway.URL + "/app.apk")
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, warm.Body)
	warm.Body.Close()

	// 第二次复用连接，正文总耗时远超空闲超时，但每块之间的间隔都小于它。
	env.served.Store(0)
	env.slowAfter.Store(1)
	env.slowDelay.Store(int64(8 * time.Millisecond))
	req, _ := http.NewRequest(http.MethodGet, env.gateway.URL+"/app.apk", nil)
	req.Header.Set("Range", "bytes=0-"+strconv.Itoa(4*1024*1024-1))
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("slow body failed after %s: %v", time.Since(start), err)
	}
	if elapsed := time.Since(start); elapsed < 3*e2eIdle {
		t.Fatalf("body finished in %s; test did not exceed the idle timeout", elapsed)
	}
	if !bytes.Equal(got, env.data[:4*1024*1024]) {
		t.Fatalf("got %d bytes, content mismatch", len(got))
	}
}

// throttledReader 每 delay 交出一块数据，模拟慢速上传客户端。
type throttledReader struct {
	data  []byte
	pos   int
	chunk int
	delay time.Duration
}

func (s *throttledReader) Read(p []byte) (int, error) {
	if s.pos >= len(s.data) {
		return 0, io.EOF
	}
	time.Sleep(s.delay)
	n := min(s.chunk, len(p), len(s.data)-s.pos)
	copy(p, s.data[s.pos:s.pos+n])
	s.pos += n
	return n, nil
}

func TestE2ESlowUploadOutlivesIdleTimeout(t *testing.T) {
	env := newE2EEnvWithIdle(t, e2eIdle)
	payload := env.data[:3*1024*1024]
	wantSum := fmt.Sprintf("%x", sha256.Sum256(payload))

	for _, chunked := range []bool{false, true} {
		body := &throttledReader{data: payload, chunk: 64 * 1024, delay: 30 * time.Millisecond}
		req, _ := http.NewRequest(http.MethodPut, env.gateway.URL+"/upload.apk", body)
		if !chunked {
			req.ContentLength = int64(len(payload))
		}
		start := time.Now()
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("chunked=%v: %v", chunked, err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		elapsed := time.Since(start)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("chunked=%v: status %d after %s", chunked, resp.StatusCode, elapsed)
		}
		if elapsed < 3*e2eIdle {
			t.Fatalf("chunked=%v: upload finished in %s; test did not exceed the idle timeout", chunked, elapsed)
		}
		if got := resp.Header.Get("X-Upload-Sha256"); got != wantSum {
			t.Fatalf("chunked=%v: upstream received size=%s hash mismatch", chunked, resp.Header.Get("X-Upload-Size"))
		}
	}
}

func TestE2EUpstreamSilenceStillTimesOut(t *testing.T) {
	env := newE2EEnvWithIdle(t, e2eIdle)
	env.stallBeforeReply.Store(int64(2 * time.Second))

	req, _ := http.NewRequest(http.MethodPut, env.gateway.URL+"/upload.apk", bytes.NewReader(env.data[:1024]))
	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	elapsed := time.Since(start)
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502 when upstream is silent", resp.StatusCode)
	}
	if elapsed > time.Second {
		t.Fatalf("idle timeout fired after %s, want about %s", elapsed, e2eIdle)
	}
}
