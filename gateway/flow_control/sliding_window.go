package flow

import (
	"sync"
	"time"
)

// SlidingWindowLimiter 滑动窗口限流
type SlidingWindowLimiter struct {
	limit      int
	window     time.Duration
	records    []time.Time
	lastAccess time.Time
	mu         sync.Mutex
}

func NewSlidingWindowLimiter(limit int, window time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		limit:      limit,
		window:     window,
		records:    make([]time.Time, 0),
		lastAccess: time.Now(),
	}
}

func (l *SlidingWindowLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	l.lastAccess = now

	cutoff := now.Add(-l.window)
	validIdx := 0
	for i, t := range l.records {
		if t.After(cutoff) {
			validIdx = i
			break
		}
		validIdx = i + 1
	}
	if validIdx > 0 {
		l.records = l.records[validIdx:]
	}

	if len(l.records) < l.limit {
		l.records = append(l.records, now)
		return true
	}
	return false
}

func (l *SlidingWindowLimiter) LastAccess() time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.lastAccess
}
