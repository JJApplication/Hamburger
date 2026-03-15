package stat

import (
	"net"
	"sync/atomic"

	geo2 "Hamburger/gateway/geo"
	"Hamburger/internal/json"
	"Hamburger/internal/logger"
)

// geo数据

const (
	GeoSet = "ip2country"
)

func (m *StatManager) syncGEOStat() {
	// 将临时的geo指针转换为数据
	geoDataMap := make(map[string]int64)

	m.geoIp.Range(func(key string, value *int64) bool {
		geoDataMap[key] = *value
		return true
	})

	data, err := json.Marshal(geoDataMap)
	if err != nil {
		logger.GetLogger().Error().Err(err).Msg("sync geoIp failed")
	}
	m.C().Set(GeoSet, data)
}

// AddGeo 使用协程处理 减少耗时
func AddGeo(addr string) {
	GetManager().AddGeo(addr)
}

func (m *StatManager) AddGeo(addr string) {
	go func() {
		if addr == "" || addr == "127.0.0.1" {
			return
		}
		cfg := m.getCfg()
		if !cfg.Stat.EnableStat {
			return
		}
		ip, _, err := net.SplitHostPort(addr)
		if err != nil {
			return
		}
		isoCode := geo2.GeoLookUp(ip)
		if isoCode == "" {
			return
		}

		// 原子操作geo指针时 只需要读锁
		geo, ok := m.geoIp.Get(isoCode)
		if !ok {
			m.geoIp.Put(isoCode, new(int64))
		} else {
			atomic.AddInt64(geo, 1)
		}
	}()
}

func GetGeoData() []byte {
	return GetManager().GetGeoData()
}

func (m *StatManager) GetGeoData() []byte {
	data, err := m.C().Get(GeoSet)
	if err != nil {
		return nil
	}
	return data
}
