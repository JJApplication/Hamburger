package svr

import (
	"Hamburger/internal/config"
	"Hamburger/internal/config/backproxy_config"
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// TcpProxyServer TCP 透明代理服务器
type TcpProxyServer struct {
	cfg         *config.Config
	logger      *zerolog.Logger
	host        string
	port        int
	backendConf backproxy_config.BackendServer
	listener    net.Listener
	stopCh      chan struct{}
	stopOnce    sync.Once
	started     bool
}

// NewTcpProxyServer 创建TCP代理服务器
func NewTcpProxyServer(cfg *config.Config, logger *zerolog.Logger, backendConf backproxy_config.BackendServer) *TcpProxyServer {
	return &TcpProxyServer{
		cfg:         cfg,
		logger:      logger,
		host:        backendConf.Host,
		port:        backendConf.Port,
		backendConf: backendConf,
		stopCh:      make(chan struct{}),
	}
}

// Start 启动TCP代理服务器
func (s *TcpProxyServer) Start() {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		s.logger.Error().Err(err).Str("address", addr).Msg("tcp proxy listen error")
		return
	}
	s.listener = ln
	go s.acceptLoop()
	s.started = true
}

// Stop 停止TCP代理服务器
func (s *TcpProxyServer) Stop() error {
	s.started = false
	if s.listener != nil {
		s.stopOnce.Do(func() {
			close(s.stopCh)
		})
		return s.listener.Close()
	}
	return nil
}

// IsStarted 返回是否已启动
func (s *TcpProxyServer) IsStarted() bool {
	return s.started
}

// Name 返回服务名
func (s *TcpProxyServer) Name() string {
	return s.backendConf.ServiceName
}

// GetAddr 返回监听地址
func (s *TcpProxyServer) GetAddr() string {
	return fmt.Sprintf("%s:%d", s.host, s.port)
}

func (s *TcpProxyServer) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopCh:
				return
			default:
				s.logger.Error().Err(err).Msg("tcp proxy accept error")
				continue
			}
		}
		go s.handleConn(conn)
	}
}

func (s *TcpProxyServer) handleConn(client net.Conn) {
	defer client.Close()
	target := s.backendConf.Tcp.Target
	if target == "" {
		return
	}
	timeout := time.Duration(s.backendConf.Tcp.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 0
	}

	server, err := net.DialTimeout("tcp", target, s.dialTimeout())
	if err != nil {
		s.logger.Error().Err(err).Str("target", target).Msg("tcp proxy dial error")
		return
	}
	defer server.Close()

	if timeout > 0 {
		_ = client.SetDeadline(time.Now().Add(timeout))
		_ = server.SetDeadline(time.Now().Add(timeout))
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = s.copyWithLimit(server, client, s.backendConf.Tcp.MaxBody)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(client, server)
	}()

	wg.Wait()
}

func (s *TcpProxyServer) copyWithLimit(dst io.Writer, src io.Reader, max int64) (int64, error) {
	if max <= 0 {
		return io.Copy(dst, src)
	}
	limited := &io.LimitedReader{R: src, N: max}
	n, err := io.Copy(dst, limited)
	if err != nil {
		return n, err
	}
	if limited.N == 0 {
		return n, context.Canceled
	}
	return n, nil
}

func (s *TcpProxyServer) dialTimeout() time.Duration {
	if s.backendConf.Tcp.Timeout <= 0 {
		return 0
	}
	return time.Duration(s.backendConf.Tcp.Timeout) * time.Second
}
