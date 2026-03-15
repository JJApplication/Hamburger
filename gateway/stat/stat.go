package stat

import (
	"encoding/binary"
	"fmt"
	"sync/atomic"
	"time"
)

// 指标统计

// 统计如下指标
// TotalRequest 总请求数
// APIRequest 后端请求数
// StaticRequest 前端请求数
// FailRequest 失败次数

const (
	Total = iota
	API
	Static
	Fail
	Today
)

// Add 后台异步的状态统计
func Add(tp int) {
	GetManager().Add(tp)
}

func (m *StatManager) Add(tp int) {
	go func() {
		cfg := m.getCfg()
		if !cfg.Stat.EnableStat {
			return
		}
		switch tp {
		case Total:
			m.addTotal()
		case API:
			m.addAPI()
		case Static:
			m.addStatic()
		case Fail:
			m.addFail()
		default:
			m.addTotal()
		}
	}()
}

// Get 从缓存中读取数据
func Get(tp int) int64 {
	return GetManager().Get(tp)
}

func (m *StatManager) Get(tp int) int64 {
	switch tp {
	case Total:
		totalStatByte, err := m.C().Get("total")
		if err != nil {
			return 0
		}
		return int64(binary.BigEndian.Uint64(totalStatByte))
	case API:
		apiStatByte, err := m.C().Get("api")
		if err != nil {
			return 0
		}
		return int64(binary.BigEndian.Uint64(apiStatByte))
	case Static:
		staticStatByte, err := m.C().Get("static")
		if err != nil {
			return 0
		}
		return int64(binary.BigEndian.Uint64(staticStatByte))
	case Fail:
		failStatByte, err := m.C().Get("fail")
		if err != nil {
			return 0
		}
		return int64(binary.BigEndian.Uint64(failStatByte))
	case Today:
		todayStatByte, err := m.C().Get("today")
		if err != nil {
			return 0
		}
		return int64(binary.BigEndian.Uint64(todayStatByte))
	default:
		return 0
	}
}

func (m *StatManager) addTotal() {
	atomic.AddInt64(&m.total, 1)
	m.addToday()
}

func (m *StatManager) addAPI() {
	atomic.AddInt64(&m.api, 1)
}

func (m *StatManager) addStatic() {
	atomic.AddInt64(&m.static, 1)
}

func (m *StatManager) addFail() {
	atomic.AddInt64(&m.fail, 1)
}

func (m *StatManager) addToday() {
	atomic.AddInt64(&m.today, 1)
}

func (m *StatManager) syncStat() {
	totalStat := atomic.LoadInt64(&m.total)
	apiStat := atomic.LoadInt64(&m.api)
	staticStat := atomic.LoadInt64(&m.static)
	failStat := atomic.LoadInt64(&m.fail)

	totalStatByte := make([]byte, 8)
	binary.BigEndian.PutUint64(totalStatByte, uint64(totalStat))
	apiByte := make([]byte, 8)
	binary.BigEndian.PutUint64(apiByte, uint64(apiStat))
	staticByte := make([]byte, 8)
	binary.BigEndian.PutUint64(staticByte, uint64(staticStat))
	failByte := make([]byte, 8)
	binary.BigEndian.PutUint64(failByte, uint64(failStat))

	m.C().Set("total", totalStatByte)
	m.C().Set("api", apiByte)
	m.C().Set("static", staticByte)
	m.C().Set("fail", failByte)

	// 对today特殊处理
	now := time.Now()
	todayDate := fmt.Sprintf("%d-%d-%d", now.Year(), now.Month(), now.Day())
	// 如果没有键则增加 同时删除旧键
	date, _ := m.C().Get("date")
	if todayDate == string(date) {
		// 同步数据
		todayStat := atomic.LoadInt64(&m.today)
		todayByte := make([]byte, 8)
		binary.BigEndian.PutUint64(todayByte, uint64(todayStat))
		m.C().Set("today", todayByte)
	} else {
		// 不存在数据
		atomic.StoreInt64(&m.today, 0)
		todayStat := atomic.LoadInt64(&m.today)
		todayByte := make([]byte, 8)
		binary.BigEndian.PutUint64(todayByte, uint64(todayStat))
		m.C().Set("today", todayByte)
		m.C().Set("date", []byte(todayDate))
	}
}
