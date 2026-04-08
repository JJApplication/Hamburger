package runtime

import (
	"Hamburger/internal/config"
	"Hamburger/internal/constant"
	"Hamburger/internal/structure"
	"Hamburger/internal/utils"
	"sync"
)

var (
	DomainLock        sync.RWMutex
	Domains           []string
	DomainsRuntimeMap struct {
		Domains        []string
		DomainsMap     *structure.Map[config.DomainService] // 域名对应的服务
		DomainFrontMap *structure.Map[string]               // front -> domain
		ServiceMap     *structure.Map[config.Service]       // 所有服务列表
	}
)

func InitRuntimeDomains(cfg *config.AppConfig) {
	loadRuntimeDomains(cfg)
}

func loadRuntimeDomains(cfg *config.AppConfig) {
	domainFile := cfg.DomainMap
	if domainFile == "" {
		loadDefaultDomainsMap()
		return
	}

	var dmap config.DomainServiceMap
	if err := utils.FileUnmarshal(domainFile, &dmap); err != nil {
		loadDefaultDomainsMap()
		return
	}

	// validate
	if err := ValidateServiceMap(dmap); err != nil {
		loadDefaultDomainsMap()
		return
	}

	m := structure.NewMap[config.DomainService]()
	for _, val := range dmap.DomainService {
		m.Put(val.Domain, val)
	}

	sm := structure.NewMap[config.Service]()
	for _, service := range dmap.Sevices {
		sm.Put(service.ServiceName, service)
	}

	fm := structure.NewMap[string]()
	for _, domain := range dmap.DomainService {
		service, ok := sm.Get(domain.Service)
		if ok && service.ServiceType == constant.FrontendType {
			fm.Put(service.ServiceName, domain.Domain)
		}
	}

	Domains = m.Keys()
	DomainLock.Lock()
	defer DomainLock.Unlock()
	DomainsRuntimeMap = struct {
		Domains        []string
		DomainsMap     *structure.Map[config.DomainService]
		DomainFrontMap *structure.Map[string]
		ServiceMap     *structure.Map[config.Service]
	}{Domains: m.Keys(), DomainsMap: m, DomainFrontMap: fm, ServiceMap: sm}
}

func loadDefaultDomainsMap() {
	DomainsRuntimeMap = struct {
		Domains        []string
		DomainsMap     *structure.Map[config.DomainService]
		DomainFrontMap *structure.Map[string]
		ServiceMap     *structure.Map[config.Service]
	}{
		Domains:        nil,
		DomainsMap:     structure.NewMap[config.DomainService](),
		DomainFrontMap: structure.NewMap[string](),
		ServiceMap:     structure.NewMap[config.Service](),
	}
}

func GetDomainsSnapshot() ([]string, map[string]string, map[string]string) {
	DomainLock.RLock()
	defer DomainLock.RUnlock()

	domains := make([]string, len(DomainsRuntimeMap.Domains))
	copy(domains, DomainsRuntimeMap.Domains)

	domainMap := map[string]string{}
	for _, domain := range DomainsRuntimeMap.DomainsMap.Keys() {
		item, ok := DomainsRuntimeMap.DomainsMap.Get(domain)
		if !ok {
			continue
		}
		domainMap[domain] = item.Service
	}

	frontMap := map[string]string{}
	for _, key := range DomainsRuntimeMap.DomainFrontMap.Keys() {
		val, ok := DomainsRuntimeMap.DomainFrontMap.Get(key)
		if !ok {
			continue
		}
		frontMap[key] = val
	}

	return domains, domainMap, frontMap
}
