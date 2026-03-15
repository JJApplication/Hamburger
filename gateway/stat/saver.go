package stat

import (
	"os"

	"Hamburger/gateway/stat/db"
	"Hamburger/gateway/stat/model"
	"Hamburger/internal/json"
	"Hamburger/internal/structure"
)

// 持久化存储数据到文件

func (m *StatManager) LoadStat() *structure.Map[int64] {
	cfg := m.getCfg()
	if cfg.Stat.Compatible {
		m.compatibleStat()
	}
	if cfg.Stat.UseDB {
		return m.dbLoader()
	}
	return m.fileLoader()
}

func (m *StatManager) SaveStat() {
	cfg := m.getCfg()
	if cfg.Stat.UseDB {
		m.dbSaver()
	} else {
		m.fileSaver()
	}
}

func (m *StatManager) fileSaver() {
	cfg := m.getCfg()
	if _, err := os.Stat(cfg.Stat.SaveFile); os.IsNotExist(err) {
		// 创建文件
		data, _ := json.Marshal(map[string]int64{
			"total":  0,
			"api":    0,
			"static": 0,
			"fail":   0,
			"today":  0,
		})
		_ = os.WriteFile(cfg.Stat.SaveFile, data, os.ModePerm)
	}
	statData := make(map[string]int64)
	statData["total"] = m.Get(Total)
	statData["api"] = m.Get(API)
	statData["static"] = m.Get(Static)
	statData["fail"] = m.Get(Fail)
	statData["today"] = m.Get(Today)

	data, _ := json.Marshal(statData)
	_ = os.WriteFile(cfg.Stat.SaveFile, data, os.ModePerm)
}

func (m *StatManager) fileLoader() *structure.Map[int64] {
	cfg := m.getCfg()
	data, err := os.ReadFile(cfg.Stat.SaveFile)
	if err != nil {
		return structure.NewMap[int64]()
	}
	var stat = structure.NewMap[int64]()
	var tmp map[string]int64
	if err = json.Unmarshal(data, &tmp); err != nil {
		return structure.NewMap[int64]()
	}

	for k, v := range tmp {
		stat.Put(k, v)
	}

	return stat
}

func (m *StatManager) dbLoader() *structure.Map[int64] {
	var stat model.StatModel
	if err := db.GetDB().First(&stat).Error; err != nil {
		return nil
	}
	var statMap = structure.NewMap[int64]()
	statMap.Put("total", stat.Total)
	statMap.Put("api", stat.API)
	statMap.Put("static", stat.Static)
	statMap.Put("fail", stat.Fail)

	return statMap
}

func (m *StatManager) dbSaver() {
	var stat model.StatModel
	if err := db.GetDB().First(&stat).Error; err != nil {
		// 新建
		db.GetDB().Create(&model.StatModel{
			Total:  m.Get(Total),
			API:    m.Get(API),
			Static: m.Get(Static),
			Fail:   m.Get(Fail),
		})
		return
	}
	db.GetDB().Model(&model.StatModel{}).Where("id=?", stat.ID).Updates(map[string]interface{}{
		"total":  m.Get(Total),
		"api":    m.Get(API),
		"static": m.Get(Static),
		"fail":   m.Get(Fail),
	})
}

func (m *StatManager) compatibleStat() {
	cfg := m.getCfg()
	data, err := os.ReadFile(cfg.Stat.SaveFile)
	if err != nil {
		return
	}
	var tmp map[string]int64
	if err = json.Unmarshal(data, &tmp); err != nil {
		return
	}
	var stat model.StatModel
	if err = db.GetDB().First(&stat).Error; err != nil {
		// 新建
		db.GetDB().Create(&model.StatModel{
			Total:  tmp["total"],
			API:    tmp["api"],
			Static: tmp["static"],
			Fail:   tmp["fail"],
		})
		return
	}
}
