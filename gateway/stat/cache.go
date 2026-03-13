package stat

import (
	"Hamburger/gateway/stat/db"
	"Hamburger/gateway/stat/model"
	"Hamburger/internal/structure"
	"context"
	"sync/atomic"
	"time"

	"github.com/allegro/bigcache/v3"
)

var (
	bc     *bigcache.BigCache
	total  int64
	api    int64
	static int64
	fail   int64
	today  int64

	// geo数据
	geoIp *structure.Map[*int64] // 地区请求

	// 网站访问数据
	domainStat *structure.Map[*int64]

	// 连接数数据
	connStatGw    *structure.Map[*int64]
	connStatFront *structure.Map[*int64]
)

func C() *bigcache.BigCache {
	return bc
}

func init() {
	cache, _ := bigcache.New(context.Background(), bigcache.Config{
		Shards:             64,
		LifeWindow:         48 * time.Hour,
		CleanWindow:        30 * time.Minute,
		MaxEntriesInWindow: 256,
		MaxEntrySize:       1024,
	})

	bc = cache
}

func initCacheFromFile() {
	// 初始化数据库
	db.GetDB().AutoMigrate(&model.StatModel{})
	db.GetDB().AutoMigrate(&model.GeoModel{})
	db.GetDB().AutoMigrate(&model.DomainModel{})
	m := LoadStat()
	if m != nil {
		atomic.StoreInt64(&total, m.MustGet("total"))
		atomic.StoreInt64(&api, m.MustGet("api"))
		atomic.StoreInt64(&static, m.MustGet("static"))
		atomic.StoreInt64(&fail, m.MustGet("fail"))
		atomic.StoreInt64(&today, m.MustGet("today"))
	}
	// 立即初始化一次
	go syncStat()

	geoIp = LoadGeoStat()
	go syncGEOStat()

	// 加载域名统计信息
	domainStat = LoadDomainStat()
	go syncDomainStat()
}
