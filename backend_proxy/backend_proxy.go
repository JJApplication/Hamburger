package backend_proxy

import (
	"Hamburger/internal/config"
	"github.com/rs/zerolog"
)

type BackendProxy struct {
	cfg     *config.Config
	logger  *zerolog.Logger
	servers map[string]*Server
}

func NewBackendProxy(cfg *config.Config, logger *zerolog.Logger) *BackendProxy {
	if !cfg.PxyBackend.Enabled {
		return &BackendProxy{
			cfg:    cfg,
			logger: logger,
		}
	}
	bp := BackendProxy{
		cfg:     cfg,
		logger:  logger,
		servers: make(map[string]*Server),
	}
	for _, server := range cfg.PxyBackend.Servers {
		svr := NewBackendServer(cfg, logger, server)
		bp.servers[server.ServiceName] = svr
	}

	return &bp
}

func (bp *BackendProxy) Start() {
	for _, svr := range bp.servers {
		svr.Start()
	}
}

func (bp *BackendProxy) Stop() {
	for _, svr := range bp.servers {
		if err := svr.Stop(); err != nil {
			bp.logger.Error().Err(err).Str("service", svr.Name()).Msg("failed to stop server")
		}
	}
}

func (bp *BackendProxy) Status() {
	if len(bp.servers) == 0 {
		bp.logger.Info().Msg("backend proxy no servers available")
		return
	}
	for _, svr := range bp.servers {
		bp.logger.Info().Str("service", svr.Name()).Bool("running", svr.IsStarted()).Msg("backend proxy status")
	}
}
