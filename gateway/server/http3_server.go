package server

import (
	"Hamburger/internal/config"
	"Hamburger/internal/utils"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/rs/zerolog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

// Server HTTP/3 服务器
// 基于 QUIC 协议实现的 HTTP/3 服务器
type Server struct {
	config   config.HTTP3Config // HTTP/3 配置
	server   *http3.Server      // HTTP/3 服务器实例
	listener *quic.Listener     // QUIC 监听器
	handler  http.Handler       // 请求处理器
	logger   *zerolog.Logger    // 日志记录器
	mu       sync.RWMutex       // 读写锁
	started  bool               // 是否已启动
	ctx      context.Context    // 上下文
	cancel   context.CancelFunc // 取消函数

	Name    string
	Address string
}

// NewHttp3Server 创建新的 HTTP/3 服务器
func NewHttp3Server(cfg config.HTTP3Config, handler http.Handler, logger *zerolog.Logger) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		config:  cfg,
		handler: handler,
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start 启动 HTTP/3 服务器
func (s *Server) Start(addr string, tlsConfig *tls.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("HTTP/3 服务器已经启动")
	}

	if !s.config.Enabled {
		return fmt.Errorf("HTTP/3 功能未启用")
	}

	// 创建 QUIC 配置
	quicConfig := s.createQUICConfig()

	// 创建 UDP 监听器
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("解析 UDP 地址失败: %v", err)
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("创建 UDP 监听器失败: %v", err)
	}

	// 创建 QUIC 监听器
	s.listener, err = quic.Listen(udpConn, tlsConfig, quicConfig)
	if err != nil {
		udpConn.Close()
		return fmt.Errorf("创建 QUIC 监听器失败: %v", err)
	}

	// 创建 HTTP/3 服务器
	s.server = &http3.Server{
		Handler:    s.handler,
		TLSConfig:  tlsConfig,
		QUICConfig: quicConfig,
	}

	// 启动服务器
	go s.serve()

	s.started = true
	s.logger.Printf("HTTP/3 服务器已启动，监听地址: %s", addr)

	return nil
}

// createQUICConfig 创建 QUIC 配置
func (s *Server) createQUICConfig() *quic.Config {
	return &quic.Config{
		// 最大连接数
		MaxIncomingStreams: int64(s.config.MaxConnections),
		// 空闲超时
		MaxIdleTimeout: utils.ToSecond(s.config.IdleTimeout),
		// 保活间隔
		KeepAlivePeriod: utils.ToSecond(s.config.KeepAlive),
		// 启用数据报
		EnableDatagrams: true,
	}
}

// serve 运行服务器
func (s *Server) serve() {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Printf("HTTP/3 服务器发生 panic: %v", r)
		}
	}()

	err := s.server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.logger.Printf("HTTP/3 服务器运行错误: %v", err)
	} else {
		s.logger.Printf("HTTP/3 服务器已停止")
	}
}

// Stop 停止 HTTP/3 服务器
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return fmt.Errorf("HTTP/3 服务器未启动")
	}

	s.logger.Printf("正在停止 HTTP/3 服务器...")

	// 创建停止超时上下文
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer stopCancel()

	// 优雅停止服务器
	if err := s.server.Shutdown(stopCtx); err != nil {
		s.logger.Printf("HTTP/3 服务器优雅停止失败: %v", err)
		// 强制关闭
		s.server.Close()
	}

	// 关闭监听器
	if s.listener != nil {
		s.listener.Close()
	}

	// 取消上下文
	s.cancel()

	s.started = false
	s.logger.Printf("HTTP/3 服务器已停止")

	return nil
}

// IsStarted 检查服务器是否已启动
func (s *Server) IsStarted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.started
}

// GetStats 获取服务器统计信息
func (s *Server) GetStats() *Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &Stats{
		Enabled:        s.config.Enabled,
		Started:        s.started,
		MaxConnections: s.config.MaxConnections,
		IdleTimeout:    utils.ToSecond(s.config.IdleTimeout),
		KeepAlive:      utils.ToSecond(s.config.KeepAlive),
	}

	// 如果服务器已启动，获取连接统计
	if s.started && s.listener != nil {
		// 注意：quic-go 可能不提供直接的连接统计接口
		// 这里可能需要自己维护连接计数
		stats.ActiveConnections = 0 // TODO: 实现连接计数
	}

	return stats
}

// Stats HTTP/3 服务器统计信息
type Stats struct {
	Enabled           bool          `json:"enabled"`            // 是否启用
	Started           bool          `json:"started"`            // 是否已启动
	MaxConnections    int           `json:"max_connections"`    // 最大连接数
	ActiveConnections int           `json:"active_connections"` // 当前活跃连接数
	IdleTimeout       time.Duration `json:"idle_timeout"`       // 空闲超时
	KeepAlive         time.Duration `json:"keep_alive"`         // 保活时间
}

// UpdateConfig 更新配置
func (s *Server) UpdateConfig(newConfig config.HTTP3Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查配置是否有变化
	if s.config.Enabled == newConfig.Enabled &&
		s.config.MaxConnections == newConfig.MaxConnections &&
		s.config.IdleTimeout == newConfig.IdleTimeout &&
		s.config.KeepAlive == newConfig.KeepAlive {
		return nil // 配置无变化
	}

	s.config = newConfig
	s.logger.Printf("HTTP/3 配置已更新")

	// 如果服务器正在运行且配置有重大变化，需要重启
	if s.started {
		s.logger.Printf("HTTP/3 配置变化，需要重启服务器")
		// 注意：这里不直接重启，而是返回错误让调用者决定
		return fmt.Errorf("HTTP/3 配置变化，需要重启服务器")
	}

	return nil
}

// SetHandler 设置请求处理器
func (s *Server) SetHandler(handler http.Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.handler = handler
	if s.server != nil {
		s.server.Handler = handler
	}
}

// GetConfig 获取当前配置
func (s *Server) GetConfig() config.HTTP3Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// HealthCheck 健康检查
func (s *Server) HealthCheck() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.config.Enabled {
		return fmt.Errorf("HTTP/3 功能未启用")
	}

	if !s.started {
		return fmt.Errorf("HTTP/3 服务器未启动")
	}

	if s.listener == nil {
		return fmt.Errorf("HTTP/3 监听器未初始化")
	}

	return nil
}

// GetListenAddr 获取监听地址
func (s *Server) GetListenAddr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.listener != nil {
		return s.listener.Addr().String()
	}

	return ""
}
