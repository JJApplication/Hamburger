package runtime

import (
	"Hamburger/internal/config"
	"Hamburger/internal/constant"
	"Hamburger/internal/logger"
	"Hamburger/internal/structure"
	"Hamburger/internal/utils"
	"sync"
)

var (
	DomainLock        sync.RWMutex
	DomainsRuntimeMap struct {
		Domains        []string
		DomainsMap     *structure.Map[config.Service] // 动态匹配域名 -> 服务
		DomainFrontMap *structure.Map[string]         // front -> domain
		ServiceMap     *structure.Map[config.Service] // 所有服务列表
	}
)

func InitRuntimeDomains(cfg *config.AppConfig) {
	loadRuntimeDomains(cfg)
}

func loadRuntimeDomains(cfg *config.AppConfig) {
	domainFile := cfg.DomainMap
	if domainFile == "" {
		logger.L().Warn().Str("file", domainFile).Msg("domain service file not exist")
		loadDefaultDomainsMap()
		return
	}

	var dmap config.DomainServiceMap
	if err := utils.FileUnmarshal(domainFile, &dmap); err != nil {
		logger.L().Fatal().Str("file", domainFile).Err(err).Msg("unmarshal domain service map failed")
		loadDefaultDomainsMap()
		return
	}

	// validate
	if err := ValidateServiceMap(dmap); err != nil {
		logger.L().Fatal().Err(err).Msg("validate service map failed")
		// validate 失败应该退出程序后检查
	}

	// 域名 -> 服务映射初始为空
	m := structure.NewMap[config.Service]()
	var md []string

	sm := structure.NewMap[config.Service]()
	for _, service := range dmap.Sevices {
		sm.Put(service.ServiceName, service)
		// 添加域名正则表达式
		if service.ServiceDomain != "" {
			md = append(md, service.ServiceDomain)
		}
	}

	// 前端服务 -> 域名正则映射
	fm := structure.NewMap[string]()
	for _, service := range dmap.Sevices {
		if service.ServiceType == constant.FrontendType {
			fm.Put(service.ServiceName, service.ServiceDomain)
		}
	}

	DomainLock.Lock()
	defer DomainLock.Unlock()
	DomainsRuntimeMap = struct {
		Domains        []string
		DomainsMap     *structure.Map[config.Service]
		DomainFrontMap *structure.Map[string]
		ServiceMap     *structure.Map[config.Service]
	}{Domains: md, DomainsMap: m, DomainFrontMap: fm, ServiceMap: sm}

	logger.L().Info().Int("count", DomainsRuntimeMap.DomainsMap.Size()).Msg("[runtime] domains rules")
	logger.L().Info().Int("count", DomainsRuntimeMap.ServiceMap.Size()).Msg("[runtime] services")
}

func loadDefaultDomainsMap() {
	DomainsRuntimeMap = struct {
		Domains        []string
		DomainsMap     *structure.Map[config.Service]
		DomainFrontMap *structure.Map[string]
		ServiceMap     *structure.Map[config.Service]
	}{
		Domains:        nil,
		DomainsMap:     structure.NewMap[config.Service](),
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
		domainMap[domain] = item.ServiceName
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

func GetDomain2Service(host string) (config.Service, bool) {
	if domainMap, ok := DomainsRuntimeMap.DomainsMap.Get(host); ok {
		return domainMap, true
	}
	// 基于domain的正则解析
	for _, service := range DomainsRuntimeMap.ServiceMap.Values() {
		if service.ServiceDomain != "" {
			if utils.MatchDomainByRegex(service.ServiceDomain, host) {
				DomainsRuntimeMap.DomainsMap.Put(host, service)
				return service, true
			}
		}
	}

	return config.Service{}, false
}
