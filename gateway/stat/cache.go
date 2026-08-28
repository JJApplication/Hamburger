package stat

import (
	"Hamburger/gateway/stat/db"
	"Hamburger/gateway/stat/model"
)

func (m *StatManager) initCacheFromFile() {
	// 旧累计数据是否使用 SQLite 由 UseDB 独立控制。历史统计可能已经
	// 打开了同一个 DB，但 UseDB=false 时这里必须继续走文件加载路径。
	if cfg := m.getCfg(); cfg != nil && cfg.Stat.UseDB && db.GetDB() != nil {
		// 初始化旧累计表；新的历史表由 NewHistoryStore 创建。
		db.GetDB().AutoMigrate(&model.StatModel{})
		db.GetDB().AutoMigrate(&model.GeoModel{})
		db.GetDB().AutoMigrate(&model.DomainModel{})
	}
	statMap := m.LoadStat()
	if statMap != nil {
		m.setCounters(
			statMap.MustGet("total"),
			statMap.MustGet("api"),
			statMap.MustGet("static"),
			statMap.MustGet("fail"),
			statMap.MustGet("today"),
		)
	}
	// 立即初始化一次
	go m.syncStat()

	m.geoIp = m.LoadGeoStat()
	go m.syncGEOStat()

	// 加载域名统计信息
	m.domainStat = m.LoadDomainStat()
	go m.syncDomainStat()
}
