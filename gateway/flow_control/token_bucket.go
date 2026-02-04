package flow

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// TokenBucketLimiter 令牌桶限流
type TokenBucketLimiter struct {
	limiter    *rate.Limiter
	lastAccess time.Time
	mu         sync.Mutex
}

func NewTokenBucketLimiter(limit int, window time.Duration) *TokenBucketLimiter {
	r := rate.Limit(float64(limit) / window.Seconds())
	return &TokenBucketLimiter{
		limiter:    rate.NewLimiter(r, limit),
		lastAccess: time.Now(),
	}
}

func (l *TokenBucketLimiter) Allow() bool {
	l.mu.Lock()
	l.lastAccess = time.Now()
	l.mu.Unlock()
	return l.limiter.Allow()
}

func (l *TokenBucketLimiter) LastAccess() time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.lastAccess
}
