package backend_proxy

import (
	"Hamburger/backend_proxy/svr"
	"Hamburger/internal/config"
	"strings"

	"github.com/rs/zerolog"
)

const (
	ServerTypeWebDav      = "webdav"
	ServerTypeWeb         = "web"
	ServerTypeTransparent = "transparent"
	ServerTypeTCP         = "tcp"
)

type BackendProxy struct {
	cfg     *config.Config
	logger  *zerolog.Logger
	servers map[string]svr.BackendSvr
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
		servers: make(map[string]svr.BackendSvr),
	}
	for _, server := range cfg.PxyBackend.Servers {
		serverType := strings.TrimSpace(strings.ToLower(server.Type))
		var bsvr svr.BackendSvr
		switch serverType {
		case ServerTypeWebDav:
			bsvr = svr.NewWebDavServer(cfg, logger, server)
		case ServerTypeTransparent:
			bsvr = svr.NewTransparentServer(cfg, logger, server)
		case ServerTypeTCP:
			bsvr = svr.NewTcpProxyServer(cfg, logger, server)
		default:
			bsvr = svr.NewHttpServer(cfg, logger, server)
		}
		bp.servers[server.ServiceName] = bsvr
	}

	return &bp
}

func (bp *BackendProxy) Start() {
	for _, s := range bp.servers {
		s.Start()
	}
}

func (bp *BackendProxy) Stop() {
	for _, s := range bp.servers {
		if err := s.Stop(); err != nil {
			bp.logger.Error().Err(err).Str("service", s.Name()).Msg("failed to stop server")
		}
	}
}

func (bp *BackendProxy) Status() {
	if len(bp.servers) == 0 {
		bp.logger.Info().Msg("backend proxy no servers available")
		return
	}
	for _, s := range bp.servers {
		bp.logger.Info().Str("service", s.Name()).Bool("running", s.IsStarted()).Msg("backend proxy status")
	}
}
