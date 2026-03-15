package stat

import (
	"sync/atomic"

	"Hamburger/internal/json"
	"Hamburger/internal/logger"
)

const (
	DomainStat = "domain"
)

func AddDomainStat(domain string) {
	GetManager().AddDomainStat(domain)
}

func (m *StatManager) AddDomainStat(domain string) {
	go func() {
		cfg := m.getCfg()
		if !cfg.Stat.EnableStat {
			return
		}
		if domain == "" {
			return
		}

		// 原子操作geo指针时 只需要读锁
		ds, ok := m.domainStat.Get(domain)
		if !ok {
			m.domainStat.Put(domain, new(int64))
		} else {
			atomic.AddInt64(ds, 1)
		}
	}()
}

func GetDomainStat() []byte {
	return GetManager().GetDomainStat()
}

func (m *StatManager) GetDomainStat() []byte {
	data, err := m.C().Get(DomainStat)
	if err != nil {
		return nil
	}
	return data
}

func (m *StatManager) syncDomainStat() {
	domainDataMap := make(map[string]int64)

	m.domainStat.Range(func(key string, value *int64) bool {
		domainDataMap[key] = *value
		return true
	})

	data, err := json.Marshal(domainDataMap)
	if err != nil {
		logger.GetLogger().Error().Err(err).Msg("sync domainStat failed")
	}
	m.C().Set(DomainStat, data)
}
