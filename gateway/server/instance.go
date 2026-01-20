package server

import (
	"Hamburger/internal/config"
	"github.com/rs/zerolog"
	"net"
	"net/http"
	"sync"
)

type ServerInstance struct {
	Name     string              // 服务器名称
	Server   *http.Server        // HTTP 服务器
	Listener net.Listener        // 网络监听器
	TLS      bool                // 是否启用 TLS
	Started  bool                // 是否已启动
	Error    error               // 启动错误
	Config   config.ServerConfig // 服务器配置
	Logger   *zerolog.Logger
	lock     sync.RWMutex
}
