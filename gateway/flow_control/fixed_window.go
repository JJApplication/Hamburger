package flow

import (
	"sync"
	"time"
)

// FixedWindowLimiter 固定窗口限流
type FixedWindowLimiter struct {
	limit      int
	window     time.Duration
	count      int
	resetTime  time.Time
	lastAccess time.Time
	mu         sync.Mutex
}

func NewFixedWindowLimiter(limit int, window time.Duration) *FixedWindowLimiter {
	return &FixedWindowLimiter{
		limit:      limit,
		window:     window,
		resetTime:  time.Now().Add(window),
		lastAccess: time.Now(),
	}
}

func (l *FixedWindowLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.lastAccess = time.Now()
	if time.Now().After(l.resetTime) {
		l.count = 0
		l.resetTime = time.Now().Add(l.window)
	}

	if l.count < l.limit {
		l.count++
		return true
	}
	return false
}

func (l *FixedWindowLimiter) LastAccess() time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.lastAccess
}
