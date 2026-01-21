package utils

import (
	"Hamburger/internal/logger"
	"time"
)

// 时间性能指标

// PerformTime 包裹耗时函数
func PerformTime(f func()) (time.Time, time.Time, time.Duration) {
	start := time.Now()
	f()
	end := time.Now()
	elapsed := end.Sub(start)
	return start, end, elapsed
}

// 直接打印起始时间

func PerformCalc(event string, start time.Time) {
	end := time.Now()
	logger.GetLogger().Debug().Str("event", event).
		Str("start", start.Format(time.RFC3339Nano)).
		Str("end", end.Format(time.RFC3339Nano)).
		Dur("duration", end.Sub(start)).Msg("perform test")
}
