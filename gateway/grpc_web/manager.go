package grpc_web

import (
	"Hamburger/internal/config"
	"Hamburger/internal/logger"
	"sync"
)

var (
	globalProxy *GrpcWebProxy
	proxyOnce   sync.Once
)

func GetGrpcWebProxy() *GrpcWebProxy {
	proxyOnce.Do(func() {
		cfg := config.Get()
		if cfg.Features.GrpcProxy.Enabled {
			globalProxy = NewGrpcWebProxy(&cfg.Features.GrpcProxy)
			logger.GetLogger().Info().Int("hosts", len(cfg.Features.GrpcProxy.Hosts)).Msg("grpc-web proxy initialized with allowed hosts")
		} else {
			logger.GetLogger().Debug().Msg("grpc-web proxy is disabled")
		}
	})
	return globalProxy
}

func IsEnabled() bool {
	proxy := GetGrpcWebProxy()
	return proxy != nil && proxy.config.Enabled
}
