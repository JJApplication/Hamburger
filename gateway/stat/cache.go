package stat

import (
	"Hamburger/gateway/stat/db"
	"Hamburger/gateway/stat/model"
)

func (m *StatManager) initCacheFromFile() {
	// 初始化数据库
	db.GetDB().AutoMigrate(&model.StatModel{})
	db.GetDB().AutoMigrate(&model.GeoModel{})
	db.GetDB().AutoMigrate(&model.DomainModel{})
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
