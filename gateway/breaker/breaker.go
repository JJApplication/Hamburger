/*
Package breaker 是网关的内存三态熔断器。

每个后端目标在内存中维护 Closed、Open、Half-Open 三种状态。
Closed 按配置窗口统计连续失败和失败率；达到阈值后进入 Open，
冷却结束后只放行有限的探测请求。链路表不落盘，也不使用后台定时器。
*/
package breaker

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"Hamburger/internal/config/core_config"
	"Hamburger/internal/config/loader"
	"Hamburger/internal/logger"
)

const (
	// StateClosed 表示正常转发，并持续统计失败。
	StateClosed State = "closed"
	// StateOpen 表示熔断打开，请求在到达后端前被拒绝。
	StateOpen State = "open"
	// StateHalfOpen 表示冷却结束，只允许少量探测请求。
	StateHalfOpen State = "half-open"
)

// ErrCircuitOpen 表示目标当前处于熔断打开或半开探测已满。
var ErrCircuitOpen = errors.New("circuit breaker is open")

// State 是熔断器对外展示的状态名。
type State string

// TargetSnapshot 是内存链路表中一条后端目标的只读副本。
type TargetSnapshot struct {
	Key                  string
	State                State
	Generation           uint64
	Total                int
	Failed               int
	ConsecutiveFailures  int
	ConsecutiveSuccesses int
	Inflight             int
	Expiry               time.Time
	WindowStart          time.Time
}

type outcome int

const (
	outcomeSuccess outcome = iota
	outcomeFailure
	outcomeIgnore
)

type counts struct {
	total                int
	failed               int
	consecutiveFailures  int
	consecutiveSuccesses int
}

type targetBreaker struct {
	mu          sync.Mutex
	key         string
	state       State
	generation  uint64
	expiry      time.Time
	windowStart time.Time
	inflight    int
	counts      counts
}

// Breaker 保存全部后端目标的熔断状态。
type Breaker struct {
	mu      sync.RWMutex
	targets map[string]*targetBreaker
	cfg     core_config.BreakConfig
	now     func() time.Time
}

var current atomic.Pointer[Breaker]

type targetContextKey struct{}

// WithTarget 把后端熔断键写入请求上下文。
func WithTarget(ctx context.Context, key string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, targetContextKey{}, key)
}

// TargetFrom 读取请求上下文中的后端熔断键。
func TargetFrom(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	key, ok := ctx.Value(targetContextKey{}).(string)
	if !ok || key == "" {
		return "", false
	}
	return key, true
}

// TargetKey 优先使用上下文中的后端服务名，否则回退到请求 URL 的 Host。
func TargetKey(req *http.Request) string {
	if req == nil {
		return ""
	}
	if key, ok := TargetFrom(req.Context()); ok {
		return key
	}
	if req.URL != nil {
		return req.URL.Host
	}
	return ""
}

// InitBreaker 用当前全局配置创建熔断器。配置缺失时使用默认值，不会启动后台任务。
func InitBreaker() {
	var cfg core_config.BreakConfig
	if loaded := loader.Get(); loaded != nil {
		cfg = loaded.Features.Break
	}
	current.Store(newBreaker(cfg))
}

// ReloadConfig 热更新熔断阈值。已有链路状态保留，下一次判定使用新配置。
func ReloadConfig(cfg core_config.BreakConfig) {
	normalized := cfg.Effective()
	b := current.Load()
	if b == nil {
		current.Store(newBreaker(normalized))
		return
	}
	b.mu.Lock()
	b.cfg = normalized
	b.mu.Unlock()
}

func newBreaker(cfg core_config.BreakConfig) *Breaker {
	cfg = cfg.Effective()
	return &Breaker{
		targets: make(map[string]*targetBreaker, cfg.Bucket),
		cfg:     cfg,
		now:     time.Now,
	}
}

func (b *Breaker) settings() core_config.BreakConfig {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.cfg
}

func (b *Breaker) clock() time.Time {
	b.mu.RLock()
	now := b.now
	b.mu.RUnlock()
	if now == nil {
		return time.Now()
	}
	return now()
}

func (b *Breaker) setNow(now func() time.Time) {
	b.mu.Lock()
	b.now = now
	b.mu.Unlock()
}

