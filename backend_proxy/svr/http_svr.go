package svr

import (
	"Hamburger/internal/config"
	"Hamburger/internal/config/backproxy_config"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
)

// 传统的HTTP服务器

// 后端Server服务器

type Server struct {
	cfg         *config.Config
	logger      *zerolog.Logger
	host        string
	port        int
	backendConf backproxy_config.BackendServer
	svr         *http.Server
	started     bool
}

func NewHttpServer(cfg *config.Config, logger *zerolog.Logger, backendConf backproxy_config.BackendServer) *Server {
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
	for _, rs := range s.backendConf.Http.Response {
		mux.HandleFunc(rs.Path, func(w http.ResponseWriter, r *http.Request) {
			for k, v := range rs.Headers {
				w.Header().Set(k, v)
			}
			w.WriteHeader(rs.Code)
			w.Write([]byte(rs.Msg))
		})
	}

	if s.backendConf.Http.EnableStatic {
		staticDir := strings.TrimSpace(s.backendConf.Http.StaticDir)
		if staticDir != "" {
			fileServer := http.FileServer(http.Dir(staticDir))
			staticPrefix := strings.TrimSpace(s.backendConf.Http.StaticPrefix)
			if staticPrefix == "" {
				mux.Handle("/", fileServer)
			} else {
				if !strings.HasPrefix(staticPrefix, "/") {
					staticPrefix = "/" + staticPrefix
				}
				mux.Handle(staticPrefix, http.StripPrefix(staticPrefix, fileServer))
				mux.Handle(staticPrefix+"/", http.StripPrefix(staticPrefix, fileServer))
			}
		}
	}

	return s.auth(mux)
}

func (s *Server) auth(next http.Handler) http.Handler {
	user := s.backendConf.Http.User
	pass := s.backendConf.Http.Password
	if user == "" && pass == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != user || p != pass {
			w.Header().Set("WWW-Authenticate", "Basic realm=\"http\"")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
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
