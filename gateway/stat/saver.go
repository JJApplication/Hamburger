package stat

import (
	"Hamburger/gateway/stat/db"
	"Hamburger/internal/config"
	"Hamburger/internal/json"
	"Hamburger/internal/structure"
	"os"
)

// 持久化存储数据到文件

func LoadStat() *structure.Map[int64] {
	cfg := config.Get()
	if cfg.Stat.Compatible {
		compatibleStat(cfg)
	}
	if cfg.Stat.UseDB {
		return dbLoader()
	}
	return fileLoader(cfg)
}

func SaveStat(cfg *config.Config) {
	if cfg.Stat.UseDB {
		dbSaver()
	} else {
		fileSaver(cfg.Stat.SaveFile)
	}
}

func fileSaver(f string) {
	if _, err := os.Stat(f); os.IsNotExist(err) {
		// 创建文件
		data, _ := json.Marshal(map[string]int64{
			"total":  0,
			"api":    0,
			"static": 0,
			"fail":   0,
			"today":  0,
		})
		_ = os.WriteFile(f, data, os.ModePerm)
	}
	m := make(map[string]int64)
	m["total"] = Get(Total)
	m["api"] = Get(API)
	m["static"] = Get(Static)
	m["fail"] = Get(Fail)
	m["today"] = Get(Today)

	data, _ := json.Marshal(m)
	_ = os.WriteFile(f, data, os.ModePerm)
}

func fileLoader(cfg *config.Config) *structure.Map[int64] {
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

func dbLoader() *structure.Map[int64] {
	var stat StatModel
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

func dbSaver() {
	var stat StatModel
	if err := db.GetDB().First(&stat).Error; err != nil {
		// 新建
		db.GetDB().Create(&StatModel{
			Total:  Get(Total),
			API:    Get(API),
			Static: Get(Static),
			Fail:   Get(Fail),
		})
		return
	}
	db.GetDB().Model(&StatModel{}).Where("id=?", stat.ID).Updates(map[string]interface{}{
		"total":  Get(Total),
		"api":    Get(API),
		"static": Get(Static),
		"fail":   Get(Fail),
	})
}

func compatibleStat(cfg *config.Config) {
	data, err := os.ReadFile(cfg.Stat.SaveFile)
	if err != nil {
		return
	}
	var tmp map[string]int64
	if err = json.Unmarshal(data, &tmp); err != nil {
		return
	}
	var stat StatModel
	if err = db.GetDB().First(&stat).Error; err != nil {
		// 新建
		db.GetDB().Create(&StatModel{
			Total:  tmp["total"],
			API:    tmp["api"],
			Static: tmp["static"],
			Fail:   tmp["fail"],
		})
		return
	}
}
