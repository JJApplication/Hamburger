package connectprotocol

import (
	"Hamburger/internal/config"
	"testing"
)

func TestConfigStateRejectsInvalidUpdateWithoutChangingSnapshot(t *testing.T) {
	state, err := NewConfigState(config.ConnectProtocolConfig{
		Enabled:   true,
		BaseRoute: "/rpc/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := state.Load().BaseRoute; got != "/rpc" {
		t.Fatalf("initial route = %q, want /rpc", got)
	}
	if err := state.Store(config.ConnectProtocolConfig{Enabled: true, BaseRoute: "/"}); err == nil {
		t.Fatal("invalid route update unexpectedly succeeded")
	}
	got := state.Load()
	if !got.Enabled || got.BaseRoute != "/rpc" {
		t.Fatalf("snapshot changed after rejected update: %+v", got)
	}
}
