package prehandler

import (
	"Hamburger/internal/serror"
	"fmt"
	"net/http"
)

type DomainsControl struct {
	enabled bool
}

func NewDomainsControl() *DomainsControl {
	return &DomainsControl{
		enabled: true,
	}
}

func (d *DomainsControl) Handle(r *http.Request) error {
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
	if GetServiceManager().IsDomainStopped(domain) {
		r.Header.Set(serror.SandwichInternalFlag, serror.SandwichServiceStopped)
		return fmt.Errorf("domain %s service stopped", domain)
	}
	return nil
}

func (d *DomainsControl) Name() string {
	return "DomainsControl"
}

func (d *DomainsControl) Enabled() bool {
	return d.enabled
}
