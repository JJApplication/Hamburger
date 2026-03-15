package stat

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"Hamburger/internal/config"
	"Hamburger/internal/structure"

	"github.com/allegro/bigcache/v3"
)

type StatManager struct {
	cfg *config.Config

	bc *bigcache.BigCache

	total  int64
	api    int64
	static int64
	fail   int64
	today  int64

	geoIp *structure.Map[*int64]

	domainStat *structure.Map[*int64]

	connStatGw    *structure.Map[*int64]
	connStatFront *structure.Map[*int64]
}

var (
	managerOnce sync.Once
	managerInst *StatManager
)

func NewStatManager(cfg *config.Config) *StatManager {
	cache, _ := bigcache.New(context.Background(), bigcache.Config{
		Shards:             64,
		LifeWindow:         48 * time.Hour,
		CleanWindow:        30 * time.Minute,
		MaxEntriesInWindow: 256,
		MaxEntrySize:       1024,
	})
	return &StatManager{
		cfg:           cfg,
		bc:            cache,
		geoIp:         structure.NewMap[*int64](),
		domainStat:    structure.NewMap[*int64](),
		connStatGw:    structure.NewMap[*int64](),
		connStatFront: structure.NewMap[*int64](),
	}
}

func InitManager(cfg *config.Config) *StatManager {
	managerOnce.Do(func() {
		managerInst = NewStatManager(cfg)
	})
	return managerInst
}

func GetManager() *StatManager {
	return InitManager(config.Get())
}

func (m *StatManager) C() *bigcache.BigCache {
	return m.bc
}

func (m *StatManager) getCfg() *config.Config {
	if m.cfg != nil {
		return m.cfg
	}
	cfg := config.Get()
	m.cfg = cfg
	return cfg
}

func (m *StatManager) setCounters(total, api, static, fail, today int64) {
	atomic.StoreInt64(&m.total, total)
	atomic.StoreInt64(&m.api, api)
	atomic.StoreInt64(&m.static, static)
	atomic.StoreInt64(&m.fail, fail)
	atomic.StoreInt64(&m.today, today)
}
