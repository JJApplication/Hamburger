package backend_proxy

import (
	"Hamburger/internal/config"
	"context"
	"fmt"
	"github.com/rs/zerolog"
	"net/http"
)

// 后端Server服务器

type Server struct {
	cfg         *config.Config
	logger      *zerolog.Logger
	host        string
	port        int
	backendConf config.BackendServer
	svr         *http.Server
	started     bool
}

func NewBackendServer(cfg *config.Config, logger *zerolog.Logger, backendConf config.BackendServer) *Server {
	return &Server{
		cfg:         cfg,
		logger:      logger,
		host:        backendConf.Host,
		port:        backendConf.Port,
		backendConf: backendConf,
	}
}

// GetHandler 处理后端配置生成handler
func (s *Server) GetHandler() http.Handler {
	mux := http.NewServeMux()
	for _, rs := range s.backendConf.Response {
		mux.HandleFunc(rs.Path, func(w http.ResponseWriter, r *http.Request) {
			for k, v := range rs.Headers {
				w.Header().Set(k, v)
			}
			w.WriteHeader(rs.Code)
			w.Write([]byte(rs.Msg))
		})
	}

	return mux
}

func (s *Server) Start() {
	svr := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.host, s.port),
		Handler: s.GetHandler(),
	}
	s.svr = svr
	go func() {
		s.started = true
		if err := svr.ListenAndServe(); err != nil {
			s.logger.Error().Err(err).Msg("backend server start error")
		}
	}()
}

func (s *Server) Stop() error {
	s.started = false
	return s.svr.Shutdown(context.Background())
}

func (s *Server) IsStarted() bool {
	return s.started
}

func (s *Server) GetAddr() string {
	return fmt.Sprintf("%s:%d", s.host, s.port)
}

func (s *Server) Name() string {
	return s.backendConf.ServiceName
}
