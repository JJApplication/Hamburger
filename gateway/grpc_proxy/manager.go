package grpc_proxy

import (
	"Hamburger/internal/config"
	"Hamburger/internal/logger"
	"sync"
)

var (
	globalProxy *GrpcProxy
	proxyOnce   sync.Once
)

// GetGrpcProxy 获取全局gRPC代理实例
func GetGrpcProxy() *GrpcProxy {
	proxyOnce.Do(func() {
		cfg := config.Get()
		if cfg.Features.GrpcProxy.Enabled {
			globalProxy = NewGrpcProxy(&cfg.Features.GrpcProxy)
			logger.GetLogger().Info().Int("hosts", len(cfg.Features.GrpcProxy.Hosts)).Msg("gRPC proxy initialized with allowed hosts")
		} else {
			logger.GetLogger().Debug().Msg("gRPC proxy is disabled")
		}
	})
	return globalProxy
}

// IsEnabled 检查gRPC代理是否启用
func IsEnabled() bool {
	proxy := GetGrpcProxy()
	return proxy != nil && proxy.config.Enabled
}
