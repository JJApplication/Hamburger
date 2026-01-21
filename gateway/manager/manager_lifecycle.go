package manager

import (
	"Hamburger/gateway/server"
	"Hamburger/gateway/tls"
	"Hamburger/internal/config"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// NewManager 创建新的服务器管理器
func NewManager(cfg *config.Config, logger *zerolog.Logger, handler http.Handler) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	tlsManager := tls.NewTLSManager(cfg, logger)
	m := &Manager{
		config:      cfg,
		servers:     make(map[string]*server.ServerInstance),
		http3Server: make(map[string]*server.Server),
		handler:     handler,
		ctx:         ctx,
		cancel:      cancel,
		logger:      logger,
	}
	tlsManager.RegisterBeforeAutoCert(m.beforeHandleAutoCert)
	tlsManager.RegisterBeforeAutoCert(m.afterHandleAutoCert)
	m.tlsManager = tlsManager

	return m
}

// Start 启动所有配置的服务器
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return fmt.Errorf("server manager already started")
	}

	// 获取启用的服务器配置
	enabledServers := GetEnabledServers(m.config)
	if len(enabledServers) == 0 {
		return fmt.Errorf("no enabled server configuration")
	}

	m.logger.Info().Msgf("starting %d server instances", len(enabledServers))

	// 启动每个服务器
	for _, serverConfig := range enabledServers {
		if err := m.startServer(serverConfig, m.logger); err != nil {
			m.logger.Error().Msgf("failed to start server %s: %v", serverConfig.Name, err)
			// 继续启动其他服务器，不因为一个失败而全部停止
			continue
		}
		m.logger.Info().Msgf("started server %s successfully, address: %s:%d", serverConfig.Name, serverConfig.Host, serverConfig.Port)

		// http3 server
		if m.config.Features.HTTP3.Enabled && serverConfig.Protocol == "https" {
			if err := m.startHttp3Server(m.config.Features.HTTP3, serverConfig, m.logger, m.handler); err != nil {
				m.logger.Error().Msgf("failed to start http3 server %s: %v", serverConfig.Name, err)
				continue
			}
			m.logger.Info().Msgf("started http3 server %s successfully, address: %s:%d", serverConfig.Name, serverConfig.Host, serverConfig.Port)
		}
	}

	// 检查是否至少有一个服务器启动成功
	if len(m.servers) == 0 {
		return fmt.Errorf("no servers started successfully")
	}

	m.started = true
	m.logger.Info().Msgf("server manager started, successfully started %d servers", len(m.servers))

	return nil
}

// startServer 启动单个服务器
func (m *Manager) startServer(serverConfig config.ServerConfig, logger *zerolog.Logger) error {
	instance, err := server.CommonHttpServer(m.config, serverConfig, logger, m.handler, m.tlsManager)
	if err != nil {
		return err
	}

	// 保存服务器实例
	m.servers[serverConfig.Name] = instance

	// 在 goroutine 中启动服务器
	m.wg.Add(1)
	go m.runServer(instance)

	return nil
}

func (m *Manager) startHttp3Server(cfg config.HTTP3Config, serverConfig config.ServerConfig, logger *zerolog.Logger, handler http.Handler) error {
	addr := fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port)
	http3Srv := server.NewHttp3Server(cfg, handler, logger)
	m.http3Server[serverConfig.Name] = http3Srv

	m.wg.Add(1)
	go func() {
		http3Srv.Name = serverConfig.Name
		http3Srv.Address = addr
		err := http3Srv.Start(addr, m.tlsManager.GetTlsConfig(serverConfig.TLS))
		if err != nil {
			m.logger.Error().Err(err).Str("server", serverConfig.Name).Msg("failed to start http3 service")
		}
	}()
	return nil
}

// runServer 运行服务器
func (m *Manager) runServer(instance *server.ServerInstance) {
	defer m.wg.Done()

	// 标记服务器已启动
	instance.Started = true

	// 启动服务器
	err := instance.Server.Serve(instance.Listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		instance.Error = err
		m.logger.Printf("server %s runtime error: %v", instance.Name, err)
	} else {
		m.logger.Printf("server %s stopped", instance.Name)
	}

	// 标记服务器已停止
	instance.Started = false
}

