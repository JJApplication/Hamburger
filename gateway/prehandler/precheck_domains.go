package prehandler

import (
	"Hamburger/gateway/runtime"
	"Hamburger/internal/config/loader"
	"Hamburger/internal/utils"
	"fmt"
	"net/http"
)

type PreCheckDomains struct {
	enabled        bool
	allowedDomains []string
}

func NewPreCheckDomains() *PreCheckDomains {
	cf := loader.Get()
	return &PreCheckDomains{
		enabled:        cf.Middleware.DomainCheck.Enabled,
		allowedDomains: runtime.DomainsRuntimeMap.Domains,
	}
}

func (cf *PreCheckDomains) Handle(r *http.Request) error {
	domain, err := normalizeDomain(r.Host)
	if err != nil {
		return nil
	}
	if domain == "" {
		return nil
	}
	if domain == "127.0.0.1" || domain == "localhost" {
		return nil
	}
	switch {
	}
	for _, ad := range cf.allowedDomains {
		if ad == domain {
			return nil
		}
		if utils.MatchDomainByRegex(ad, domain) {
			return nil
		}
	}

	return fmt.Errorf("domain %s not allowed", domain)
}

func (cf *PreCheckDomains) Name() string {
	return "PreCheckDomains"
}

func (cf *PreCheckDomains) Enabled() bool {
	return cf.enabled
}
