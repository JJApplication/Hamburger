package health_probe

import (
	"context"
	"github.com/allegro/bigcache/v3"
	"sync"
	"time"
)

// 健康检查探针
//
// 健康检查状态以缓存存储在bigcache定时刷新

var (
	healthProbeCache *bigcache.BigCache
	probeTimesMu     sync.RWMutex
	probeTimes       = map[string]time.Time{}
)

var (
	HealthStatusLive = []byte("1")
	HealthStatusDead = []byte("0")
)

func init() {
	healthProbeCache, _ = bigcache.New(context.Background(), bigcache.Config{
		Shards:             32,
		LifeWindow:         0,
		CleanWindow:        0,
		MaxEntriesInWindow: 32,
		MaxEntrySize:       256,
	})
}

// GetAllProbes 以域名作为探针检查的key
//
// 对于纯后端服务检查其对应的端口
// 对于前端服务由于直接通过gw代理永远为alive
// 对于前后端服务仅检查后端
func GetAllProbes() map[string]string {
	if healthProbeCache == nil {
		return nil
	}
	result := make(map[string]string)
	cache := healthProbeCache.Iterator()
	for cache.SetNext() {
		info, err := cache.Value()
		if err != nil {
			continue
		}
		result[info.Key()] = string(info.Value())
	}

	return result
}

func SetProbe(domain string, probe []byte) {
	if healthProbeCache == nil {
		return
	}
	healthProbeCache.Set(domain, probe)
	probeTimesMu.Lock()
	probeTimes[domain] = time.Now().UTC()
	probeTimesMu.Unlock()
}

func GetProbe(domain string) []byte {
	if healthProbeCache == nil {
		return nil
	}
	result, _ := healthProbeCache.Get(domain)
	return result
}

// GetProbeTime reports when a domain was last checked. The timestamp is kept
// alongside the cache because the cache stores only the health result.
func GetProbeTime(domain string) (time.Time, bool) {
	probeTimesMu.RLock()
	value, ok := probeTimes[domain]
	probeTimesMu.RUnlock()
	return value, ok
}
