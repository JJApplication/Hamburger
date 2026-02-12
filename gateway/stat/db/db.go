package db

import (
	"Hamburger/internal/config"
	"Hamburger/internal/logger"
	"sync"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	db      *gorm.DB
	mu      sync.Mutex
	enabled bool
)

// Init 初始化时序统计模块，从配置加载并连接数据库
// 成功后根据配置设置启用状态与时间间隔
func Init(cfg *config.Config) error {
	enabled = cfg.Stat.UseDB
	if !enabled || cfg.Stat.DBFile == "" || !cfg.Stat.EnableStat {
		return nil
	}

	var err error
	db, err = gorm.Open(sqlite.Open(cfg.Stat.DBFile), &gorm.Config{})
	if err != nil {
		logger.GetLogger().Error().Err(err).Msg("failed to open database")
		return err
	}
	return nil
}

// EnsureTable 保证当前间隔表已创建
func EnsureTable(table string, dst interface{}) {
	mu.Lock()
	defer mu.Unlock()
	// 存在则跳过
	if db.Migrator().HasTable(table) {
		return
	}
	if err := db.Table(table).AutoMigrate(dst); err != nil {
		logger.GetLogger().Error().Err(err).Str("table", table).Msg("failed to create table")
		return
	}
}

func GetDB() *gorm.DB {
	return db
}
