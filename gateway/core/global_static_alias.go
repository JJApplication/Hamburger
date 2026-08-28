package core

import (
	"Hamburger/gateway/runtime"
	"Hamburger/gateway/stat"
	"Hamburger/internal/structure"
	"net/http"
	"path/filepath"
	"strings"
)

// 全局默认的静态代理

var (
	// service: {api: file}
	globalStaticCache *structure.Map[*structure.Map[string]]
)

func init() {
	globalStaticCache = structure.NewMap[*structure.Map[string]]()
}

func (p *Proxy) GlobalStaticAlias(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f, ok := p.matchStaticAlias(r); ok {
			stat.MarkRoute(r, stat.RouteFrontend)
			http.ServeFile(w, r, f)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// 仅针对自定义服务
func (p *Proxy) matchStaticAlias(req *http.Request) (file string, ok bool) {
	host := req.Host
	requestPath := req.URL.Path

	if host == "" || requestPath == "" {
		return "", false
	}

	if strings.Contains(requestPath, "..") {
		return "", false
	}

	service, ok := runtime.GetDomain2Service(host)
	if !ok {
		return "", false
	}
	if len(service.ProxyPass) == 0 {
		return "", false
	}
	if file, ok = p.cachedAlias(service.ServiceName, requestPath); ok {
		return file, true
	}
	for _, proxy := range service.ProxyPass {
		if strings.HasPrefix(requestPath, proxy.API) && proxy.StaticDirect.StaticRoot != "" {
			urlPath := strings.TrimPrefix(requestPath, proxy.API)
			file = filepath.Join(proxy.StaticDirect.StaticRoot, urlPath)
			p.cacheAlias(service.ServiceName, requestPath, file)

			return file, true
		}
	}

	return "", false
}

func (p *Proxy) cachedAlias(service string, requestPath string) (string, bool) {
	if sc, ok := globalStaticCache.Get(service); ok {
		if file, ok := sc.Get(requestPath); ok {
			return file, true
		}
	}

	return "", false
}

func (p *Proxy) cacheAlias(service, requestPath, file string) {
	if sc, ok := globalStaticCache.Get(service); ok {
		if _, ok = sc.Get(requestPath); ok {
			return
		}
		sc.Put(requestPath, file)
	}
	sc := structure.NewMap[string]()
	sc.Put(requestPath, file)
	globalStaticCache.Put(service, sc)
}
