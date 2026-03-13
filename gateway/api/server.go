package api

import (
	"Hamburger/gateway/api/middleware"
	"Hamburger/gateway/api/route"
	"Hamburger/gateway/api/service"
	"Hamburger/internal/config"
	"Hamburger/internal/utils"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"golang.org/x/net/http2"
)

type Server struct {
	enabled bool
	addr    string
	port    int

	logger  *zerolog.Logger
	engine  *gin.Engine
	server  *http.Server
	service *service.APIService
	jwtCfg  config.JWTConfig
	http2   config.APIHTTP2Config
}

func NewAPIServer(cfg *config.Config, logger *zerolog.Logger) *Server {
	apiCfg := cfg.ApiServerConfig
	if !apiCfg.Enabled {
		return new(Server)
	}
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	s := &Server{
		enabled: apiCfg.Enabled,
		addr:    apiCfg.Host,
		port:    apiCfg.Port,
		logger:  logger,
		engine:  engine,
		service: service.NewAPIService(apiCfg),
		jwtCfg:  apiCfg.JWT,
		http2:   apiCfg.HTTP2,
	}
	middleware.Register(s.engine)
	route.Register(s.engine, s.service, middleware.JWT(s.jwtCfg))
	s.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", apiCfg.Host, apiCfg.Port),
		Handler: engine,
	}
	return s
}

func (s *Server) Start() error {
	if !s.enabled || s.server == nil {
		return nil
	}
	s.logger.Info().Str("address", s.addr).Int("port", s.port).Msg("start api server")
	if s.http2.Enabled && s.http2.Insecure {
		return s.startInsecureHTTP2()
	}
	go func() {
		err := s.server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error().Err(err).Msg("api server listen err")
		}
	}()
	return nil
}

func (s *Server) Stop() error {
	if !s.enabled || s.server == nil {
		return nil
	}
	if err := s.service.CloseDB(); err != nil {
		return err
	}
	return s.server.Shutdown(context.Background())
}

func (s *Server) startInsecureHTTP2() error {
	proto := &http.Protocols{}
	proto.SetHTTP1(true)
	proto.SetHTTP2(true)
	proto.SetUnencryptedHTTP2(true)
	h2 := s.http2
	httpServer := &http.Server{
		Addr:              s.server.Addr,
		Handler:           s.server.Handler,
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
		return err
	}
	s.server = httpServer
	go func() {
		err := s.server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error().Err(err).Msg("api server http2 listen err")
		}
	}()
	return nil
}
