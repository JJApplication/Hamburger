package manager

import (
	"Hamburger/internal/config"
	"Hamburger/internal/config/core_config"
	"net/http"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestStartRejectsNegativeMaxQuerySize(t *testing.T) {
	logger := zerolog.Nop()
	m := NewManager(&config.Config{
		Security: core_config.SecurityConfig{MaxQuerySize: -1},
	}, &logger, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	err := m.Start()
	if err == nil {
		t.Fatal("Start should reject a negative security.max_query_size")
	}
	if !strings.Contains(err.Error(), "security.max_query_size") {
		t.Fatalf("Start error = %q, want security.max_query_size context", err)
	}
}
