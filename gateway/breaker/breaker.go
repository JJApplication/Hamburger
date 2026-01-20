/*
Create: 2022/9/7
Project: Sandwich
Github: https://github.com/landers1037
Copyright Renj
*/

// Package breaker
// 自动熔断器
package breaker

import (
	"Hamburger/internal/config"
	"Hamburger/internal/logger"
	"Hamburger/internal/structure"
	"Hamburger/internal/utils"
	"time"
)

const (
	DefaultMaxError = 5
	DefaultBucket   = 10
)

// 熔断控制器
// 在需要转发的微服务返回大量失败时，直接熔断当前的连接请求禁止客户端访问

var breaker *Breaker

func InitBreaker() {
	breaker = NewBreaker()
	go breaker.Reset()
}

type BreakerBucket struct {
	errorConn int
	bucket    chan int
}

type Breaker struct {
	serviceBucket *structure.Map[*BreakerBucket]
	cf            *config.BreakConfig
}

func NewBreaker() *Breaker {
	return &Breaker{
		serviceBucket: structure.NewSizeMap[*BreakerBucket](DefaultBucket),
		cf:            &config.Get().Features.Break,
	}
}

func (b *Breaker) Get(domain string) bool {
	sb, ok := b.serviceBucket.Get(domain)
	if !ok {
		b.add(domain)
		return true
	}
	if len(sb.bucket) >= sb.errorConn {
		logger.GetLogger().Info().Str("domain", domain).Msg("breaker now is broken")
		return false
	}
	return true
}

func (b *Breaker) Set(domain string) bool {
	sb, ok := b.serviceBucket.Get(domain)
	if !ok {
		return true
	}
	if len(sb.bucket) < sb.errorConn {
		sb.bucket <- 1
		return true
	}

	return false
}

func (b *Breaker) add(domain string) {
	b.serviceBucket.Put(domain, &BreakerBucket{
		errorConn: utils.DefaultInt(b.cf.MaxError, DefaultMaxError),
		bucket:    make(chan int, utils.DefaultInt(b.cf.Bucket, DefaultBucket)),
	})
}

// Reset 自定重置
func (b *Breaker) Reset() {
	ticker := time.Tick(time.Duration(b.cf.Reset) * time.Second)
	for range ticker {
		b.serviceBucket.Range(func(key string, value *BreakerBucket) bool {
			value.bucket = make(chan int, b.cf.Bucket)
			logger.GetLogger().Info().Str("domain", key).Msg("breaker now is reset")
			return true
		})
	}
}

func Get(domain string) bool {
	return breaker.Get(domain)
}

func Set(domain string) bool {
	return breaker.Set(domain)
}
