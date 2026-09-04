package api

import (
	"Hamburger/gateway/api/middleware"
	"Hamburger/gateway/api/route"
	"Hamburger/gateway/api/service"
	"Hamburger/internal/config"
	"Hamburger/internal/config/loader"
	"Hamburger/internal/config/svr_config"
	"Hamburger/internal/utils"
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"golang.org/x/net/http2"
)

type Server struct {
	mu  sync.Mutex
	cfg *config.Config

	enabled bool
	addr    string
	port    int

	logger  *zerolog.Logger
	engine  *gin.Engine
	server  *http.Server
	service *service.APIService
	jwtCfg  svr_config.JWTConfig
	http2   svr_config.APIHTTP2Config
}

// NewAPIServer constructs the standalone API listener. A shared service may
// be supplied by the initializer so the gateway Connect handler and this
// listener use the same store and business logic.
func NewAPIServer(cfg *config.Config, logger *zerolog.Logger, shared ...*service.APIService) *Server {
	apiCfg := cfg.ApiServerConfig
	var apiService *service.APIService
	if len(shared) > 0 {
		apiService = shared[0]
	}
	if apiService == nil && apiCfg.Enabled {
		apiService = service.NewAPIService(apiCfg)
	}
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	s := &Server{
		cfg:     cfg,
		enabled: apiCfg.Enabled,
		addr:    apiCfg.Host,
		port:    apiCfg.Port,
		logger:  logger,
		engine:  engine,
		service: apiService,
		jwtCfg:  apiCfg.JWT,
		http2:   apiCfg.HTTP2,
	}
	return s
}

func (s *Server) Start() error {
	s.mu.Lock()
	if cfg := loader.SnapshotOf(s.cfg); cfg != nil {
		s.applyConfigLocked(cfg.ApiServerConfig)
	}
	if !s.enabled {
		s.mu.Unlock()
		return nil
	}
	if s.server != nil {
		s.mu.Unlock()
		return nil
	}
	server, err := s.newHTTPServerLocked()
	if err != nil {
		s.mu.Unlock()
		return err
	}
	s.server = server
	addr, port := s.addr, s.port
	s.mu.Unlock()

	if s.logger != nil {
		s.logger.Info().Str("address", addr).Int("port", port).Msg("start api server")
	}
	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			if s.logger != nil {
				s.logger.Error().Err(err).Msg("api server listen err")
			}
		}
	}()
	return nil
}

func (s *Server) Stop() error {
	s.mu.Lock()
	server := s.server
	s.server = nil
	enabled := s.enabled
	s.mu.Unlock()
	if !enabled || server == nil {
		return nil
	}
	return server.Shutdown(context.Background())
}

func (s *Server) SetServerControl(stopFn map[string]func() error, restartFn map[string]func() error) {
	if s == nil || s.service == nil {
		return
	}
	s.service.SetServerControl(stopFn, restartFn)
}

func (s *Server) applyConfigLocked(apiCfg svr_config.ApiServerConfig) {
	s.enabled = apiCfg.Enabled
	s.addr = apiCfg.Host
	s.port = apiCfg.Port
	s.jwtCfg = apiCfg.JWT
	s.http2 = apiCfg.HTTP2
}

func (s *Server) newHTTPServerLocked() (*http.Server, error) {
	engine := gin.New()
	middleware.Register(engine)
	route.Register(engine, s.service, middleware.JWT(s.jwtCfg))
	s.engine = engine

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.addr, s.port),
		Handler: engine,
	}
	if s.http2.Enabled && s.http2.Insecure {
		return s.newInsecureHTTP2ServerLocked(server)
	}
	return server, nil
}

func (s *Server) newInsecureHTTP2ServerLocked(server *http.Server) (*http.Server, error) {
	proto := &http.Protocols{}
	proto.SetHTTP1(true)
	proto.SetHTTP2(true)
	proto.SetUnencryptedHTTP2(true)
	h2 := s.http2
	httpServer := &http.Server{
		Addr:              server.Addr,
		Handler:           server.Handler,
		ReadTimeout:       time.Second * time.Duration(utils.DefaultInt64(h2.ReadTimeout, 30)),
		WriteTimeout:      time.Second * time.Duration(utils.DefaultInt64(h2.WriteTimeout, 30)),
		IdleTimeout:       time.Second * time.Duration(utils.DefaultInt64(h2.IdleTimeout, 60)),
		ReadHeaderTimeout: time.Second * time.Duration(utils.DefaultInt64(h2.ReadHeaderTimeout, 10)),
		MaxHeaderBytes:    int(utils.DefaultInt64(h2.MaxHeaderBytes, 5<<20)),
		Protocols:         proto,
	}
	h2Server := &http2.Server{}
	if h2.MaxHandlers > 0 {
		h2Server.MaxHandlers = h2.MaxHandlers
	}
	if h2.MaxConcurrentStreams > 0 {
		h2Server.MaxConcurrentStreams = uint32(h2.MaxConcurrentStreams)
	}
	if err := http2.ConfigureServer(httpServer, h2Server); err != nil {
		return nil, err
	}
	return httpServer, nil
}
