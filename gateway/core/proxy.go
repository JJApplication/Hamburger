package core

import (
	"Hamburger/gateway/precheck"
	"Hamburger/gateway/prehandler"
	"Hamburger/gateway/resolver"
	"Hamburger/internal/config"
	"Hamburger/internal/constant"
	"net/http"
	"sync"

	"github.com/rs/zerolog"
)

type Proxy struct {
	conf   *config.Config
	logger *zerolog.Logger
	lock   sync.RWMutex
	once   sync.Once

	handler      http.Handler
	frontHandler http.Handler
	ruler        *resolver.Ruler
}

func NewProxy(conf *config.Config, logger *zerolog.Logger) *Proxy {
	return &Proxy{
		conf:   conf,
		logger: logger,
		lock:   sync.RWMutex{},
		once:   sync.Once{},
		ruler:  resolver.NewRuler(conf, logger),
	}
}

func (p *Proxy) Create() http.Handler {
	p.once.Do(func() {
		switch p.conf.CoreProxy.ProxyMode {
		case constant.ProxyMode_HTTP:
			p.handler = NewHttpProxy(p.conf, p.logger)
		case constant.ProxyMode_FastHTTP:
			p.handler = NewFastProxy(p.conf, p.logger)
		default:
			p.handler = NewHttpProxy(p.conf, p.logger)
		}
	})

	return p.proxy()
}

func (p *Proxy) proxy() http.Handler {
	base := p.GlobalStaticAlias(p.handler)
	inProcess := p.ServeFrontendInProcess(base)
	checked := precheck.NewRouter(inProcess).Handler()
	return prehandler.WrapStripInternalHeaders(checked, p.conf)
}
