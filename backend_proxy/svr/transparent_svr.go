package svr

import (
	"Hamburger/internal/config"
	"Hamburger/internal/config/backproxy_config"
	"context"
	"fmt"
	"github.com/rs/zerolog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type TransparentServer struct {
	cfg         *config.Config
	logger      *zerolog.Logger
	host        string
	port        int
	backendConf backproxy_config.BackendServer
	svr         *http.Server
	started     bool
}

func NewTransparentServer(cfg *config.Config, logger *zerolog.Logger, backendConf backproxy_config.BackendServer) *TransparentServer {
	return &TransparentServer{
		cfg:         cfg,
		logger:      logger,
		host:        backendConf.Host,
		port:        backendConf.Port,
		backendConf: backendConf,
	}
}

func (s *TransparentServer) Start() {
	svr := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.host, s.port),
		Handler: s.GetHandler(),
	}
	s.svr = svr
	go func() {
		s.started = true
		if err := svr.ListenAndServe(); err != nil {
			s.logger.Error().Err(err).Msg("transparent server start error")
		}
	}()
}

func (s *TransparentServer) Stop() error {
	s.started = false
	return s.svr.Shutdown(context.Background())
}

func (s *TransparentServer) IsStarted() bool {
	return s.started
}

func (s *TransparentServer) Name() string {
	return s.backendConf.ServiceName
}

func (s *TransparentServer) GetAddr() string {
	return fmt.Sprintf("%s:%d", s.host, s.port)
}

func (s *TransparentServer) GetHandler() http.Handler {
	target, err := s.parseTarget()
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		})
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	origDirector := proxy.Director
	basePath := target.Path
	transparent := s.backendConf.Transparent
	proxy.Director = func(req *http.Request) {
		origDirector(req)
		newPath := s.rewritePath(req.URL.Path, transparent.RewritePrefix, transparent.RewriteTo)
		req.URL.Path = s.joinPath(basePath, newPath)
		req.Host = target.Host
	}

	return s.auth(proxy)
}

func (s *TransparentServer) auth(next http.Handler) http.Handler {
	user := s.backendConf.Transparent.User
	pass := s.backendConf.Transparent.Password
	if user == "" && pass == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != user || p != pass {
			w.Header().Set("WWW-Authenticate", "Basic realm=\"transparent\"")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *TransparentServer) parseTarget() (*url.URL, error) {
	target := strings.TrimSpace(s.backendConf.Transparent.Target)
	if target == "" {
		return nil, fmt.Errorf("empty target")
	}
	if !strings.Contains(target, "://") {
		scheme := strings.TrimSpace(s.backendConf.Transparent.Scheme)
		if scheme == "" {
			scheme = "http"
		}
		target = scheme + "://" + target
	}
	return url.Parse(target)
}

func (s *TransparentServer) rewritePath(path string, prefix string, rewriteTo string) string {
	if prefix == "" || !strings.HasPrefix(path, prefix) {
		return path
	}
	rest := strings.TrimPrefix(path, prefix)
	if rewriteTo == "" {
		if rest == "" {
			return "/"
		}
		if !strings.HasPrefix(rest, "/") {
			return "/" + rest
		}
		return rest
	}
	if !strings.HasPrefix(rewriteTo, "/") {
		rewriteTo = "/" + rewriteTo
	}
	if strings.HasSuffix(rewriteTo, "/") && strings.HasPrefix(rest, "/") {
		rest = strings.TrimPrefix(rest, "/")
	}
	return rewriteTo + rest
}

func (s *TransparentServer) joinPath(base string, next string) string {
	if next == "" {
		next = "/"
	}
	if !strings.HasPrefix(next, "/") {
		next = "/" + next
	}
	if base == "" || base == "/" {
		return next
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	if strings.HasSuffix(base, "/") {
		base = strings.TrimSuffix(base, "/")
	}
	return base + next
}
