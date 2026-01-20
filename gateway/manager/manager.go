package manager

import (
	"Hamburger/gateway/server"
	"Hamburger/gateway/tls"
	"Hamburger/internal/config"
	"context"
	"github.com/rs/zerolog"
	"net/http"
	"sync"
)

// Manager 服务器管理器
// 负责管理多个服务器实例的启动、停止和监控
type Manager struct {
	stoppedHTTP80      *server.ServerInstance // 刷新前被停止的80端口HTTP服务器，用于刷新后恢复
	autoCertTempServer *http.Server           // 刷新证书时临时占用80端口的HTTP服务器
	config             *config.Config
	servers            map[string]*server.ServerInstance // 服务器实例映射
	http3Server        map[string]*server.Server         // http3服务器
	handler            http.Handler                      // 请求处理器
	mu                 sync.RWMutex                      // 读写锁
	ctx                context.Context                   // 上下文
	cancel             context.CancelFunc                // 取消函数
	wg                 sync.WaitGroup                    // 等待组
	started            bool                              // 是否已启动
	logger             *zerolog.Logger                   // 日志记录器
	tlsManager         *tls.TLSManager
}
