package runtime

import (
	"Hamburger/internal/data"
	"Hamburger/internal/logger"
	"Hamburger/internal/structure"
	"fmt"
)

// 域名和后端端口映射

var (
	DomainPortsMap *structure.Map[[]int]
)

func init() {
	DomainPortsMap = structure.NewMap[[]int]()
}

// InitDomainPortsMap 初始化域名端口组映射
func InitDomainPortsMap() {
	loadDomainPortsMap()
}

// RefreshDomainPortsMap 更新端口组
func RefreshDomainPortsMap() {
	loadDomainPortsMap()
}

func loadDomainPortsMap() {
	portsData := data.GetAppFromMongo()
	for _, v := range portsData {
		logger.GetLogger().Info().Str("app", v.Meta.Name).Str("domain", v.Meta.Meta.Domain).Any("ports", v.Meta.RunData.Ports).Msg("find app from mongo")
	}

	// 托管随机端口服务和固定端口服务
	for _, d := range portsData {
		logger.GetLogger().Info().Str("app", d.Meta.Name).Msg("load app to pool")
		if d.Meta.Meta.Domain != "" {
			DomainPortsMap.Put(d.Meta.Meta.Domain, d.Meta.RunData.Ports)
		}
		if d.Meta.RunData.RandomPort {
			logger.GetLogger().Info().Str("app", d.Meta.Name).Msg("load app of random port")
		}
	}

	logger.L().Info().Msg("load domainPortsPool")
	DomainPortsMap.Range(func(key string, value []int) bool {
		logger.L().Info().Str("app", key).Any("ports", value).Msg("load app ports info")
		return true
	})
}

func getDomainPort(host string) []int {
	if d, ok := DomainPortsMap.Get(host); ok {
		return d
	}
	return nil
}

// DomainReflect 将端口转换为ip地址 单机的ip都是127.0.0.1
func DomainReflect(host string) []string {
	group := getDomainPort(host)
	if len(group) == 0 {
		return nil
	}
	var dGroup []string
	for _, v := range group {
		dGroup = append(dGroup, fmt.Sprintf("127.0.0.1:%d", v))
	}

	return dGroup
}
