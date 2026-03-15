package stat

import (
	"os"

	"Hamburger/gateway/stat/db"
	"Hamburger/gateway/stat/model"
	"Hamburger/internal/json"
	"Hamburger/internal/logger"
	"Hamburger/internal/structure"
)

func (m *StatManager) LoadDomainStat() *structure.Map[*int64] {
	cfg := m.getCfg()
	if cfg.Stat.Compatible {
		m.compatibleDomain()
	}
	if cfg.Stat.UseDB {
		return m.domainDBLoader()
	}
	return m.domainFileLoader()
}

func (m *StatManager) SaveDomainStat() {
	cfg := m.getCfg()
	if cfg.Stat.UseDB {
		m.domainDBSaver()
	} else {
		m.domainFileSaver()
	}
}

func (m *StatManager) domainFileLoader() *structure.Map[*int64] {
	cfg := m.getCfg()
	statMap := structure.NewMap[*int64]()
	data, err := os.ReadFile(cfg.Stat.DomainFile)
	if err != nil {
		return statMap
	}
	var res map[string]int64
	if err = json.Unmarshal(data, &res); err != nil {
		return statMap
	}
	for k, v := range res {
		statMap.Put(k, &v)
	}

	return statMap
}

func (m *StatManager) domainFileSaver() {
	cfg := m.getCfg()
	if _, err := os.Stat(cfg.Stat.DomainFile); os.IsNotExist(err) {
		// 创建文件
		data, _ := json.Marshal(map[string]int64{})
		_ = os.WriteFile(cfg.Stat.DomainFile, data, os.ModePerm)
	}
	domainStatByte, err := m.C().Get(DomainStat)
	if err != nil {
		logger.GetLogger().Error().Err(err).Msg("Get DomainStat failed")
		return
	}
	_ = os.WriteFile(cfg.Stat.DomainFile, domainStatByte, os.ModePerm)
}

func (m *StatManager) domainDBLoader() *structure.Map[*int64] {
	statMap := structure.NewMap[*int64]()
	var domains []model.DomainModel
	db.GetDB().Model(&model.DomainModel{}).Find(&domains)
	for _, domain := range domains {
		statMap.Put(domain.Domain, &domain.Count)
	}

	return statMap
}

func (m *StatManager) domainDBSaver() {
	var data map[string]int64
	domainStatByte, err := m.C().Get(DomainStat)
	if err != nil {
		return
	}
	if err = json.Unmarshal(domainStatByte, &data); err != nil {
		return
	}
	for k, v := range data {
		var count int64
		db.GetDB().Model(&model.DomainModel{}).Where("domain = ?", k).Count(&count)
		if count > 0 {
			db.GetDB().Model(&model.DomainModel{}).Where("domain = ?", k).Update("count", v)
		} else {
			db.GetDB().Model(&model.DomainModel{}).Create(&model.DomainModel{
				Domain: k,
				Count:  v,
			})
		}
	}
}

func (m *StatManager) compatibleDomain() {
	cfg := m.getCfg()
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
		db.GetDB().Model(&model.DomainModel{}).Where("domain = ?", k).Count(&count)
		if count <= 0 {
			db.GetDB().Model(&model.DomainModel{}).Create(&model.DomainModel{
				Domain: k,
				Count:  v,
			})
		}
	}
}
