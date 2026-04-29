package webdav

import (
	"Hamburger/internal/config"
	"Hamburger/internal/config/exp_config"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
)

func TestWebDAVIsolationByUser(t *testing.T) {
	baseDir := t.TempDir()
	aliceRoot := filepath.Join(baseDir, "alice")
	bobRoot := filepath.Join(baseDir, "bob")
	if err := os.MkdirAll(aliceRoot, 0o755); err != nil {
		t.Fatalf("mkdir alice root failed: %v", err)
	}
	if err := os.MkdirAll(bobRoot, 0o755); err != nil {
		t.Fatalf("mkdir bob root failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(aliceRoot, "shared.txt"), []byte("alice-file"), 0o644); err != nil {
		t.Fatalf("write alice file failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bobRoot, "shared.txt"), []byte("bob-file"), 0o644); err != nil {
		t.Fatalf("write bob file failed: %v", err)
	}

	s := newTestServer(exp_config.WebDAVConfig{
		Enabled: true,
		Users: []exp_config.WebDAVUserConfig{
			{Username: "alice", Password: "pass1", RootDir: aliceRoot},
			{Username: "bob", Password: "pass2", RootDir: bobRoot},
		},
	})
	handler := s.authHandler()

	aliceResp := performRequest(handler, http.MethodGet, "/shared.txt", "alice", "pass1", nil)
	if aliceResp.Code != http.StatusOK {
		t.Fatalf("alice read status = %d", aliceResp.Code)
	}
	if body := readBody(t, aliceResp); body != "alice-file" {
		t.Fatalf("alice read body = %q", body)
	}

	bobResp := performRequest(handler, http.MethodGet, "/shared.txt", "bob", "pass2", nil)
	if bobResp.Code != http.StatusOK {
		t.Fatalf("bob read status = %d", bobResp.Code)
	}
	if body := readBody(t, bobResp); body != "bob-file" {
		t.Fatalf("bob read body = %q", body)
	}
}

func TestWebDAVReadOnlyRejectWrite(t *testing.T) {
	baseDir := t.TempDir()
	root := filepath.Join(baseDir, "readonly")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root failed: %v", err)
	}

	s := newTestServer(exp_config.WebDAVConfig{
		Enabled:  true,
		ReadOnly: true,
		Users: []exp_config.WebDAVUserConfig{
			{Username: "alice", Password: "pass1", RootDir: root},
		},
	})
	handler := s.authHandler()

	resp := performRequest(handler, http.MethodPut, "/new.txt", "alice", "pass1", []byte("new-content"))
	if resp.Code < 400 {
		t.Fatalf("write status = %d", resp.Code)
	}
	if _, err := os.Stat(filepath.Join(root, "new.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected new.txt not created, err=%v", err)
	}
}

func TestWebDAVUnauthorized(t *testing.T) {
	baseDir := t.TempDir()
	root := filepath.Join(baseDir, "root")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root failed: %v", err)
	}

	s := newTestServer(exp_config.WebDAVConfig{
		Enabled: true,
		Users: []exp_config.WebDAVUserConfig{
			{Username: "alice", Password: "pass1", RootDir: root},
		},
	})
	handler := s.authHandler()

	resp := performRequest(handler, http.MethodGet, "/", "alice", "wrong", nil)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", resp.Code)
	}
}

func newTestServer(webdavCfg exp_config.WebDAVConfig) *WebDAVServer {
	cfg := &config.Config{
		ExpConfig: exp_config.ExpConfig{
			WebDAV: webdavCfg,
		},
	}
	logger := zerolog.Nop()
	return NewWebDAVServer(cfg, &logger)
}

func performRequest(handler http.Handler, method, path, user, pass string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.SetBasicAuth(user, pass)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func readBody(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	raw, err := io.ReadAll(rr.Body)
	if err != nil {
		t.Fatalf("read body failed: %v", err)
	}
	return string(raw)
}
