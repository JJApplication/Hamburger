package stat

import (
	"Hamburger/gateway/stat/db"
	"Hamburger/internal/config"
	"Hamburger/internal/json"
	"Hamburger/internal/logger"
	"Hamburger/internal/structure"
	"os"
)

func LoadDomainStat() *structure.Map[*int64] {
	cfg := config.Get()
	if cfg.Stat.Compatible {
		compatibleDomain(cfg)
	}
	if cfg.Stat.UseDB {
		return domainDBLoader()
	}
	return domainFileLoader(cfg)
}

func SaveDomainStat() {
	cfg := config.Get()
	if cfg.Stat.UseDB {
		domainDBSaver()
	} else {
		domainFileSaver(cfg)
	}
}

func domainFileLoader(cfg *config.Config) *structure.Map[*int64] {
	m := structure.NewMap[*int64]()
	data, err := os.ReadFile(cfg.Stat.DomainFile)
	if err != nil {
		return m
	}
	var res map[string]int64
	if err = json.Unmarshal(data, &res); err != nil {
		return m
	}
	for k, v := range res {
		m.Put(k, &v)
	}

	return m
}

func domainFileSaver(cfg *config.Config) {
	if _, err := os.Stat(cfg.Stat.DomainFile); os.IsNotExist(err) {
		// 创建文件
		data, _ := json.Marshal(map[string]int64{})
		_ = os.WriteFile(cfg.Stat.GeoFile, data, os.ModePerm)
	}
	domainStatByte, err := C().Get(DomainStat)
	if err != nil {
		logger.GetLogger().Error().Err(err).Msg("Get DomainStat failed")
		return
	}
	_ = os.WriteFile(cfg.Stat.DomainFile, domainStatByte, os.ModePerm)
}

func domainDBLoader() *structure.Map[*int64] {
	m := structure.NewMap[*int64]()
	var domains []DomainModel
	db.GetDB().Model(&DomainModel{}).Find(&domains)
	for _, domain := range domains {
		m.Put(domain.Domain, &domain.Count)
	}

	return m
}

func domainDBSaver() {
	var data map[string]int64
	domainStatByte, err := C().Get(DomainStat)
	if err != nil {
		return
	}
	if err = json.Unmarshal(domainStatByte, &data); err != nil {
		return
	}
	for k, v := range data {
		var count int64
		db.GetDB().Model(&DomainModel{}).Where("domain = ?", k).Count(&count)
		if count > 0 {
			db.GetDB().Model(&DomainModel{}).Where("domain = ?", k).Update("count", v)
		} else {
			db.GetDB().Model(&DomainModel{}).Create(&DomainModel{
				Domain: k,
				Count:  v,
			})
		}
	}
}

func compatibleDomain(cfg *config.Config) {
	data, err := os.ReadFile(cfg.Stat.DomainFile)
	if err != nil {
		return
	}
	var res map[string]int64
	if err = json.Unmarshal(data, &res); err != nil {
		return
	}
	for k, v := range res {
		var count int64
		db.GetDB().Model(&DomainModel{}).Where("domain = ?", k).Count(&count)
		if count <= 0 {
			db.GetDB().Model(&DomainModel{}).Create(&DomainModel{
				Domain: k,
				Count:  v,
			})
		}
	}
}
