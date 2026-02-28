package static_direct

import (
	"Hamburger/gateway/runtime"
	"Hamburger/internal/config"
	"Hamburger/internal/constant"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"github.com/valyala/fasthttp"
)

// 静态文件直通服务器
//
// 仅作静态代理不缓存数据

const (
	DirectHeader = "X-Direct-Access"
)

type StaticDirectServer struct {
	cfg                *config.Config
	logger             *zerolog.Logger
	dynamicFileHandler map[string][]DynamicFileHandler // 动态注册的文件处理器
	mu                 sync.RWMutex
	host               string
	port               int
	svr                *http.Server
	fastSvr            *fasthttp.Server
	fastListener       net.Listener
}

type DynamicFileHandler struct {
	Registered bool
	API        string
	Root       string // 唯一路径
	AllowExt   []string
}

var (
	staticDirectSvr *StaticDirectServer
)

func IsEnabled() bool {
	if staticDirectSvr == nil {
		return false
	}
	return staticDirectSvr.cfg.StaticDirectServer.Enabled
}

func GetSvr() *StaticDirectServer {
	return staticDirectSvr
}

func NewStaticDirectServer(cfg *config.Config, logger *zerolog.Logger) *StaticDirectServer {
	return &StaticDirectServer{
		cfg:                cfg,
		logger:             logger,
		dynamicFileHandler: initDynamicHandler(cfg.PxyFrontend.Servers),
	}
}

func initDynamicHandler(fcs []config.FrontServerConfig) map[string][]DynamicFileHandler {
	m := make(map[string][]DynamicFileHandler)
	for _, fc := range fcs {
		// 根据前端服务查询域名
		domain, ok := runtime.DomainsRuntimeMap.DomainFrontMap.Get(fc.Name)
		if ok && fc.Name != "" {
			m[domain] = make([]DynamicFileHandler, 0)
			for _, fb := range fc.Backends {
				if fb.IsStatic && fb.API != "" {
					m[domain] = append(m[domain], DynamicFileHandler{
						Registered: true,
						API:        fb.API,
						Root:       fb.StaticDirect.StaticRoot,
						AllowExt:   fb.StaticDirect.AllowExt,
					})
				}
			}
		}
	}

	return m
}

func (s *StaticDirectServer) Start() error {
	sdConfig := s.cfg.StaticDirectServer
	if !sdConfig.Enabled {
		return nil
	}
	s.host = sdConfig.Host
	s.port = sdConfig.Port
	addr := fmt.Sprintf("%s:%d", sdConfig.Host, sdConfig.Port)
	if s.cfg.CoreProxy.ProxyMode == constant.ProxyMode_FastHTTP {
		s.fastSvr = &fasthttp.Server{
			Handler: s.fastHandler,
		}
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		s.fastListener = listener
		return s.fastSvr.Serve(listener)
	}
	s.svr = &http.Server{
		Addr: addr,
	}
	s.RegRootHandler()

	return s.svr.ListenAndServe()
}

func (s *StaticDirectServer) RegRootHandler() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		directHost := r.Header.Get(DirectHeader)
		if directHost == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		requestPath := r.URL.Path
		s.mu.RLock()
		handlers := s.dynamicFileHandler[directHost]
		s.mu.RUnlock()
		if len(handlers) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		for _, handler := range handlers {
			if !handler.Registered || handler.API == "" || handler.Root == "" {
				continue
			}
			if !strings.HasPrefix(requestPath, handler.API) {
				continue
			}
			relative := strings.TrimPrefix(requestPath, handler.API)
			relative = strings.TrimPrefix(relative, "/")
			targetPath := filepath.Join(handler.Root, filepath.FromSlash(relative))
			if len(handler.AllowExt) > 0 && !isAllowedExt(handler.AllowExt, targetPath) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			http.ServeFile(w, r, targetPath)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	s.svr.Handler = mux
}

func (s *StaticDirectServer) Stop() error {
	sdConfig := s.cfg.StaticDirectServer
	if !sdConfig.Enabled {
		return nil
	}
	if s.cfg.CoreProxy.ProxyMode == constant.ProxyMode_FastHTTP {
		if s.fastListener != nil {
			_ = s.fastListener.Close()
		}
		if s.fastSvr != nil {
			return s.fastSvr.Shutdown()
		}
		return nil
	}
	return s.svr.Shutdown(context.Background())
}

func (s *StaticDirectServer) GetAddr() string {
	return fmt.Sprintf("%s:%d", s.host, s.port)
}

func (s *StaticDirectServer) IsStaticDirect(request *http.Request) (*url.URL, bool) {
	host := request.Host
	requestPath := request.URL.Path
	s.mu.RLock()
	defer s.mu.RUnlock()
	if handlers, ok := s.dynamicFileHandler[host]; ok {
		for _, handler := range handlers {
			if handler.Registered && strings.HasPrefix(requestPath, handler.API) {
				request.URL.Host = s.GetAddr()
				request.Header.Set("Host", request.Host) // 设置真实HOST
				request.Header.Set(DirectHeader, host)   // 直通的域名
				return request.URL, true
			}
		}
	}

	return request.URL, false
}

func isAllowedExt(allowExt []string, filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == "" {
		return false
	}
	for _, item := range allowExt {
		item = strings.TrimSpace(strings.ToLower(item))
		if item == "" {
			continue
		}
		if !strings.HasPrefix(item, ".") {
			item = "." + item
		}
		if ext == item {
			return true
		}
	}
	return false
}

func (s *StaticDirectServer) fastHandler(ctx *fasthttp.RequestCtx) {
	method := string(ctx.Method())
	if method != http.MethodGet && method != http.MethodHead {
		ctx.SetStatusCode(http.StatusMethodNotAllowed)
		return
	}
	directHost := string(ctx.Request.Header.Peek(DirectHeader))
	if directHost == "" {
		ctx.SetStatusCode(http.StatusNotFound)
		return
	}
	requestPath := string(ctx.Path())
	s.mu.RLock()
	handlers := s.dynamicFileHandler[directHost]
	s.mu.RUnlock()
	if len(handlers) == 0 {
		ctx.SetStatusCode(http.StatusNotFound)
		return
	}
	for _, handler := range handlers {
		if !handler.Registered || handler.API == "" || handler.Root == "" {
			continue
		}
		if !strings.HasPrefix(requestPath, handler.API) {
			continue
		}
		relative := strings.TrimPrefix(requestPath, handler.API)
		relative = strings.TrimPrefix(relative, "/")
		targetPath := filepath.Join(handler.Root, filepath.FromSlash(relative))
		if len(handler.AllowExt) > 0 && !isAllowedExt(handler.AllowExt, targetPath) {
			ctx.SetStatusCode(http.StatusForbidden)
			return
		}
		fasthttp.ServeFile(ctx, targetPath)
		return
	}
	ctx.SetStatusCode(http.StatusNotFound)
}
