package stat

import (
	"os"

	"Hamburger/gateway/stat/db"
	"Hamburger/gateway/stat/model"
	"Hamburger/internal/json"
	"Hamburger/internal/logger"
	"Hamburger/internal/structure"
)

func (m *StatManager) LoadGeoStat() *structure.Map[*int64] {
	cfg := m.getCfg()
	if cfg.Stat.Compatible {
		m.compatibleGeo()
	}
	if cfg.Stat.UseDB {
		return m.geoDBLoader()
	}

	return m.geoFileLoader()
}

func (m *StatManager) SaveGeoStat() {
	cfg := m.getCfg()
	if cfg.Stat.UseDB {
		m.geoDBSaver()
	} else {
		m.geoFileSaver()
	}
}

func (m *StatManager) geoFileLoader() *structure.Map[*int64] {
	cfg := m.getCfg()
	var geoStat = structure.NewMap[*int64]()

	data, err := os.ReadFile(cfg.Stat.GeoFile)
	if err != nil {
		return geoStat
	}

	var tmp map[string]int64
	if err = json.Unmarshal(data, &tmp); err != nil {
		return geoStat
	}
	for k, v := range tmp {
		geoStat.Put(k, &v)
	}

	return geoStat
}

func (m *StatManager) geoDBLoader() *structure.Map[*int64] {
	var geoStat = structure.NewMap[*int64]()

	var geoData []model.GeoModel
	db.GetDB().Find(&geoData)
	for _, geo := range geoData {
		geoStat.Put(geo.ISOCode, &geo.Count)
	}

	return geoStat
}

func (m *StatManager) geoFileSaver() {
	cfg := m.getCfg()
	if _, err := os.Stat(cfg.Stat.GeoFile); os.IsNotExist(err) {
		// 创建文件
		data, _ := json.Marshal(map[string]int64{})
		_ = os.WriteFile(cfg.Stat.GeoFile, data, os.ModePerm)
	}
	geoStatByte, err := m.C().Get(GeoSet)
	if err != nil {
		logger.GetLogger().Error().Err(err).Msg("Get GeoSet failed")
		return
	}
	_ = os.WriteFile(cfg.Stat.GeoFile, geoStatByte, os.ModePerm)
}

func (m *StatManager) geoDBSaver() {
	var geoMap = make(map[string]int64)
	geoStatByte, err := m.C().Get(GeoSet)
	if err != nil {
		logger.GetLogger().Error().Err(err).Msg("Get GeoSet failed")
		return
	}
	if err := json.Unmarshal(geoStatByte, &geoMap); err != nil {
		logger.GetLogger().Error().Err(err).Msg("Unmarshal GeoSet failed")
		return
	}
	// 每次都是更新+增量相加
	for k, v := range geoMap {
		// 存在判断
		var count int64
		db.GetDB().Model(&model.GeoModel{}).Where("iso_code = ?", k).Count(&count)
		if count > 0 {
			db.GetDB().Model(&model.GeoModel{}).Where("iso_code = ?", k).Updates(map[string]interface{}{
				"count": v,
			})
		} else {
			db.GetDB().Create(&model.GeoModel{
				ISOCode: k,
				Count:   v,
			})
		}
	}
}

func (m *StatManager) compatibleGeo() {
	cfg := m.getCfg()
	// 加载文件内容到DB
	data, err := os.ReadFile(cfg.Stat.GeoFile)
	if err != nil {
		return
	}

	var geoMap map[string]int64
	if err = json.Unmarshal(data, &geoMap); err != nil {
		return
	}

	// 每次都是更新+增量相加
	for k, v := range geoMap {
		// 存在判断
		var count int64
		db.GetDB().Model(&model.GeoModel{}).Where("iso_code = ?", k).Count(&count)
		if count <= 0 {
			db.GetDB().Create(&model.GeoModel{
				ISOCode: k,
				Count:   v,
			})
		}
	}
}
