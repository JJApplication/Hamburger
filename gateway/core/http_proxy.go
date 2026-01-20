package core

import (
	"Hamburger/internal/config"
	"Hamburger/internal/utils"
	"github.com/rs/zerolog"
	"net/http"
	"net/http/httputil"
	"time"
)

// NewHttpProxy 创建基于 net/http 的反向代理
func NewHttpProxy(cfg *config.Config, logger *zerolog.Logger) http.Handler {
	// 使用默认的Director, ModifyResponse, ErrorHandler
	proxy := &httputil.ReverseProxy{
		Director:       ProxyDirector(cfg, logger),
		Transport:      getOptimizedTransport("http"), // 使用http transport
		FlushInterval:  time.Duration(utils.DefaultInt64(cfg.CoreProxy.FlushInterval, FlushInterval)) * time.Millisecond,
		ErrorLog:       nil,
		BufferPool:     getBufferPool(utils.DefaultInt(cfg.CoreProxy.BufSize, BufferSize)),
		ModifyResponse: ProxyModifyResponse(cfg, logger),
		ErrorHandler:   ProxyErrorHandler(logger),
	}

	return proxy
}
