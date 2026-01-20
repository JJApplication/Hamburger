package runtime

import (
	"Hamburger/internal/config"
	"Hamburger/internal/json"
	"Hamburger/internal/structure"
	"os"
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

	data, err := os.ReadFile(domainFile)
	if err != nil {
		loadDefaultDomainsMap()
		return
	}
	dmap := map[string]serviceMap{}
	if err = json.Unmarshal(data, &dmap); err != nil {
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
