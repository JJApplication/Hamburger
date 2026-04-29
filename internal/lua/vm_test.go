package lua

import (
	"Hamburger/internal/config/svr_config"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestManagerInitInjectGlobals(t *testing.T) {
	t.Setenv("HAMBURGER_LUA_ENV", "from-test")
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "base.lua"), []byte(`
log.Info("lua loaded")
function request_handle(req)
  return nil
end
`), 0644); err != nil {
		t.Fatalf("write lua file failed: %v", err)
	}

	m := &manager{}
	if err := m.init(svr_config.LuaConfig{
		Enabled:     true,
		ScriptsRoot: root,
	}, nil); err != nil {
		t.Fatalf("init lua vm failed: %v", err)
	}
	if m.state == nil {
		t.Fatal("lua state should not be nil")
	}
	if got := m.state.GetGlobal("log"); got.Type().String() != "table" {
		t.Fatalf("unexpected log type: %s", got.Type().String())
	}
	if got := m.state.GetGlobal("hamburger"); got.Type().String() != "table" {
		t.Fatalf("unexpected hamburger type: %s", got.Type().String())
	}
	if got := m.state.GetGlobal("constant"); got.Type().String() != "table" {
		t.Fatalf("unexpected constant type: %s", got.Type().String())
	}
	envValue := m.state.GetGlobal("env").(*lua.LTable).RawGetString("HAMBURGER_LUA_ENV").String()
	if envValue != "from-test" {
		t.Fatalf("unexpected env value: %s", envValue)
	}
	if len(m.middlewares) != 1 {
		t.Fatalf("unexpected middleware count: %d", len(m.middlewares))
	}
}

func TestHandleRequestApplyResult(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "handler.lua"), []byte(`
function request_handle(req)
  return {
    host = "lua.example.com",
    path = "/lua-path",
    raw_query = "from=lua",
    header_set = {["X-Lua"] = "ok"},
    header_add = {["X-Trace"] = "lua"},
    header_del = {"X-Delete"},
  }
end
`), 0644); err != nil {
		t.Fatalf("write lua file failed: %v", err)
	}

	m := &manager{}
	if err := m.init(svr_config.LuaConfig{
		Enabled:     true,
		ScriptsRoot: root,
	}, nil); err != nil {
		t.Fatalf("init lua vm failed: %v", err)
	}

	request := &http.Request{
		Method: "GET",
		Host:   "origin.example.com",
		Header: http.Header{
			"X-Delete": []string{"remove-me"},
		},
		URL: &url.URL{
			Path: "/origin",
		},
	}
	if err := m.handleRequest(request); err != nil {
		t.Fatalf("handle request failed: %v", err)
	}
	if request.Host != "lua.example.com" {
		t.Fatalf("unexpected host: %s", request.Host)
	}
	if request.URL.Path != "/lua-path" {
		t.Fatalf("unexpected path: %s", request.URL.Path)
	}
	if request.URL.RawQuery != "from=lua" {
		t.Fatalf("unexpected raw query: %s", request.URL.RawQuery)
	}
	if request.Header.Get("X-Lua") != "ok" {
		t.Fatalf("unexpected X-Lua header: %s", request.Header.Get("X-Lua"))
	}
	if request.Header.Get("X-Trace") != "lua" {
		t.Fatalf("unexpected X-Trace header: %s", request.Header.Get("X-Trace"))
	}
	if request.Header.Get("X-Delete") != "" {
		t.Fatalf("X-Delete header should be removed: %s", request.Header.Get("X-Delete"))
	}
}

func TestHandleRequestReject(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "reject.lua"), []byte(`
function request_handle(req)
  return {
    allow = false,
    error = "blocked by lua",
  }
end
`), 0644); err != nil {
		t.Fatalf("write lua file failed: %v", err)
	}

	m := &manager{}
	if err := m.init(svr_config.LuaConfig{
		Enabled:     true,
		ScriptsRoot: root,
	}, nil); err != nil {
		t.Fatalf("init lua vm failed: %v", err)
	}

	request := &http.Request{
		Method: "GET",
		Host:   "origin.example.com",
		Header: http.Header{},
		URL:    &url.URL{Path: "/origin"},
	}
	err := m.handleRequest(request)
	if err == nil {
		t.Fatal("expected reject error")
	}
	if err.Error() != "blocked by lua" {
		t.Fatalf("unexpected error: %v", err)
	}
}
