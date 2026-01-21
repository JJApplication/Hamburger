package sequence

import (
	"Hamburger/gateway/stat/db"
	"fmt"
	"strings"
	"sync"
	"time"

	"Hamburger/internal/config"
	"Hamburger/internal/logger"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SequenceRecord 时序记录表结构
// 每个时间间隔生成一张独立表，记录域名、路径、方法的请求次数
type SequenceRecord struct {
	ID        uint      `gorm:"primaryKey"`                     // 主键ID
	Domain    string    `gorm:"size:255;index:idx_dpm,unique"`  // 请求域名
	Path      string    `gorm:"size:1024;index:idx_dpm,unique"` // API路径
	Method    string    `gorm:"size:16;index:idx_dpm,unique"`   // 请求方法
	Count     int64     `gorm:"not null;default:0"`             // 请求次数
	UpdatedAt time.Time `gorm:"autoUpdateTime"`                 // 最近更新时间
}

type SequenceManger struct {
	enable   bool
	db       *gorm.DB
	interval time.Duration
	mutex    sync.Mutex
}

var sequence *SequenceManger

func InitSequenceManager(cfg *config.Config, db *gorm.DB) {
	sequence = &SequenceManger{
		enable:   cfg.Stat.Sequence.Enabled,
		interval: time.Duration(cfg.Stat.Sequence.Interval) * time.Second,
		mutex:    sync.Mutex{},
		db:       db,
	}
}

func SeqMgt() *SequenceManger {
	return sequence
}

func (seq *SequenceManger) IsEnabled() bool {
	return seq.enable && seq.db != nil
}

// RecordRequest 记录一次API请求
// 参数为：域名、API路径、请求方法
func (seq *SequenceManger) RecordRequest(domain, path, method string) {
	if !seq.enable || seq.db == nil {
		return
	}

	table := seq.tableNameFor(time.Now())
	db.EnsureTable(table, &SequenceRecord{})

	// 采用UPSERT自增计数
	seq.mutex.Lock()
	defer seq.mutex.Unlock()

	rec := &SequenceRecord{
		Domain:    domain,
		Path:      path,
		Method:    method,
		Count:     1,
		UpdatedAt: time.Now(),
	}

	err := seq.db.Table(table).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "domain"}, {Name: "path"}, {Name: "method"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"count": gorm.Expr("count + 1"), "updated_at": time.Now()}),
	}).Create(rec).Error
	if err != nil {
		logger.GetLogger().Error().Err(err).Msg("sequence: record failed")
	}
}

// QueryRequests 查询某时间段内API请求次数
// 可根据时间段、域名、路径、方法进行过滤，返回总计数
func (seq *SequenceManger) QueryRequests(from, to time.Time, domain, path, method string) (int64, error) {
	if !seq.enable || seq.db == nil {
		return 0, nil
	}
	if to.Before(from) {
		from, to = to, from
	}

	// 从起始桶到结束桶逐表汇总
	start := seq.bucketStart(from)
	end := seq.bucketStart(to)

	var total int64
	for t := start; !t.After(end); t = t.Add(seq.interval) {
		table := seq.tableNameFor(t)
		q := seq.db.Table(table).Model(&SequenceRecord{})
		if domain != "" {
			q = q.Where("domain = ?", domain)
		}
		if path != "" {
			q = q.Where("path = ?", path)
		}
		if method != "" {
			q = q.Where("method = ?", method)
		}
		var seg int64
		err := q.Select("SUM(count)").Scan(&seg).Error
		if err != nil {
			// 表不存在时跳过
			if strings.Contains(strings.ToLower(err.Error()), "no such table") {
				continue
			}
			return 0, err
		}
		total += seg
	}
	return total, nil
}

// bucketStart 返回指定时间所在间隔的起始时间
func (seq *SequenceManger) bucketStart(t time.Time) time.Time {
	return t.Truncate(seq.interval)
}

// tableNameFor 返回指定时间的表名
// 表名格式：api_sequence_YYYYMMDD_HHMM
func (seq *SequenceManger) tableNameFor(t time.Time) string {
	b := seq.bucketStart(t)
	return fmt.Sprintf("api_sequence_%s", b.Format("20060102_1504"))
}
