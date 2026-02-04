package prehandler

import (
	"Hamburger/internal/config"
	"net/http"
	"sync"
)

// HeaderSanitizer 请求头安全清理器
// 在转发前移除敏感或可伪造的头部
type HeaderSanitizer struct {
	enabled bool
	// deny 要剔除的敏感头部
	deny map[string]struct{}
	// keep 必须保留的内部/跟踪头部
	keep map[string]struct{}
}

var (
	sanitizer     *HeaderSanitizer
	sanitizerOnce sync.Once
)

// NewHeaderSanitizer 创建单例的HeaderSanitizer
func NewHeaderSanitizer() *HeaderSanitizer {
	sanitizerOnce.Do(func() {
		cf := config.Get()
		k := map[string]struct{}{
			cf.ProxyHeader.TraceId:            {},
			cf.ProxyHeader.FrontendHostHeader: {},
			cf.ProxyHeader.BackendHeader:      {},
			cf.ProxyHeader.ProxyApp:           {},
		}
		d := map[string]struct{}{
			"Authorization":       {},
			"Proxy-Authorization": {},
			"Cookie":              {},
			"X-Forwarded-For":     {},
			"X-Real-IP":           {},
			"X-Client-IP":         {},
			"Forwarded":           {},
			"X-Forwarded-Proto":   {},
			"X-Forwarded-Host":    {},
			"X-Forwarded-Port":    {},
			"X-Amzn-Trace-Id":     {},
			"X-Request-Id":        {},
			"CF-Connecting-IP":    {},
		}
		sanitizer = &HeaderSanitizer{enabled: cf.Middleware.Sanitizer.Enabled, deny: d, keep: k}
	})
	return sanitizer
}

// Access 执行请求头清理
func (h *HeaderSanitizer) Access(r *http.Request) {
	if r == nil {
		return
	}
	if !h.enabled {
		return
	}
	// 遍历并删除敏感头（保留keep）
	for name := range h.deny {
		if _, ok := h.keep[name]; ok {
			continue
		}
		r.Header.Del(name)
	}
}

func (h *HeaderSanitizer) Handle(r *http.Request) error {
	h.Access(r)
	return nil
}

func (h *HeaderSanitizer) Name() string {
	return "HeaderSanitizer"
}

func (h *HeaderSanitizer) Enabled() bool {
	return h.enabled
}
