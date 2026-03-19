package runtime

import (
	"Hamburger/internal/data"
	"Hamburger/internal/logger"
	"Hamburger/internal/structure"
)

// mongo db适配器

type DomainPortsMongoAdapter struct {
}

// LoadDomainPorts 从mongo db加载域名端口组映射
func (d *DomainPortsMongoAdapter) LoadDomainPorts() *structure.Map[[]int] {
	dataMap := structure.NewMap[[]int]()
	portsData := data.GetAppFromMongo()
	for _, v := range portsData {
		logger.GetLogger().Info().Str("app", v.Meta.Name).Str("domain", v.Meta.Meta.Domain).Any("ports", v.Meta.RunData.Ports).Msg("find app from mongo")
	}

	// 托管随机端口服务和固定端口服务
	for _, d := range portsData {
		logger.GetLogger().Info().Str("app", d.Meta.Name).Msg("load app to pool")
		// 仅代理有端口的域名服务
		if d.Meta.Meta.Domain != "" && len(d.Meta.RunData.Ports) > 0 {
			dataMap.Put(d.Meta.Meta.Domain, d.Meta.RunData.Ports)
			if d.Meta.RunData.RandomPort {
				logger.GetLogger().Info().Str("app", d.Meta.Name).Msg("load app of random port")
			}
		}
	}

	logger.L().Info().Msg("load domainPortsPool")
	dataMap.Range(func(key string, value []int) bool {
		logger.L().Info().Str("app", key).Any("ports", value).Msg("load app ports info")
		return true
	})

	return dataMap
}
