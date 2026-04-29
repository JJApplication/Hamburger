package runtime

import (
	"Hamburger/internal/config"
	"Hamburger/internal/utils"
	"fmt"
)

// ValidateServiceMap 校验域名服务映射
func ValidateServiceMap(dm config.DomainServiceMap) error {
	// 临时的域名map
	tempDomainMap := map[string]struct{}{}
	// 域名正则规则
	var tempDomainPatterns []string
	// 临时的服务map
	tempServiceMap := map[string]struct{}{}

	for _, service := range dm.Sevices {
		if service.ServiceName == "" {
			return fmt.Errorf("service name is empty")
		}
		if _, ok := tempServiceMap[service.ServiceName]; ok {
			return fmt.Errorf("duplicate service name: %s", service.ServiceName)
		}
		tempServiceMap[service.ServiceName] = struct{}{}

		// 允许服务不配置域名 不对外暴露
		if service.ServiceDomain != "" {
			if _, ok := tempDomainMap[service.ServiceDomain]; ok {
				return fmt.Errorf("duplicate service domain: %s", service.ServiceDomain)
			}
			tempDomainPatterns = append(tempDomainPatterns, service.ServiceDomain)
			tempDomainMap[service.ServiceDomain] = struct{}{}
		}

	}

	return validateDomainDuplicate(tempDomainPatterns)
}

// 检查域名正则是否互斥规则
func validateDomainDuplicate(domains []string) error {
	if len(domains) == 0 {
		return nil
	}
	ok, badPatterns := utils.IsRegexListMutuallyExclusive(domains)
	if !ok {
		return fmt.Errorf("invalid domain list: %s", badPatterns)
	}
	return nil
}
