package connectprotocol

import (
	"Hamburger/internal/config"
	"sync/atomic"
)

// ConfigState stores the Connect facade settings as an atomic snapshot. The
// rest of the application keeps its historical in-place configuration model;
// Connect requests use this snapshot so a reload cannot expose a partially
// written route or stream setting.
type ConfigState struct {
	value atomic.Value // stores config.ConnectProtocolConfig
}

// NewConfigState validates and stores an initial Connect configuration.
func NewConfigState(initial config.ConnectProtocolConfig) (*ConfigState, error) {
	state := &ConfigState{}
	if err := state.Store(initial); err != nil {
		return nil, err
	}
	return state, nil
}

// Load returns the last successfully committed configuration snapshot.
func (s *ConfigState) Load() config.ConnectProtocolConfig {
	if s == nil {
		return config.ConnectProtocolConfig{}
	}
	value := s.value.Load()
	if value == nil {
		return config.ConnectProtocolConfig{}
	}
	return value.(config.ConnectProtocolConfig)
}

// Store validates and atomically commits a new configuration snapshot.
func (s *ConfigState) Store(next config.ConnectProtocolConfig) error {
	normalized, err := BaseRoute(next.BaseRoute)
	if err != nil {
		return err
	}
	next.BaseRoute = normalized
	s.value.Store(next)
	return nil
}