func (b *Breaker) target(key string) *targetBreaker {
	b.mu.RLock()
	item := b.targets[key]
	b.mu.RUnlock()
	if item != nil {
		return item
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if item = b.targets[key]; item != nil {
		return item
	}
	item = &targetBreaker{key: key, state: StateClosed}
	b.targets[key] = item
	return item
}

// Allow 判断目标是否可以访问后端。半开状态下会占用一个探测名额。
// 返回的 generation 必须原样传给 Report 或 Ignore。
func (b *Breaker) Allow(key string) (uint64, error) {
	return b.admit(key, true)
}

func (b *Breaker) admit(key string, consume bool) (uint64, error) {
	if b == nil || key == "" {
		return 0, nil
	}
	cfg := b.settings()
	if !cfg.IsEnabled() {
		return 0, nil
	}
	return b.target(key).admit(b.clock(), cfg, consume)
}

// Get 只读判断目标当前是否放行，不占用半开探测名额。
func (b *Breaker) Get(key string) bool {
	_, err := b.admit(key, false)
	return err == nil
}

// Set 记录一次后端失败。目标已熔断时返回 false。
func (b *Breaker) Set(key string) bool {
	generation, err := b.Allow(key)
	if err != nil || generation == 0 {
		return err == nil
	}
	b.Report(key, generation, false)
	return true
}

// Report 按 Allow 返回的代际号记录成功或失败。过期结果会被忽略。
func (b *Breaker) Report(key string, generation uint64, success bool) {
	if b == nil || key == "" || generation == 0 {
		return
	}
	result := outcomeFailure
	if success {
		result = outcomeSuccess
	}
	b.target(key).report(b.clock(), b.settings(), generation, result)
}

// Ignore 释放半开探测名额，不改变成功或失败计数。
func (b *Breaker) Ignore(key string, generation uint64) {
	if b == nil || key == "" || generation == 0 {
		return
	}
	b.target(key).report(b.clock(), b.settings(), generation, outcomeIgnore)
}

// Observe 根据一次上游调用结果更新熔断器。客户端主动取消不计入失败。
func (b *Breaker) Observe(key string, generation uint64, ctx context.Context, resp *http.Response, err error) {
	if b == nil || key == "" || generation == 0 {
		return
	}
	if canceled(ctx, err) {
		b.Ignore(key, generation)
		return
	}
	if err != nil || (resp != nil && b.isFailureStatus(resp.StatusCode)) {
		b.Report(key, generation, false)
		return
	}
	b.Report(key, generation, true)
}

func canceled(ctx context.Context, err error) bool {
	if err != nil && errors.Is(err, context.Canceled) {
		return true
	}
	return ctx != nil && errors.Is(ctx.Err(), context.Canceled)
}

// IsFailureStatus 判断后端状态码是否计入失败。
func (b *Breaker) IsFailureStatus(code int) bool {
	if b == nil {
		return false
	}
	return b.isFailureStatus(code)
}

func (b *Breaker) isFailureStatus(code int) bool {
	for _, status := range b.settings().TripStatusCodes {
		if status == code {
			return true
		}
	}
	return false
}

// Snapshot 返回内存链路表的稳定副本。
func (b *Breaker) Snapshot() []TargetSnapshot {
	if b == nil {
		return nil
	}
	b.mu.RLock()
	items := make([]*targetBreaker, 0, len(b.targets))
	for _, item := range b.targets {
		items = append(items, item)
	}
	b.mu.RUnlock()

	out := make([]TargetSnapshot, 0, len(items))
	for _, item := range items {
		out = append(out, item.snapshot())
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Key < out[j].Key
	})
	return out
}

// ResetTarget 删除一个后端目标的熔断状态。
func (b *Breaker) ResetTarget(key string) {
	if b == nil || key == "" {
		return
	}
	b.mu.Lock()
	delete(b.targets, key)
	b.mu.Unlock()
}

// ResetAll 清空内存链路表。
func (b *Breaker) ResetAll() {
	if b == nil {
		return
	}
	b.mu.Lock()
	b.targets = make(map[string]*targetBreaker, b.cfg.Bucket)
	b.mu.Unlock()
}

func (t *targetBreaker) admit(now time.Time, cfg core_config.BreakConfig, consume bool) (uint64, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.roll(now, cfg)
	switch t.state {
	case StateOpen:
		return t.generation, ErrCircuitOpen
	case StateHalfOpen:
		if t.inflight >= cfg.HalfOpenMaxRequests {
			return t.generation, ErrCircuitOpen
		}
		if consume {
			t.inflight++
		}
		return t.generation, nil
	default:
		t.state = StateClosed
		return t.generation, nil
	}
}

func (t *targetBreaker) report(now time.Time, cfg core_config.BreakConfig, generation uint64, result outcome) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if generation == 0 || generation != t.generation {
		return
	}
	switch t.state {
	case StateHalfOpen:
		if t.inflight > 0 {
			t.inflight--
		}
		switch result {
		case outcomeIgnore:
			return
		case outcomeFailure:
			t.toOpen(now, cfg)
		default:
			t.counts.total++
			t.counts.consecutiveSuccesses++
			t.counts.consecutiveFailures = 0
			if t.counts.consecutiveSuccesses >= cfg.SuccessThreshold {
				t.toClosed(now)
			}
		}
	case StateClosed:
		if result == outcomeIgnore {
			return
		}
		t.roll(now, cfg)
		if t.generation != generation || t.state != StateClosed {
			return
		}
		t.counts.total++
		if result == outcomeSuccess {
			t.counts.consecutiveFailures = 0
			t.counts.consecutiveSuccesses++
			return
		}
		t.counts.failed++
		t.counts.consecutiveFailures++
		t.counts.consecutiveSuccesses = 0
		if t.shouldTrip(cfg) {
			t.toOpen(now, cfg)
		}
	}
}

