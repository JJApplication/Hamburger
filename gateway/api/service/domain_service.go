package service

import (
	"Hamburger/gateway/health_probe"
	"Hamburger/gateway/prehandler"
	"Hamburger/gateway/runtime"
	"Hamburger/internal/utils"
	"fmt"
	"strings"
	"time"
)

func (s *APIService) StopDomainService(domain string) error {
	if strings.TrimSpace(domain) == "" {
		return fmt.Errorf("domain is empty")
	}
	return prehandler.GetServiceManager().StopDomain(domain)
}

func (s *APIService) StartDomainService(domain string) error {
	if strings.TrimSpace(domain) == "" {
		return fmt.Errorf("domain is empty")
	}
	return prehandler.GetServiceManager().StartDomain(domain)
}

// DomainProbe is the management response for one manual health check.
type DomainProbe struct {
	Domain    string    `json:"domain"`
	Service   string    `json:"service,omitempty"`
	Status    string    `json:"status"`
	LastProbe time.Time `json:"last_probe"`
}

// ProbeManagementDomain checks one configured domain and records the result
// where the domain list reads it.
func (s *APIService) ProbeManagementDomain(domain string) (DomainProbe, error) {
	normalized := strings.ToLower(strings.TrimSpace(domain))
	if normalized == "" {
		return DomainProbe{}, fmt.Errorf("domain is empty")
	}
	if utils.IsDomainRegex(normalized) {
		return DomainProbe{}, fmt.Errorf("regular expression domains cannot be probed")
	}
	domains, serviceMap, _ := runtime.GetDomainsSnapshot()
	serviceName := serviceMap[normalized]
	if serviceName == "" {
		serviceName = serviceMap[strings.TrimSpace(domain)]
	}
	configured := serviceName != ""
	for _, item := range domains {
		if strings.EqualFold(strings.TrimSpace(item), normalized) {
			configured = true
			break
		}
	}
	if !configured {
		return DomainProbe{}, fmt.Errorf("domain is not configured")
	}
	result := health_probe.ProbeDomain(normalized, serviceName)
	return DomainProbe{Domain: result.Domain, Service: result.Service, Status: result.Status, LastProbe: result.CheckedAt}, nil
}
