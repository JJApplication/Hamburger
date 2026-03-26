package prehandler

import (
	"fmt"
	"net"
	"strings"
	"sync"
)

type DomainServiceStatus string

const (
	DomainServiceRunning DomainServiceStatus = "running"
	DomainServiceStopped DomainServiceStatus = "stopped"
)

type ServiceManager struct {
	lock        sync.RWMutex
	domainState map[string]DomainServiceStatus
}

var (
	serviceManagerOnce sync.Once
	serviceManager     *ServiceManager
)

func GetServiceManager() *ServiceManager {
	serviceManagerOnce.Do(func() {
		serviceManager = &ServiceManager{
			domainState: make(map[string]DomainServiceStatus),
		}
	})
	return serviceManager
}

func (m *ServiceManager) StopDomain(domain string) error {
	normalized, err := normalizeDomain(domain)
	if err != nil {
		return err
	}
	m.lock.Lock()
	m.domainState[normalized] = DomainServiceStopped
	m.lock.Unlock()
	return nil
}

func (m *ServiceManager) StartDomain(domain string) error {
	normalized, err := normalizeDomain(domain)
	if err != nil {
		return err
	}
	m.lock.Lock()
	m.domainState[normalized] = DomainServiceRunning
	m.lock.Unlock()
	return nil
}

func (m *ServiceManager) IsDomainStopped(domain string) bool {
	normalized, err := normalizeDomain(domain)
	if err != nil {
		return false
	}
	m.lock.RLock()
	status, ok := m.domainState[normalized]
	m.lock.RUnlock()
	return ok && status == DomainServiceStopped
}

func normalizeDomain(domain string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(domain))
	if normalized == "" {
		return "", fmt.Errorf("domain is empty")
	}
	if strings.HasPrefix(normalized, "[") && strings.Contains(normalized, "]") {
		host, _, err := net.SplitHostPort(normalized)
		if err == nil {
			normalized = host
		}
	}
	if host, _, err := net.SplitHostPort(normalized); err == nil {
		normalized = host
	} else if strings.Count(normalized, ":") == 1 {
		if idx := strings.LastIndex(normalized, ":"); idx > 0 {
			normalized = normalized[:idx]
		}
	}
	normalized = strings.Trim(normalized, "[]")
	if normalized == "" {
		return "", fmt.Errorf("domain is empty")
	}
	return normalized, nil
}
