package prehandler

import (
	"Hamburger/gateway/pre_auth"
	"Hamburger/internal/config/loader"
	"Hamburger/internal/serror"
	"net/http"
	"sync"
)

type PreAuth struct {
	enabled bool
	service *pre_auth.Service
}

var (
	preAuthOnce sync.Once
	preAuth     *PreAuth
)

func NewPreAuth() *PreAuth {
	preAuthOnce.Do(func() {
		cfg := loader.Get()
		service := pre_auth.NewService(cfg.PreAuthConfig)
		preAuth = &PreAuth{
			enabled: service.Enabled(),
			service: service,
		}
	})
	return preAuth
}

func (p *PreAuth) Handle(req *http.Request) error {
	if !p.Enabled() {
		return nil
	}
	_, err := p.service.Authenticate(req)
	if err != nil {
		req.Header.Set(serror.SandwichInternalFlag, serror.SandwichPreAuthFailed)
		return err
	}
	return nil
}

func (p *PreAuth) Name() string {
	return "PreAuth"
}

func (p *PreAuth) Enabled() bool {
	return p.enabled
}
