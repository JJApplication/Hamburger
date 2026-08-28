package db

import (
	"Hamburger/internal/config"
	"Hamburger/internal/logger"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	db      *gorm.DB
	mu      sync.RWMutex
	enabled bool
)

// Init 初始化累计统计与历史统计共享的 SQLite 连接。
//
// UseDB 只控制旧的累计 Stat/Geo/Domain 数据是否使用数据库；历史统计
// 只要 sequence.enabled 且配置了 db_file，就必须初始化同一个 SQLite 文件。
func Init(cfg *config.Config) error {
	if cfg == nil {
		return nil
	}
	shouldOpen := cfg.Stat.UseDB || cfg.Stat.Sequence.Enabled
	enabled = shouldOpen && cfg.Stat.DBFile != ""
	if !enabled {
		return nil
	}
	if dir := filepath.Dir(cfg.Stat.DBFile); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create stat database directory: %w", err)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if db != nil {
		return nil
	}

	var err error
	// sqlite3 accepts these URI parameters and applies them to every opened
	// connection. PRAGMA is repeated below for drivers that ignore URI args.
	db, err = gorm.Open(sqlite.Open(cfg.Stat.DBFile+"?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL"), &gorm.Config{})
	if err != nil {
		logger.GetLogger().Error().Err(err).Msg("failed to open database")
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		db = nil
		return err
	}
	// Readers may run alongside the single history writer while WAL is active.
	sqlDB.SetMaxOpenConns(8)
	sqlDB.SetMaxIdleConns(8)
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA auto_vacuum=INCREMENTAL",
	} {
		if err := db.Exec(pragma).Error; err != nil {
			logger.GetLogger().Warn().Err(err).Str("pragma", pragma).Msg("stat database pragma failed")
		}
	}
	return nil
}

// EnsureTable 保证当前间隔表已创建
func EnsureTable(table string, dst interface{}) {
	mu.Lock()
	defer mu.Unlock()
	if db == nil {
		return
	}
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
	mu.RLock()
	defer mu.RUnlock()
	return db
}

// Close 关闭共享 SQLite 连接。应用优雅退出时由 StatManager 在刷盘后调用。
func Close() error {
	mu.Lock()
	defer mu.Unlock()
	if db == nil {
		enabled = false
		return nil
	}
	sqlDB, err := db.DB()
	db = nil
	enabled = false
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
