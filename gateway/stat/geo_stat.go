package stat

import (
	geo2 "Hamburger/gateway/geo"
	"Hamburger/internal/config"
	"Hamburger/internal/json"
	"Hamburger/internal/logger"
	"net"
	"sync/atomic"
)

// geo数据

const (
	GeoSet = "ip2country"
)

// 同步数据到缓存中
func syncGEOStat() {
	// 将临时的geo指针转换为数据
	geoDataMap := make(map[string]int64)

	geoIp.Range(func(key string, value *int64) bool {
		geoDataMap[key] = *value
		return true
	})

	data, err := json.Marshal(geoDataMap)
	if err != nil {
		logger.GetLogger().Error().Err(err).Msg("sync geoIp failed")
	}
	C().Set(GeoSet, data)
}

// AddGeo 使用协程处理 减少耗时
func AddGeo(addr string) {
	go func() {
		cfg := config.Get()
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
		geo, ok := geoIp.Get(isoCode)
		if !ok {
			geoIp.Put(isoCode, new(int64))
		} else {
			atomic.AddInt64(geo, 1)
		}
	}()
}

func GetGeoData() []byte {
	data, err := C().Get(GeoSet)
	if err != nil {
		return nil
	}
	return data
}
