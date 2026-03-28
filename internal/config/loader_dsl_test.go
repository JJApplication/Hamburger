package config

import (
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
	if cfg.CustomHeader["ProxyServer"] == "" {
		t.Fatalf("unexpected custom header: %+v", cfg.CustomHeader)
	}
}
