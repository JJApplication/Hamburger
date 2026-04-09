package core

import (
	"Hamburger/gateway/runtime"
	"net/http"
	"path/filepath"
	"strings"
)

// 全局默认的静态代理

func (p *Proxy) GlobalStaticAlias(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f, ok := p.matchStaticAlias(r); ok {
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

	domain, ok := runtime.DomainsRuntimeMap.DomainsMap.Get(host)
	if !ok {
		return "", false
	}
	service, ok := runtime.DomainsRuntimeMap.ServiceMap.Get(domain.Service)
	if !ok {
		return "", false
	}
	if len(service.ProxyPass) == 0 {
		return "", false
	}
	for _, proxy := range service.ProxyPass {
		if strings.HasPrefix(requestPath, proxy.API) && proxy.StaticDirect.StaticRoot != "" {
			urlPath := strings.TrimPrefix(requestPath, proxy.API)
			file = filepath.Join(proxy.StaticDirect.StaticRoot, urlPath)

			return file, true
		}
	}

	return "", false
}
