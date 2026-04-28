package server

import (
	"Hamburger/internal/config/core_config"
	"context"
	"net"
	"sync"

	"github.com/rs/zerolog"
)

type serverRunner interface {
	Serve(net.Listener) error
	Shutdown(context.Context) error
	Close() error
}

type ServerInstance struct {
	Name     string                   // 服务器名称
	Server   serverRunner             // HTTP 服务器
	Listener net.Listener             // 网络监听器
	TLS      bool                     // 是否启用 TLS
	Started  bool                     // 是否已启动
	Error    error                    // 启动错误
	Config   core_config.ServerConfig // 服务器配置
	Logger   *zerolog.Logger
	lock     sync.RWMutex
}
