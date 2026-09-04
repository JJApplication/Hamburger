package config

import "testing"

func TestDefaultConnectProtocolConfig(t *testing.T) {
	cfg := GetDefaultConfig()
	if cfg.ConnectProtocol.Enabled {
		t.Fatal("ConnectProtocol should be disabled by default")
	}
	if cfg.ConnectProtocol.BaseRoute != "/hamburger.service" {
		t.Fatalf("default ConnectProtocol base route = %q", cfg.ConnectProtocol.BaseRoute)
	}
	if cfg.ConnectProtocol.EnableBidiStream {
		t.Fatal("bidirectional streams should be disabled by default")
	}
}
