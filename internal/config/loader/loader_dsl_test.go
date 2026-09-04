package loader

import (
	"Hamburger/internal/config"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDSL(t *testing.T) {
	t.Setenv("DSL_SERVER_PORT", "9443")
	content := `
{
  pxy_backend_file: "config/backend.hamburger",
  pxy_frontend_file: "config/frontend.hamburger",
  max_cores: 1<<3,
  custom_header: {
    ProxyServer: @AppName
  },
  api_server_config: {
    enabled: true,
    host: "127.0.0.1",
    port: $DSL_SERVER_PORT
  },
  connect_protocol: {
    enabled: true,
    base_route: "/rpc/",
    enable_bidi_stream: true
  }
}
`
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "config.hamburger")
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	cfg, err := LoadConfig(file)
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}
	if cfg.MaxCores != 8 {
		t.Fatalf("unexpected max_cores: %d", cfg.MaxCores)
	}
	if cfg.ApiServerConfig.Port != 9443 {
		t.Fatalf("unexpected api port: %d", cfg.ApiServerConfig.Port)
	}
	if !cfg.ConnectProtocol.Enabled || cfg.ConnectProtocol.BaseRoute != "/rpc/" || !cfg.ConnectProtocol.EnableBidiStream {
		t.Fatalf("unexpected ConnectProtocol config: %+v", cfg.ConnectProtocol)
	}
	if cfg.CustomHeader["ProxyServer"] == "" {
		t.Fatalf("unexpected custom header: %+v", cfg.CustomHeader)
	}
}

func TestLoadConfigDSLWithEnvFile(t *testing.T) {
	content := `
{
  api_server_config: {
    enabled: true,
    host: "127.0.0.1",
    port: $DSL_SERVER_PORT
  }
}
`
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "hamburger.env"), []byte("DSL_SERVER_PORT=9555\n"), 0644); err != nil {
		t.Fatalf("write env file failed: %v", err)
	}
	file := filepath.Join(tmpDir, "config.hamburger")
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("write config file failed: %v", err)
	}

	cfg, err := LoadConfig(file)
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}
	if cfg.ApiServerConfig.Port != 9555 {
		t.Fatalf("unexpected api port: %d", cfg.ApiServerConfig.Port)
	}
}

func TestLoadConfigJSONConnectProtocol(t *testing.T) {
	content := `{"connect_protocol":{"enabled":true,"base_route":"/json-rpc/","enable_bidi_stream":true}}`
	file := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	appCfg, err := LoadConfig(file)
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}
	cfg := Merge(appCfg)
	if !cfg.ConnectProtocol.Enabled || cfg.ConnectProtocol.BaseRoute != "/json-rpc/" || !cfg.ConnectProtocol.EnableBidiStream {
		t.Fatalf("unexpected merged ConnectProtocol config: %+v", cfg.ConnectProtocol)
	}
}

func TestMergeConnectProtocolDefaultRoute(t *testing.T) {
	cfg := Merge(&config.AppConfig{})
	if cfg.ConnectProtocol.BaseRoute != "/hamburger.service" {
		t.Fatalf("default merged ConnectProtocol base route = %q", cfg.ConnectProtocol.BaseRoute)
	}
}
