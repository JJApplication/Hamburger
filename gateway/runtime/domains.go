package runtime

import (
	"Hamburger/internal/config"
	"Hamburger/internal/structure"
	"Hamburger/internal/utils"
	"sync"
)

var (
	DomainLock        sync.RWMutex
	Domains           []string
	DomainsRuntimeMap struct {
		Domains        []string
		DomainsMap     *structure.Map[serviceMap]
		DomainFrontMap *structure.Map[string] // front -> domain
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

	dmap := map[string]serviceMap{}
	if err := utils.FileUnmarshal(domainFile, &dmap); err != nil {
		loadDefaultDomainsMap()
		return
	}
	m := structure.NewMap[serviceMap]()
	for key, val := range dmap {
		m.Put(key, val)
	}

	fm := structure.NewMap[string]()
	for key, val := range dmap {
		if val.Frontend != "" {
			fm.Put(val.Frontend, key)
		}
	}

	Domains = m.Keys()
	DomainLock.Lock()
	defer DomainLock.Unlock()
	DomainsRuntimeMap = struct {
		Domains        []string
		DomainsMap     *structure.Map[serviceMap]
		DomainFrontMap *structure.Map[string]
	}{Domains: m.Keys(), DomainsMap: m, DomainFrontMap: fm}
}

func loadDefaultDomainsMap() {
	DomainsRuntimeMap = struct {
		Domains        []string
		DomainsMap     *structure.Map[serviceMap]
		DomainFrontMap *structure.Map[string]
	}{Domains: nil, DomainsMap: structure.NewMap[serviceMap](), DomainFrontMap: structure.NewMap[string]()}
}

func GetDomainsSnapshot() ([]string, map[string]map[string]string, map[string]string) {
	DomainLock.RLock()
	defer DomainLock.RUnlock()

	domains := make([]string, len(DomainsRuntimeMap.Domains))
	copy(domains, DomainsRuntimeMap.Domains)

	domainMap := map[string]map[string]string{}
	for _, domain := range DomainsRuntimeMap.DomainsMap.Keys() {
		item, ok := DomainsRuntimeMap.DomainsMap.Get(domain)
		if !ok {
			continue
		}
		domainMap[domain] = map[string]string{
			"frontend": item.Frontend,
			"backend":  item.Backend,
		}
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
