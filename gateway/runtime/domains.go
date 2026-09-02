package runtime

import (
	"net"
	"regexp"
	"strings"
	"sync"

	"Hamburger/internal/config"
	"Hamburger/internal/constant"
	"Hamburger/internal/logger"
	"Hamburger/internal/structure"
	"Hamburger/internal/utils"
)

var (
	DomainLock        sync.RWMutex
	DomainsRuntimeMap struct {
		Domains        []string
		DomainsMap     *structure.Map[config.Service] // 动态匹配域名 -> 服务
		RegexDomains   []RegexDomainService           // 仅正则域名，保持配置顺序
		DomainFrontMap *structure.Map[string]         // front -> domain
		ServiceMap     *structure.Map[config.Service] // 所有服务列表
	}
)

// RegexDomainService is a precompiled regular-expression domain rule.
type RegexDomainService struct {
	Pattern string
	Service config.Service
	Matcher *regexp.Regexp
}

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

	// 普通域名在加载时直接建立索引；只有正则域名保留到请求时遍历。
	m := structure.NewMap[config.Service]()
	var md []string
	var regexDomains []RegexDomainService

	sm := structure.NewMap[config.Service]()
	for _, service := range dmap.Sevices {
		sm.Put(service.ServiceName, service)
		// 添加域名正则表达式
		if service.ServiceDomain != "" {
			md = append(md, service.ServiceDomain)
			if utils.IsDomainRegex(service.ServiceDomain) {
				pattern := strings.TrimSpace(service.ServiceDomain)
				pattern = strings.TrimSpace(pattern[1 : len(pattern)-1])
				matcher, err := regexp.Compile(pattern)
				if err == nil {
					regexDomains = append(regexDomains, RegexDomainService{
						Pattern: service.ServiceDomain,
						Service: service,
						Matcher: matcher,
					})
				}
				continue
			}
			m.Put(strings.ToLower(NormalizeRequestHost(service.ServiceDomain)), service)
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
		RegexDomains   []RegexDomainService
		DomainFrontMap *structure.Map[string]
		ServiceMap     *structure.Map[config.Service]
	}{Domains: md, DomainsMap: m, RegexDomains: regexDomains, DomainFrontMap: fm, ServiceMap: sm}

	logger.L().Info().Int("count", DomainsRuntimeMap.DomainsMap.Size()).Msg("[runtime] domains rules")
	logger.L().Info().Int("count", DomainsRuntimeMap.ServiceMap.Size()).Msg("[runtime] services")
}

func loadDefaultDomainsMap() {
	DomainLock.Lock()
	defer DomainLock.Unlock()
	DomainsRuntimeMap = struct {
		Domains        []string
		DomainsMap     *structure.Map[config.Service]
		RegexDomains   []RegexDomainService
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
	normalizedHost := NormalizeRequestHost(host)
	hostKey := strings.ToLower(normalizedHost)
	DomainLock.RLock()
	domainsMap := DomainsRuntimeMap.DomainsMap
	// Runtime reload replaces the whole snapshot and never mutates this slice
	// in place, so retaining the immutable slice header avoids an allocation on
	// every exact-domain request.
	regexDomains := DomainsRuntimeMap.RegexDomains
	DomainLock.RUnlock()
	if domainsMap == nil {
		return config.Service{}, false
	}
	if domainMap, ok := domainsMap.Get(hostKey); ok {
		return domainMap, true
	}
	// 只有正则域名才进入请求路径遍历。
	for _, domain := range regexDomains {
		if domain.Matcher != nil && domain.Matcher.MatchString(normalizedHost) {
			domainsMap.Put(hostKey, domain.Service)
			return domain.Service, true
		}
	}

	return config.Service{}, false
}

// GetService returns a service from the current runtime snapshot without
// exposing the snapshot's mutable pointer to callers.
func GetService(name string) (config.Service, bool) {
	DomainLock.RLock()
	serviceMap := DomainsRuntimeMap.ServiceMap
	DomainLock.RUnlock()
	if serviceMap == nil {
		return config.Service{}, false
	}
	return serviceMap.Get(name)
}

// NormalizeRequestHost 去掉 Host 头中的端口，便于与 service_domain 匹配。
func NormalizeRequestHost(host string) string {
	host = strings.TrimSpace(host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		return strings.Trim(h, "[]")
	}
	if strings.Count(host, ":") == 1 && !strings.Contains(host, "]") {
		if idx := strings.LastIndex(host, ":"); idx > 0 {
			return host[:idx]
		}
	}
	return strings.Trim(host, "[]")
}

// FindPrecheckEnabledService 返回第一个开启前置检查的服务（用于按 IP 直接访问 challenge 路径等调试场景）。
func FindPrecheckEnabledService() (config.Service, bool) {
	DomainLock.RLock()
	defer DomainLock.RUnlock()
	if DomainsRuntimeMap.ServiceMap == nil {
		return config.Service{}, false
	}
	for _, service := range DomainsRuntimeMap.ServiceMap.Values() {
		if service.PreCheck.Enabled {
			return service, true
		}
	}
	return config.Service{}, false
}