func (t *targetBreaker) roll(now time.Time, cfg core_config.BreakConfig) {
	if t.generation == 0 {
		t.generation = 1
		t.state = StateClosed
		t.windowStart = now
	}
	switch t.state {
	case StateOpen:
		if !t.expiry.IsZero() && !now.Before(t.expiry) {
			t.toHalfOpen(now)
		}
	case StateClosed:
		window := time.Duration(cfg.Reset) * time.Second
		if window > 0 && !t.windowStart.IsZero() && now.Sub(t.windowStart) >= window {
			t.counts = counts{}
			t.windowStart = now
		}
	}
}

func (t *targetBreaker) shouldTrip(cfg core_config.BreakConfig) bool {
	if cfg.MaxError > 0 && t.counts.consecutiveFailures >= cfg.MaxError {
		return true
	}
	if cfg.FailureRatio > 0 && t.counts.total >= cfg.MinRequests && t.counts.total > 0 {
		ratio := float64(t.counts.failed) / float64(t.counts.total)
		if ratio >= cfg.FailureRatio {
			return true
		}
	}
	return false
}

func (t *targetBreaker) toOpen(now time.Time, cfg core_config.BreakConfig) {
	reset := time.Duration(cfg.Reset) * time.Second
	if reset <= 0 {
		reset = time.Duration(core_config.DefaultBreakResetSeconds) * time.Second
	}
	t.state = StateOpen
	t.generation++
	t.inflight = 0
	t.counts = counts{}
	t.expiry = now.Add(reset)
	t.windowStart = now
	logger.GetLogger().Info().Str("target", t.key).Time("expiry", t.expiry).Msg("breaker opened")
}

func (t *targetBreaker) toHalfOpen(now time.Time) {
	t.state = StateHalfOpen
	t.generation++
	t.inflight = 0
	t.counts = counts{}
	t.expiry = time.Time{}
	t.windowStart = now
	logger.GetLogger().Info().Str("target", t.key).Msg("breaker half-open")
}

func (t *targetBreaker) toClosed(now time.Time) {
	t.state = StateClosed
	t.generation++
	t.inflight = 0
	t.counts = counts{}
	t.expiry = time.Time{}
	t.windowStart = now
	logger.GetLogger().Info().Str("target", t.key).Msg("breaker closed")
}

func (t *targetBreaker) snapshot() TargetSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	state := t.state
	if state == "" {
		state = StateClosed
	}
	return TargetSnapshot{
		Key:                  t.key,
		State:                state,
		Generation:           t.generation,
		Total:                t.counts.total,
		Failed:               t.counts.failed,
		ConsecutiveFailures:  t.counts.consecutiveFailures,
		ConsecutiveSuccesses: t.counts.consecutiveSuccesses,
		Inflight:             t.inflight,
		Expiry:               t.expiry,
		WindowStart:          t.windowStart,
	}
}

func active() *Breaker {
	return current.Load()
}

// Allow 使用进程内熔断器判断目标。熔断器未初始化时放行。
func Allow(key string) (uint64, error) {
	b := active()
	if b == nil {
		return 0, nil
	}
	return b.Allow(key)
}

// Get 使用进程内熔断器做只读判断。
func Get(key string) bool {
	b := active()
	if b == nil {
		return true
	}
	return b.Get(key)
}

// Set 向进程内熔断器记录一次失败。
func Set(key string) bool {
	b := active()
	if b == nil {
		return true
	}
	return b.Set(key)
}

// Report 向进程内熔断器上报结果。
func Report(key string, generation uint64, success bool) {
	if b := active(); b != nil {
		b.Report(key, generation, success)
	}
}

// Ignore 释放进程内熔断器的半开探测名额。
func Ignore(key string, generation uint64) {
	if b := active(); b != nil {
		b.Ignore(key, generation)
	}
}

// Observe 按上游结果更新进程内熔断器。
func Observe(key string, generation uint64, ctx context.Context, resp *http.Response, err error) {
	if b := active(); b != nil {
		b.Observe(key, generation, ctx, resp, err)
	}
}

// IsFailureStatus 使用进程内配置判断状态码。
func IsFailureStatus(code int) bool {
	b := active()
	if b == nil {
		for _, status := range core_config.DefaultBreakTripStatusCodes() {
			if status == code {
				return true
			}
		}
		return false
	}
	return b.IsFailureStatus(code)
}

// Snapshot 返回进程内内存链路表。
func Snapshot() []TargetSnapshot {
	b := active()
	if b == nil {
		return nil
	}
	return b.Snapshot()
}

// ResetTarget 删除进程内一个后端目标的状态。
func ResetTarget(key string) {
	if b := active(); b != nil {
		b.ResetTarget(key)
	}
}

// ResetAll 清空进程内内存链路表。
func ResetAll() {
	if b := active(); b != nil {
		b.ResetAll()
	}
}