// Stop 停止所有服务器
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return fmt.Errorf("server manager not started")
	}

	m.logger.Printf("stopping %d server instances", len(m.servers))

	// 创建停止超时上下文
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer stopCancel()

	// 停止所有服务器
	var wg sync.WaitGroup
	for name, instance := range m.servers {
		wg.Add(1)
		go func(name string, instance *server.ServerInstance) {
			defer wg.Done()
			if err := instance.Server.Shutdown(stopCtx); err != nil {
				m.logger.Printf("failed to stop server %s: %v", name, err)
				// 强制关闭
				instance.Server.Close()
			} else {
				m.logger.Printf("server %s stopped gracefully", name)
			}
		}(name, instance)
	}

	// 等待所有服务器停止
	wg.Wait()

	// 取消上下文
	m.cancel()

	// 等待所有 goroutine 结束
	m.wg.Wait()

	// 清理状态
	m.servers = make(map[string]*server.ServerInstance)
	m.started = false

	m.logger.Printf("all servers stopped")
	return nil
}

// Restart 重启所有服务器
func (m *Manager) Restart() error {
	if err := m.Stop(); err != nil {
		return fmt.Errorf("failed to stop server: %v", err)
	}

	// 重新创建上下文
	m.ctx, m.cancel = context.WithCancel(context.Background())

	return m.Start()
}

// GetServerStatus 获取服务器状态
func (m *Manager) GetServerStatus() map[string]*server.ServerInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 创建副本以避免并发访问问题
	status := make(map[string]*server.ServerInstance)
	for name, instance := range m.servers {
		// 创建实例副本
		statusCopy := &server.ServerInstance{
			Name:    instance.Name,
			Config:  instance.Config,
			TLS:     instance.TLS,
			Started: instance.Started,
			Error:   instance.Error,
		}
		status[name] = statusCopy
	}

	return status
}

func (m *Manager) GetHttp3ServerStatus() map[string]*server.Server {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 创建副本以避免并发访问问题
	status := make(map[string]*server.Server)
	for name, instance := range m.http3Server {
		// 创建实例副本
		statusCopy := &server.Server{
			Name:    instance.Name,
			Address: instance.Address,
		}
		status[name] = statusCopy
	}

	return status
}

// beforeHandleAutoCert 在autocert发起证书申请/刷新前，确保80端口可用于HTTP-01挑战
func (m *Manager) beforeHandleAutoCert() error {
	// 查找当前占用80端口的HTTP服务器
	var http80 *server.ServerInstance
	m.mu.RLock()
	for _, inst := range m.servers {
		if inst.Config.Protocol == "http" && inst.Config.Port == 80 && inst.Started {
			http80 = inst
			break
		}
	}
	m.mu.RUnlock()

	// 如有占用则先停止
	if http80 != nil {
		m.logger.Info().Str("service", http80.Name).Msg("AutoTLS: port 80 occupied by server, stopping it first")
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := http80.Server.Shutdown(ctx); err != nil {
			m.logger.Error().Err(err).Msg("AutoTLS: failed to stop port 80 http server")
		}
		m.stoppedHTTP80 = http80
		// 从映射移除，避免状态混淆
		m.mu.Lock()
		delete(m.servers, http80.Name)
		m.mu.Unlock()
	}

	// 启动临时挑战服务器，监听80端口，仅用于处理 /.well-known/acme-challenge/
	addr := fmt.Sprintf("0.0.0.0:%d", 80)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("autotls: failed to create port 80 challenge listener: %v", err)
	}

	tempSrv := &http.Server{
		Addr:              addr,
		Handler:           m.tlsManager.AcmeMgr.HTTPHandler(nil),
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}
	m.autoCertTempServer = tempSrv
	go func() {
		if err := tempSrv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			m.logger.Error().Err(err).Msg("AutoTLS: challenge server runtime error")
		}
	}()
	m.logger.Info().Str("address", addr).Msg("AutoTLS: temporary challenge server started")
	return nil
}

// afterHandleAutoCert 在证书申请/刷新完成后，关闭挑战服务器并恢复原80端口HTTP服务
func (m *Manager) afterHandleAutoCert() error {
	// 关闭临时挑战服务器
	if m.autoCertTempServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = m.autoCertTempServer.Shutdown(ctx)
		m.autoCertTempServer = nil
		m.logger.Info().Msg("AutoTLS: temporary challenge server closed")
	}

	// 恢复原HTTP服务器（如存在）
	if m.stoppedHTTP80 != nil {
		m.logger.Info().Str("service", m.stoppedHTTP80.Name).Msg("AutoTLS: restarting original http server (port 80)")
		if err := m.startServer(m.stoppedHTTP80.Config, m.logger); err != nil {
			m.logger.Error().Err(err).Msg("AutoTLS: failed to restart port 80 http server")
		}
		m.stoppedHTTP80 = nil
	}
	return nil
}

// GetEnabledServers 获取所有启用的服务器配置
func GetEnabledServers(cf *config.Config) []config.ServerConfig {
	var enabled []config.ServerConfig
	for _, server := range cf.Servers {
		if server.Enabled {
			enabled = append(enabled, server)
		}
	}
	return enabled
}
