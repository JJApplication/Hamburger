package precheck_cache

import (
	"sync"
	"time"
)

// Manager 为每个 serviceName 维护独立缓存实例，便于按服务配置缓存大小与 lifeWindow。
type Manager struct {
	mu     sync.RWMutex
	caches map[string]*Cache
}

func NewManager() *Manager {
	return &Manager{
		caches: make(map[string]*Cache),
	}
}

func (m *Manager) GetOrCreate(serviceName string, cfg Config) (*Cache, error) {
	m.mu.RLock()
	c, ok := m.caches[serviceName]
	m.mu.RUnlock()
	if ok && c != nil {
		return c, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	// double-check
	if existing, ok := m.caches[serviceName]; ok && existing != nil {
		return existing, nil
	}
	newCache, err := New(cfg)
	if err != nil {
		return nil, err
	}
	m.caches[serviceName] = newCache
	return newCache, nil
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var firstErr error
	for name, c := range m.caches {
		if c == nil {
			continue
		}
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		delete(m.caches, name)
	}
	return firstErr
}

// DefaultCacheConfig 根据 TTL 推导一个合理的 lifeWindow。
func DefaultCacheConfig(ttl time.Duration, hardMaxMB int) Config {
	lw := ttl + 2*time.Minute
	if lw < 10*time.Minute {
		lw = 10 * time.Minute
	}
	return Config{
		LifeWindow:         lw,
		HardMaxCacheSize:   hardMaxMB,
		CleanupInterval:    60 * time.Second,
	}
}

