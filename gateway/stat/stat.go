package stat

import (
	"Hamburger/internal/config"
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
	go func() {
		cfg := config.Get()
		if !cfg.Stat.EnableStat {
			return
		}
		switch tp {
		case Total:
			addTotal()
		case API:
			addAPI()
		case Static:
			addStatic()
		case Fail:
			addFail()
		default:
			addTotal()
		}
	}()
}

// Get 从缓存中读取数据
func Get(tp int) int64 {
	switch tp {
	case Total:
		totalStatByte, err := C().Get("total")
		if err != nil {
			return 0
		}
		return int64(binary.BigEndian.Uint64(totalStatByte))
	case API:
		apiStatByte, err := C().Get("api")
		if err != nil {
			return 0
		}
		return int64(binary.BigEndian.Uint64(apiStatByte))
	case Static:
		staticStatByte, err := C().Get("static")
		if err != nil {
			return 0
		}
		return int64(binary.BigEndian.Uint64(staticStatByte))
	case Fail:
		failStatByte, err := C().Get("fail")
		if err != nil {
			return 0
		}
		return int64(binary.BigEndian.Uint64(failStatByte))
	case Today:
		todayStatByte, err := C().Get("today")
		if err != nil {
			return 0
		}
		return int64(binary.BigEndian.Uint64(todayStatByte))
	default:
		return 0
	}
}

func addTotal() {
	atomic.AddInt64(&total, 1)
	addToday()
}

func addAPI() {
	atomic.AddInt64(&api, 1)
}

func addStatic() {
	atomic.AddInt64(&static, 1)
}

func addFail() {
	atomic.AddInt64(&fail, 1)
}

func addToday() {
	atomic.AddInt64(&today, 1)
}

func syncStat() {
	totalStat := atomic.LoadInt64(&total)
	apiStat := atomic.LoadInt64(&api)
	staticStat := atomic.LoadInt64(&static)
	failStat := atomic.LoadInt64(&fail)

	totalStatByte := make([]byte, 8)
	binary.BigEndian.PutUint64(totalStatByte, uint64(totalStat))
	apiByte := make([]byte, 8)
	binary.BigEndian.PutUint64(apiByte, uint64(apiStat))
	staticByte := make([]byte, 8)
	binary.BigEndian.PutUint64(staticByte, uint64(staticStat))
	failByte := make([]byte, 8)
	binary.BigEndian.PutUint64(failByte, uint64(failStat))

	C().Set("total", totalStatByte)
	C().Set("api", apiByte)
	C().Set("static", staticByte)
	C().Set("fail", failByte)

	// 对today特殊处理
	now := time.Now()
	todayDate := fmt.Sprintf("%d-%d-%d", now.Year(), now.Month(), now.Day())
	// 如果没有键则增加 同时删除旧键
	date, _ := C().Get("date")
	if todayDate == string(date) {
		// 同步数据
		todayStat := atomic.LoadInt64(&today)
		todayByte := make([]byte, 8)
		binary.BigEndian.PutUint64(todayByte, uint64(todayStat))
		C().Set("today", todayByte)
	} else {
		// 不存在数据
		atomic.StoreInt64(&today, 0)
		todayStat := atomic.LoadInt64(&today)
		todayByte := make([]byte, 8)
		binary.BigEndian.PutUint64(todayByte, uint64(todayStat))
		C().Set("today", todayByte)
		C().Set("date", []byte(todayDate))
	}
}
