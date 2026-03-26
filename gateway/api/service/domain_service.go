package service

import (
	"Hamburger/gateway/prehandler"
	"fmt"
	"strings"
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
