package webdav

import (
	"Hamburger/internal/config"
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/webdav"

	"github.com/rs/zerolog"
)

type WebDAVServer struct {
	cfg      *config.Config
	logger   *zerolog.Logger
	enabled  bool
	host     string
	port     int
	prefix   string
	timeout  time.Duration
	server   *http.Server
	listener net.Listener

	lockSystem webdav.LockSystem
	handlers   map[string]http.Handler
	users      map[string]config.WebDAVUserConfig

	mu      sync.Mutex
	started bool
}

func NewWebDAVServer(cfg *config.Config, logger *zerolog.Logger) *WebDAVServer {
	conf := cfg.ExpConfig.WebDAV
	host := conf.Host
	if host == "" {
		host = "0.0.0.0"
	}
	port := conf.Port
	if port <= 0 {
		port = 1900
	}
	prefix := conf.PathPrefix
	if prefix == "" {
		prefix = "/"
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if prefix != "/" {
		prefix = strings.TrimRight(prefix, "/")
	}
	timeout := time.Duration(conf.ReadTimeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	s := &WebDAVServer{
		cfg:        cfg,
		logger:     logger,
		enabled:    conf.Enabled,
		host:       host,
		port:       port,
		prefix:     prefix,
		timeout:    timeout,
		lockSystem: webdav.NewMemLS(),
		handlers:   make(map[string]http.Handler),
		users:      make(map[string]config.WebDAVUserConfig),
	}
	s.buildUserHandlers(conf)
	return s
}

func (s *WebDAVServer) Start() error {
	if !s.enabled {
		return nil
	}
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return nil
	}
	if len(s.handlers) == 0 {
		s.mu.Unlock()
		return errors.New("webdav users are not configured")
	}
	mux := http.NewServeMux()
	handler := s.authHandler()
	if s.prefix == "/" {
		mux.Handle("/", handler)
	} else {
		mux.Handle(s.prefix+"/", http.StripPrefix(s.prefix, handler))
		mux.Handle(s.prefix, http.RedirectHandler(s.prefix+"/", http.StatusMovedPermanently))
	}
	addr := net.JoinHostPort(s.host, strconv.Itoa(s.port))
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  s.timeout,
		WriteTimeout: s.resolveWriteTimeout(),
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	s.server = server
	s.listener = ln
	s.started = true
	s.mu.Unlock()

	s.logger.Info().Str("address", addr).Str("prefix", s.prefix).Msg("webdav server started")
	err = server.Serve(ln)
	if err == nil || errors.Is(err, http.ErrServerClosed) || errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func (s *WebDAVServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		return nil
	}
	s.started = false
	if s.server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()
	err := s.server.Shutdown(ctx)
	if s.listener != nil {
		_ = s.listener.Close()
	}
	return err
}

func (s *WebDAVServer) buildUserHandlers(conf config.WebDAVConfig) {
	for _, user := range conf.Users {
		username := strings.TrimSpace(user.Username)
		if username == "" || user.Password == "" || strings.TrimSpace(user.RootDir) == "" {
			continue
		}
		rootDir, err := filepath.Abs(filepath.Clean(user.RootDir))
		if err != nil {
			continue
		}
		fs := webdav.FileSystem(webdav.Dir(rootDir))
		if conf.ReadOnly || user.ReadOnly {
			fs = readOnlyFileSystem{fs: fs}
		}
		s.users[username] = user
		s.handlers[username] = &webdav.Handler{
			Prefix:     "/",
			FileSystem: fs,
			LockSystem: s.lockSystem,
		}
	}
}

func (s *WebDAVServer) authHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			s.writeUnauthorized(w)
			return
		}
		user, found := s.users[username]
		if !found || user.Password != password {
			s.writeUnauthorized(w)
			return
		}
		handler := s.handlers[username]
		handler.ServeHTTP(w, r)
	})
}

func (s *WebDAVServer) writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="webdav"`)
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}

func (s *WebDAVServer) resolveWriteTimeout() time.Duration {
	writeTimeout := time.Duration(s.cfg.ExpConfig.WebDAV.WriteTimeout) * time.Second
	if writeTimeout <= 0 {
		return s.timeout
	}
	return writeTimeout
}

type readOnlyFileSystem struct {
	fs webdav.FileSystem
}

func (r readOnlyFileSystem) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return os.ErrPermission
}

func (r readOnlyFileSystem) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	if flag&(os.O_RDWR|os.O_WRONLY|os.O_APPEND|os.O_CREATE|os.O_TRUNC) != 0 {
		return nil, os.ErrPermission
	}
	return r.fs.OpenFile(ctx, name, flag, perm)
}

func (r readOnlyFileSystem) RemoveAll(ctx context.Context, name string) error {
	return os.ErrPermission
}

func (r readOnlyFileSystem) Rename(ctx context.Context, oldName, newName string) error {
	return os.ErrPermission
}

func (r readOnlyFileSystem) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	return r.fs.Stat(ctx, name)
}
