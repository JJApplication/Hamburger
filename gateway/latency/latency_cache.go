package latency

import (
	"Hamburger/internal/config/loader"
	"Hamburger/internal/logger"
	"context"
	"strings"
	"sync"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/gookit/goutil/timex"
)

var (
	latencyCache *bigcache.BigCache
	once         sync.Once
	requestMu    sync.Mutex
	requestStats = make(map[string]*requestStat)
)

func GetLatencyCache() *bigcache.BigCache {
	once.Do(func() {
		cfg := loader.Get()
		if cfg.Latency.Enabled {
			lc, err := bigcache.New(context.Background(), bigcache.Config{
				Shards:       64,
				LifeWindow:   timex.OneDay * 7,
				CleanWindow:  timex.OneDay,
				MaxEntrySize: 256,
			})
			if err != nil {
				logger.L().Err(err).Msg("create latency cache")
			}
			latencyCache = lc
		}
	})

	return latencyCache
}

func GetLatencyBlackList() []string {
	cache := GetLatencyCache()
	if cache == nil {
		return nil
	}
	iter := cache.Iterator()
	blacklistItems := make([]string, 0)
	for iter.SetNext() {
		info, err := iter.Value()
		if err != nil {
			continue
		}
		key := info.Key()
		if strings.HasPrefix(key, latencyBlackListPrefix) {
			blacklistItems = append(blacklistItems, strings.TrimPrefix(key, latencyBlackListPrefix))
		}
	}
	return blacklistItems
}

func AddToBlackList(addr string) {
	if addr == "" {
		return
	}
	cache := GetLatencyCache()
	if cache == nil {
		return
	}
	_ = cache.Set(latencyBlackListKey(addr), []byte("1"))
}

func IsInBlackList(addr string) bool {
	if addr == "" {
		return false
	}
	cache := GetLatencyCache()
	if cache == nil {
		return false
	}
	_, err := cache.Get(latencyBlackListKey(addr))
	return err == nil
}

type requestStat struct {
	WindowStart time.Time
	Count       int
}

func recordLatencyRequest(addr string, window time.Duration, limit int) bool {
	if addr == "" || limit <= 0 || window <= 0 {
		return false
	}
	now := time.Now()
	requestMu.Lock()
	stat, exists := requestStats[addr]
	if !exists || now.Sub(stat.WindowStart) > window {
		requestStats[addr] = &requestStat{
			WindowStart: now,
			Count:       1,
		}
		requestMu.Unlock()
		return false
	}
	stat.Count++
	overLimit := stat.Count > limit
	requestMu.Unlock()
	if overLimit {
		AddToBlackList(addr)
	}
	return overLimit
}

func latencyBlackListKey(addr string) string {
	return latencyBlackListPrefix + addr
}

const latencyBlackListPrefix = "latency:blacklist:"
