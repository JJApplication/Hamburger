package core

import (
	"Hamburger/internal/config"
	"Hamburger/internal/utils"
	"github.com/rs/zerolog"
	"net/http"
	"net/http/httputil"
	"time"
)

// NewFastProxy 创建基于 fasthttp 的反向代理
// 使用 httputil.ReverseProxy 结构，但 Transport 替换为 fasthttp 实现
func NewFastProxy(cfg *config.Config, logger *zerolog.Logger) http.Handler {
	proxy := &httputil.ReverseProxy{
		Director:       ProxyDirector(cfg, logger),
		Transport:      getOptimizedTransport("fasthttp"), // 使用fasthttp transport
		FlushInterval:  time.Duration(utils.DefaultInt64(cfg.CoreProxy.FlushInterval, FlushInterval)) * time.Millisecond,
		ErrorLog:       nil,
		BufferPool:     getBufferPool(utils.DefaultInt(cfg.CoreProxy.BufSize, BufferSize)),
		ModifyResponse: ProxyModifyResponse(cfg, logger),
		ErrorHandler:   ProxyErrorHandler(logger),
	}

	return proxy
}
