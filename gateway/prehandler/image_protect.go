package prehandler

import (
	"Hamburger/internal/config"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// 图片防盗链
// 对纯后端服务返回图片资源时做保护, 对于纯前端服务需要在前端服务Helios中处理

type ImageProtect struct {
	enabled bool
	mu      sync.RWMutex // 读锁
	mime    []string
	allow   []string
}

var imageProtect *ImageProtect

func NewImageProtectModifier() *ImageProtect {
	sync.OnceFunc(func() {
		cfg := config.Get()
		imageProtect = &ImageProtect{
			enabled: cfg.Middleware.ImageProtect.Enabled,
			mime:    cfg.Middleware.ImageProtect.ImageType,
			allow:   cfg.Middleware.ImageProtect.AllowReferer,
		}
	})()

	return imageProtect
}

func (i *ImageProtect) Handle(r *http.Request) error {
	if !i.enabled {
		return nil
	}
	// 首先判断是否为图片
	referer := r.Header.Get("Referer")
	contentType := r.Header.Get("Content-Type")

	if !i.isValid(contentType, referer) {
		return errors.New("access denied")
	}

	return nil
}

func (i *ImageProtect) Name() string {
	return "ImageProtect"
}

func (i *ImageProtect) Enabled() bool {
	return i.enabled
}

func isInSlice(src string, list []string) bool {
	for _, v := range list {
		if v == src {
			return true
		}
	}

	return false
}

func (i *ImageProtect) isValid(contentType, referer string) bool {
	baseType := strings.ToLower(strings.Split(contentType, ";")[0])
	if !isInSlice(baseType, i.mime) {
		return false
	}

	parseUrl, err := url.Parse(referer)
	if err != nil {
		return false
	}
	baseReferer := parseUrl.Hostname()

	if !isInSlice(baseReferer, i.allow) {
		return false
	}

	return true
}
