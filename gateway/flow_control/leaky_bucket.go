package flow

import (
	"sync"
	"time"
)

// LeakyBucketLimiter 漏桶限流
type LeakyBucketLimiter struct {
	capacity   float64
	rate       float64
	water      float64
	lastLeak   time.Time
	lastAccess time.Time
	mu         sync.Mutex
}

func NewLeakyBucketLimiter(limit int, window time.Duration) *LeakyBucketLimiter {
	r := float64(limit) / window.Seconds()
	return &LeakyBucketLimiter{
		capacity:   float64(limit),
		rate:       r,
		water:      0,
		lastLeak:   time.Now(),
		lastAccess: time.Now(),
	}
}

func (l *LeakyBucketLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	l.lastAccess = now

	elapsed := now.Sub(l.lastLeak).Seconds()
	leaked := elapsed * l.rate
	if leaked > 0 {
		l.water = l.water - leaked
		if l.water < 0 {
			l.water = 0
		}
		l.lastLeak = now
	}

	if l.water+1 <= l.capacity {
		l.water++
		return true
	}
	return false
}

func (l *LeakyBucketLimiter) LastAccess() time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.lastAccess
}
