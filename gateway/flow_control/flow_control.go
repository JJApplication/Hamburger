/*
Project: Sandwich flow_control.go
Created: 2021/12/14 by Landers
*/

package flow

import (
	"Hamburger/internal/config"
	"Hamburger/internal/logger"
	"fmt"
	"github.com/rs/zerolog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 流量控制
// 在访问的请求数超出限制时 禁止当前客户端请求
// 支持基于host、请求头的精准匹配和多时间范围流控

var flowController *FlowController

func InitLimiter() {
	flowController = NewFlowController()
	go flowController.CleanupExpiredRecords()
}

func GetLimiter() *FlowController {
	return flowController
}

// FlowController 流量控制器
type FlowController struct {
	logger        *zerolog.Logger
	globalLimiter *RateLimiter
	ruleLimiters  map[string]*RateLimiter
	mux           sync.RWMutex
	config        *config.FlowControlConfig
}

// RateLimiter 多时间窗口速率限制器
type RateLimiter struct {
	rules    []ParsedRule
	limiters map[string][]LimiterStrategy
	mux      sync.RWMutex
}

type ParsedRule struct {
	Requests int
	Window   time.Duration
	Mode     string
}

// LimiterStrategy 限流策略接口
type LimiterStrategy interface {
	Allow() bool
	LastAccess() time.Time
}

// FlowCheckResult 流控检查结果
type FlowCheckResult struct {
	Allowed     bool
	RuleName    string
	MatchedRule *config.FlowControlRule
	Reason      string
}

// NewFlowController 创建流量控制器
func NewFlowController() *FlowController {
	cfg := config.Get().Features.FlowControl
	fc := &FlowController{
		logger:       logger.L(),
		ruleLimiters: make(map[string]*RateLimiter),
		config:       &cfg,
	}

	// 初始化全局限流器
	if cfg.Enabled {
		fc.globalLimiter = fc.createRateLimiter([]config.RateLimit{cfg.GlobalLimit})

		// 初始化规则限流器
		for _, rule := range cfg.Rules {
			if rule.Enabled {
				fc.ruleLimiters[rule.Name] = fc.createRateLimiter(rule.Limits)
			}
		}
	}

	return fc
}

// createRateLimiter 创建速率限制器
func (fc *FlowController) createRateLimiter(limits []config.RateLimit) *RateLimiter {
	rl := &RateLimiter{
		rules:    make([]ParsedRule, 0, len(limits)),
		limiters: make(map[string][]LimiterStrategy),
	}

	for _, limit := range limits {
		duration, err := fc.parseDuration(limit.Window, limit.Unit)
		if err != nil {
			fc.logger.Error().Err(err).Str("Windows", limit.Window).Str("Unit", limit.Unit).
				Msg("Invalid duration format")
			continue
		}
		rl.rules = append(rl.rules, ParsedRule{
			Requests: limit.Requests,
			Window:   duration,
			Mode:     limit.Mode,
		})
	}

	return rl
}

// parseDuration 解析时间持续时间，支持s和min单位
func (fc *FlowController) parseDuration(window, unit string) (time.Duration, error) {
	value, err := strconv.Atoi(window)
	if err != nil {
		// 尝试从 window 中提取数字
		if strings.HasSuffix(window, "s") {
			valueStr := strings.TrimSuffix(window, "s")
			value, err = strconv.Atoi(valueStr)
			if err == nil {
				return time.Duration(value) * time.Second, nil
			}
		} else if strings.HasSuffix(window, "min") {
			valueStr := strings.TrimSuffix(window, "min")
			value, err = strconv.Atoi(valueStr)
			if err == nil {
				return time.Duration(value) * time.Minute, nil
			}
		}
		return 0, fmt.Errorf("invalid duration format: %s", window)
	}

	switch unit {
	case "s":
		return time.Duration(value) * time.Second, nil
	case "min":
		return time.Duration(value) * time.Minute, nil
	default:
		return 0, fmt.Errorf("unsupported time unit: %s", unit)
	}
}

// CheckRequest 检查请求是否被限流
func (fc *FlowController) CheckRequest(req *http.Request) *FlowCheckResult {
	if !fc.config.Enabled {
		return &FlowCheckResult{Allowed: true}
	}

	// 按优先级检查规则
	for _, rule := range fc.getSortedRules() {
		if !rule.Enabled {
			continue
		}

		if fc.matchRule(req, &rule) {
			limiter := fc.ruleLimiters[rule.Name]
			if limiter != nil {
				key := fc.generateKey(req, &rule)
				if !limiter.Allow(key) {
					return &FlowCheckResult{
						Allowed:     false,
						RuleName:    rule.Name,
						MatchedRule: &rule,
						Reason:      fmt.Sprintf("Rule '%s' rate limit exceeded", rule.Name),
					}
				}
			}
		}
	}

	// 检查全局限流
	if fc.globalLimiter != nil {
		key := fc.generateGlobalKey(req)
		if !fc.globalLimiter.Allow(key) {
			return &FlowCheckResult{
				Allowed: false,
				Reason:  "Global rate limit exceeded",
			}
		}
	}

	return &FlowCheckResult{Allowed: true}
}

// getSortedRules 获取按优先级排序的规则
func (fc *FlowController) getSortedRules() []config.FlowControlRule {
	rules := make([]config.FlowControlRule, len(fc.config.Rules))
	copy(rules, fc.config.Rules)

	// 按优先级排序，数字越小优先级越高
	for i := 0; i < len(rules)-1; i++ {
		for j := i + 1; j < len(rules); j++ {
			if rules[i].Priority > rules[j].Priority {
				rules[i], rules[j] = rules[j], rules[i]
			}
		}
	}

	return rules
}

// matchRule 匹配规则
func (fc *FlowController) matchRule(req *http.Request, rule *config.FlowControlRule) bool {
	switch rule.MatchType {
	case "host":
		return req.Host == rule.MatchValue || strings.Contains(req.Host, rule.MatchValue)
	case "header":
		headerValue := req.Header.Get(rule.HeaderKey)
		return headerValue == rule.MatchValue || strings.Contains(headerValue, rule.MatchValue)
	case "ip":
		clientIP := fc.getClientIP(req)
		return clientIP == rule.MatchValue
	default:
		return false
	}
}

// generateKey 生成限流key
func (fc *FlowController) generateKey(req *http.Request, rule *config.FlowControlRule) string {
	switch rule.MatchType {
	case "host":
		return fmt.Sprintf("rule:%s:host:%s", rule.Name, req.Host)
	case "header":
		headerValue := req.Header.Get(rule.HeaderKey)
		return fmt.Sprintf("rule:%s:header:%s:%s", rule.Name, rule.HeaderKey, headerValue)
	case "ip":
		clientIP := fc.getClientIP(req)
		return fmt.Sprintf("rule:%s:ip:%s", rule.Name, clientIP)
	default:
		return fmt.Sprintf("rule:%s:unknown", rule.Name)
	}
}

// generateGlobalKey 生成全局限流key
func (fc *FlowController) generateGlobalKey(req *http.Request) string {
	clientIP := fc.getClientIP(req)
	return fmt.Sprintf("global:ip:%s", clientIP)
}

// getClientIP 获取客户端IP
func (fc *FlowController) getClientIP(req *http.Request) string {
	// 检查 X-Forwarded-For
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// 检查 X-Real-IP
	if xri := req.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// 使用 RemoteAddr
	ip := req.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(key string) bool {
	rl.mux.Lock()
	defer rl.mux.Unlock()

	// 检查策略是否存在，不存在则创建
	strategies, exists := rl.limiters[key]
	if !exists {
		strategies = make([]LimiterStrategy, 0, len(rl.rules))
		for _, rule := range rl.rules {
			var strategy LimiterStrategy
			switch rule.Mode {
			case "fixed":
				strategy = NewFixedWindowLimiter(rule.Requests, rule.Window)
			case "leaky":
				strategy = NewLeakyBucketLimiter(rule.Requests, rule.Window)
			case "token":
				strategy = NewTokenBucketLimiter(rule.Requests, rule.Window)
			case "sliding":
				fallthrough
			default:
				strategy = NewSlidingWindowLimiter(rule.Requests, rule.Window)
			}
			strategies = append(strategies, strategy)
		}
		rl.limiters[key] = strategies
	}

	// 检查所有策略
	for _, strategy := range strategies {
		if !strategy.Allow() {
			return false
		}
	}

	return true
}

// CleanupExpiredRecords 定期清理过期记录
func (fc *FlowController) CleanupExpiredRecords() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		fc.mux.RLock()
		if fc.globalLimiter != nil {
			fc.globalLimiter.cleanup()
		}
		for _, limiter := range fc.ruleLimiters {
			limiter.cleanup()
		}
		fc.mux.RUnlock()
	}
}

// cleanup 清理限流器中的过期记录
func (rl *RateLimiter) cleanup() {
	rl.mux.Lock()
	defer rl.mux.Unlock()

	now := time.Now()

	maxWindow := time.Duration(0)
	for _, rule := range rl.rules {
		if rule.Window > maxWindow {
			maxWindow = rule.Window
		}
	}

	for key, strategies := range rl.limiters {
		lastAccess := time.Time{}
		for _, s := range strategies {
			la := s.LastAccess()
			if la.After(lastAccess) {
				lastAccess = la
			}
		}

		if now.Sub(lastAccess) > maxWindow {
			delete(rl.limiters, key)
		}
	}
}
