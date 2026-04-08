package runtime

import (
	"Hamburger/internal/config"
	"fmt"
)

// ValidateServiceMap 校验域名服务映射
func ValidateServiceMap(dm config.DomainServiceMap) error {
	// 临时的域名map
	tempDomainMap := map[string]string{}
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
	}

	for _, domainService := range dm.DomainService {
		if domainService.Domain == "" || domainService.Service == "" {
			return fmt.Errorf("invalid domain service: %s", domainService.Service)
		}
		if _, ok := tempServiceMap[domainService.Service]; !ok {
			return fmt.Errorf("domain service name: %s not exist", domainService.Service)
		}
		if _, ok := tempDomainMap[domainService.Domain]; ok {
			return fmt.Errorf("duplicate domain service: %s", domainService.Service)
		}
		tempDomainMap[domainService.Domain] = domainService.Service
	}

	return nil
}
