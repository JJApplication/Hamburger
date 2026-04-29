package core

import (
	"Hamburger/internal/config"
	"Hamburger/internal/constant"
	"github.com/rs/zerolog"
	"net/http"
	"sync"
)

type Proxy struct {
	conf   *config.Config
	logger *zerolog.Logger
	lock   sync.RWMutex
	once   sync.Once

	handler http.Handler
}

func NewProxy(conf *config.Config, logger *zerolog.Logger) *Proxy {
	return &Proxy{
		conf:   conf,
		logger: logger,
		lock:   sync.RWMutex{},
		once:   sync.Once{},
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
	return p.GlobalStaticAlias(p.handler)
}
