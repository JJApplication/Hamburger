package server

import (
	"Hamburger/internal/config/core_config"
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/net/http2"
)

const idleTestBodySize = 4 * 1024 * 1024

type idleTestReader struct {
	left  int
	delay time.Duration
}

func (s *idleTestReader) Read(p []byte) (int, error) {
	if s.left <= 0 {
		return 0, io.EOF
	}
	time.Sleep(s.delay)
	n := min(64*1024, len(p), s.left)
	for i := range p[:n] {
		p[i] = 'u'
	}
	s.left -= n
	return n, nil
}

type idleTestServer struct {
	url        string
	ctxErr     atomic.Value
	uploadSize atomic.Int64
}

// startIdleTestServer 用和生产一致的 CommonHttpServer(useNBIO=true, use_http2) 起服务，
// 读写超时都设成 1 秒，测试里的传输都持续 2 秒以上。
func startIdleTestServer(t *testing.T) *idleTestServer {
	t.Helper()
	s := &idleTestServer{}
	payload := bytes.Repeat([]byte("d"), idleTestBodySize)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := r.Context().Err(); err != nil {
				s.ctxErr.Store(err.Error())
			}
		}()
		switch r.URL.Path {
		case "/upload":
			n, err := io.Copy(io.Discard, r.Body)
			s.uploadSize.Store(n)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
		case "/download":
			w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
			for i := 0; i < len(payload); i += 64 * 1024 {
				if _, err := w.Write(payload[i : i+64*1024]); err != nil {
					return
				}
				http.NewResponseController(w).Flush()
				time.Sleep(35 * time.Millisecond)
			}
		case "/stall":
			time.Sleep(2500 * time.Millisecond)
			_, _ = w.Write([]byte("late"))
		}
	})

	log := zerolog.Nop()
	cfg := core_config.ServerConfig{
		Name: "idle-test", Host: "127.0.0.1", Port: 0, Protocol: "http", Enabled: true, UseHttp2: true,
		ReadTimeout: 1, WriteTimeout: 1, MaxRequestBody: 50 << 20,
	}
	inst, err := CommonHttpServer(cfg, &log, h, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	go inst.Server.Serve(inst.Listener)
	t.Cleanup(func() { _ = inst.Server.Close() })
	s.url = "http://" + inst.Listener.Addr().String()
	return s
}

func idleTestClients() map[string]*http.Client {
	return map[string]*http.Client{
		"http1": {Transport: &http.Transport{}},
		"h2c": {Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, addr)
			},
		}},
	}
}

func TestIdleDeadlineSlowUpload(t *testing.T) {
	for name, client := range idleTestClients() {
		t.Run(name, func(t *testing.T) {
			s := startIdleTestServer(t)
			req, _ := http.NewRequest(http.MethodPut, s.url+"/upload", &idleTestReader{left: idleTestBodySize, delay: 40 * time.Millisecond})
			req.ContentLength = idleTestBodySize
			start := time.Now()
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("upload failed after %s: %v", time.Since(start), err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("status = %d after %s", resp.StatusCode, time.Since(start))
			}
			if got := s.uploadSize.Load(); got != idleTestBodySize {
				t.Fatalf("server received %d of %d bytes", got, idleTestBodySize)
			}
			if elapsed := time.Since(start); elapsed < 2*time.Second {
				t.Fatalf("upload took %s; it must outlast the 1s timeout", elapsed)
			}
		})
	}
}

func TestIdleDeadlineSlowDownload(t *testing.T) {
	for name, client := range idleTestClients() {
		t.Run(name, func(t *testing.T) {
			s := startIdleTestServer(t)
			start := time.Now()
			resp, err := client.Get(s.url + "/download")
			if err != nil {
				t.Fatal(err)
			}
			n, err := io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if err != nil || n != idleTestBodySize {
				t.Fatalf("got %d of %d bytes after %s: %v", n, idleTestBodySize, time.Since(start), err)
			}
			if elapsed := time.Since(start); elapsed < 2*time.Second {
				t.Fatalf("download took %s; it must outlast the 1s timeout", elapsed)
			}
			time.Sleep(50 * time.Millisecond)
			if v := s.ctxErr.Load(); v != nil {
				t.Fatalf("request context canceled during active download: %v", v)
			}
		})
	}
}

func TestIdleDeadlineStillCutsSilentRequests(t *testing.T) {
	s := startIdleTestServer(t)
	client := &http.Client{Transport: &http.Transport{}}
	start := time.Now()
	resp, err := client.Get(s.url + "/stall")
	if err == nil {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(body) == "late" {
			t.Fatalf("silent request survived %s past the 1s idle timeout", time.Since(start))
		}
	}
	time.Sleep(200 * time.Millisecond)
	if s.ctxErr.Load() == nil {
		t.Fatal("silent request context was not canceled by the idle timeout")
	}
}
