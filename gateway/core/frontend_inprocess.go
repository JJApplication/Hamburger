package core

import (
	"Hamburger/frontend_proxy"
	"Hamburger/gateway/stat"
	"net/http"
)

// ServeFrontendInProcess serves frontend static files by calling Helios in
// the same process. The site name travels on the request context, so the
// request does not loop back through the frontend listener.
func (p *Proxy) ServeFrontendInProcess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		front := p.frontHandlerSnapshot()
		if front != nil && p.ruler != nil {
			if site, ok := p.ruler.FrontendStaticSite(r); ok {
				r = r.WithContext(frontend_proxy.WithSiteName(r.Context(), site))
				stat.MarkRoute(r, stat.RouteFrontend)
				stat.Add(stat.Total)
				stat.Add(stat.Static)
				stat.AddDomainStat(r.Host)
				stat.AddGeo(r.RemoteAddr)
				front.ServeHTTP(w, r)
				return
			}
		}
		if next != nil {
			next.ServeHTTP(w, r)
		}
	})
}

func (p *Proxy) frontHandlerSnapshot() http.Handler {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.frontHandler
}

// SetFrontHandler installs the in-process Helios handler used for frontend
// static requests. It can be called before the gateway starts serving.
func (p *Proxy) SetFrontHandler(h http.Handler) {
	p.lock.Lock()
	p.frontHandler = h
	p.lock.Unlock()
}
