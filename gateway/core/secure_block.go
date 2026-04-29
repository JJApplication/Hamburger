package core

import (
	"Hamburger/gateway/runtime"
	"Hamburger/internal/utils"
	"strings"
)

// BlockBlackHost 对于恶意Host 不在安全允许名单中的恶意转发 直接屏蔽
func BlockBlackHost(host string) (block bool, skip bool) {
	if strings.HasPrefix(host, "127.0.0.1") || strings.HasPrefix(host, "localhost") || strings.HasPrefix(host, "::") {
		return false, true
	}
	// 标准化host去除后面的端口
	realHost := host
	if strings.Contains(host, ":") && len(strings.Split(host, ":")) > 0 {
		realHost = strings.Split(host, ":")[0]
	}
	for _, domain := range runtime.DomainsRuntimeMap.Domains {
		if domain == realHost {
			return false, false
		}
		if utils.MatchDomainByRegex(domain, realHost) {
			return false, false
		}
	}

	return true, true
}
