package grpc_proxy

import (
	"Hamburger/internal/config"
	"github.com/rs/zerolog"
	"sync"
)

var (
	globalProxy *GrpcProxy
	proxyOnce   sync.Once
)

// GetGrpcProxy 获取全局gRPC代理实例
func GetGrpcProxy() *GrpcProxy {
	return globalProxy
}

func InitGrpcProxy(cfg *config.GrpcProxyConfig, logger *zerolog.Logger) {
	proxyOnce.Do(func() {
		if cfg.Enabled {
			globalProxy = NewGrpcProxy(cfg, logger)
			logger.Info().Int("hosts", len(cfg.Hosts)).Msg("gRPC proxy initialized with allowed hosts")
		} else {
			logger.Debug().Msg("gRPC proxy is disabled")
		}
	})
}

// IsEnabled 检查gRPC代理是否启用
func IsEnabled() bool {
	proxy := GetGrpcProxy()
	return proxy != nil && proxy.config.Enabled
}
