package svr

import (
	"Hamburger/internal/config"
	"Hamburger/internal/config/backproxy_config"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"golang.org/x/net/webdav"
)

type WebDavServer struct {
	cfg         *config.Config
	logger      *zerolog.Logger
	host        string
	port        int
	backendConf backproxy_config.BackendServer
	svr         *http.Server
	started     bool
}

func NewWebDavServer(cfg *config.Config, logger *zerolog.Logger, backendConf backproxy_config.BackendServer) *WebDavServer {
	return &WebDavServer{
		cfg:         cfg,
		logger:      logger,
		host:        backendConf.Host,
		port:        backendConf.Port,
		backendConf: backendConf,
	}
}

func (s *WebDavServer) Start() {
	svr := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.host, s.port),
		Handler: s.GetHandler(),
	}
	s.svr = svr
	go func() {
		s.started = true
		if err := svr.ListenAndServe(); err != nil {
			s.logger.Error().Err(err).Msg("webdav server start error")
		}
	}()
}

func (s *WebDavServer) Stop() error {
	s.started = false
	return s.svr.Shutdown(context.Background())
}

func (s *WebDavServer) IsStarted() bool {
	return s.started
}

func (s *WebDavServer) Name() string {
	return s.backendConf.ServiceName
}

func (s *WebDavServer) GetAddr() string {
	return fmt.Sprintf("%s:%d", s.host, s.port)
}

func (s *WebDavServer) GetHandler() http.Handler {
	root := strings.TrimSpace(s.backendConf.WebDav.Root)
	if root == "" {
		root = "."
	} else {
		_ = os.MkdirAll(root, 0755)
	}

	handler := &webdav.Handler{
		Prefix:     "/",
		FileSystem: webdav.Dir(root),
		LockSystem: webdav.NewMemLS(),
	}
	return s.auth(handler)
}

func (s *WebDavServer) auth(next http.Handler) http.Handler {
	user := s.backendConf.WebDav.User
	pass := s.backendConf.WebDav.Password
	if user == "" && pass == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != user || p != pass {
			w.Header().Set("WWW-Authenticate", "Basic realm=\"webdav\"")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
