package precheck_cache

import (
	"encoding/binary"
	"errors"
	"sync"
	"time"

	"github.com/allegro/bigcache/v3"
)

var (
	ErrNotFound = errors.New("precheck cache: not found")
)

type Config struct {
	// LifeWindow 是 bigcache 自带的过期窗口（不是精确 TTL）。
	// 为避免 key 长期堆积，建议 >= TTLSeconds + CleanupInterval。
	LifeWindow time.Duration

	// HardMaxCacheSize 以 MB 为单位，bigcache 达到该大小会开始淘汰旧 entry。
	HardMaxCacheSize int

	// CleanupInterval 定时清理间隔（例如 60s）。
	CleanupInterval time.Duration
}

// Cache 保存 requestID -> expireAtUnixMilli（毫秒）。
// bigcache 不支持遍历 key，因此额外维护一个索引表用于周期性清理过期 key。
type Cache struct {
	bc    *bigcache.BigCache
	index sync.Map // key(string) -> expireAtUnixMilli(int64)

	stopOnce sync.Once
	stopCh   chan struct{}
}

func New(cfg Config) (*Cache, error) {
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = 60 * time.Second
	}
	if cfg.LifeWindow <= 0 {
		// 兜底：bigcache lifeWindow 不宜太短，否则会提前回收。
		cfg.LifeWindow = 2 * time.Hour
	}

	c := &Cache{
		stopCh: make(chan struct{}),
	}

	bc, err := bigcache.NewBigCache(bigcache.Config{
		Shards:             1024,
		LifeWindow:         cfg.LifeWindow,
		CleanWindow:        cfg.CleanupInterval,
		MaxEntriesInWindow: 1024 * 10,
		MaxEntrySize:       128,
		HardMaxCacheSize:   cfg.HardMaxCacheSize,
		OnRemove: func(key string, _ []byte) {
			// 可能因为过期或淘汰触发，索引也要同步清理。
			c.index.Delete(key)
		},
	})
	if err != nil {
		return nil, err
	}
	c.bc = bc

	go c.janitor(cfg.CleanupInterval)
	return c, nil
}

func (c *Cache) Close() error {
	if c == nil {
		return nil
	}
	c.stopOnce.Do(func() {
		close(c.stopCh)
	})
	if c.bc != nil {
		return c.bc.Close()
	}
	return nil
}

func (c *Cache) Set(key string, expireAt time.Time) error {
	if c == nil || c.bc == nil {
		return errors.New("precheck cache: not initialized")
	}
	exp := expireAt.UnixMilli()
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(exp))
	c.index.Store(key, exp)
	return c.bc.Set(key, buf)
}

func (c *Cache) GetExpireAt(key string) (time.Time, error) {
	if c == nil || c.bc == nil {
		return time.Time{}, ErrNotFound
	}
	b, err := c.bc.Get(key)
	if err != nil {
		return time.Time{}, ErrNotFound
	}
	if len(b) < 8 {
		// 数据损坏，直接当不存在并清理
		_ = c.bc.Delete(key)
		c.index.Delete(key)
		return time.Time{}, ErrNotFound
	}
	expUnixMilli := int64(binary.BigEndian.Uint64(b[:8]))
	return time.UnixMilli(expUnixMilli), nil
}

func (c *Cache) IsValid(key string, now time.Time) bool {
	exp, err := c.GetExpireAt(key)
	if err != nil {
		return false
	}
	if now.Before(exp) {
		return true
	}
	_ = c.bc.Delete(key)
	c.index.Delete(key)
	return false
}

func (c *Cache) Delete(key string) {
	if c == nil || c.bc == nil {
		return
	}
	_ = c.bc.Delete(key)
	c.index.Delete(key)
}

func (c *Cache) janitor(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			nowUnixMilli := time.Now().UnixMilli()
			c.index.Range(func(k, v any) bool {
				key, ok := k.(string)
				if !ok {
					c.index.Delete(k)
					return true
				}
				expUnixMilli, ok := v.(int64)
				if !ok {
					c.index.Delete(key)
					return true
				}
				if expUnixMilli <= nowUnixMilli {
					_ = c.bc.Delete(key)
					c.index.Delete(key)
				}
				return true
			})
		}
	}
}

