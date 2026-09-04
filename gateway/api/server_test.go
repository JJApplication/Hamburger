package api

import (
	"Hamburger/internal/config"
	"Hamburger/internal/config/svr_config"
	"io"
	"testing"

	"github.com/rs/zerolog"
)

func TestServerCanRestartAfterStop(t *testing.T) {
	cfg := &config.Config{ApiServerConfig: svr_config.ApiServerConfig{
		Enabled: true,
		Host:    "127.0.0.1",
		Port:    0,
	}}
	logger := zerolog.New(io.Discard)
	server := NewAPIServer(cfg, &logger)
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	first := server.server
	if first == nil {
		t.Fatal("server was not created on first start")
	}
	if err := server.Stop(); err != nil {
		t.Fatal(err)
	}
	if server.server != nil {
		t.Fatal("stopped server still retained its HTTP server")
	}
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	second := server.server
	if second == nil || second == first {
		t.Fatal("restart did not create a fresh HTTP server")
	}
	if err := server.Stop(); err != nil {
		t.Fatal(err)
	}
}
