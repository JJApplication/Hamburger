package traversal

import (
	"Hamburger/internal/config"
	"Hamburger/internal/config/exp_config"
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	kcp "github.com/xtaci/kcp-go/v5"
)

const (
	defaultTraversalHost = "0.0.0.0"
	defaultTraversalPort = 19090
	tunnelJoinTimeout    = 10 * time.Second
	relayIdleTimeout     = 90 * time.Second
	relayBufferSize      = 32 * 1024
)

var relayBufferPool = sync.Pool{
	New: func() any {
		buf := make([]byte, relayBufferSize)
		return &buf
	},
}

type pendingTunnel struct {
	publicConn net.Conn
	joinCh     chan net.Conn
}

type proxyListener struct {
	name       string
	remotePort int
	listener   net.Listener
}

type registeredClient struct {
	conn   net.Conn
	writer *bufio.Writer
	mu     sync.Mutex

	lastSeenUnixNano int64
}

// Server 内网穿透服务端。
type Server struct {
	cfg              exp_config.TraversalServerConfig
	logger           *zerolog.Logger
	controlListeners []net.Listener
	proxyListeners   map[int]*proxyListener
	proxyMu          sync.Mutex

	clientMu sync.RWMutex
	client   *registeredClient

	pendingMu sync.Mutex
	pending   map[string]*pendingTunnel

	idSeed  uint64
	stopCh  chan struct{}
	stopWg  sync.WaitGroup
	stopMux sync.Once
}

// NewServer 创建内网穿透服务端。
func NewServer(cfg *config.Config, logger *zerolog.Logger) *Server {
	serverCfg := cfg.ExpConfig.Traversal
	if strings.TrimSpace(serverCfg.ListenHost) == "" {
		serverCfg.ListenHost = defaultTraversalHost
	}
	if serverCfg.ListenPort <= 0 {
		serverCfg.ListenPort = defaultTraversalPort
	}
	serverCfg.Protocol = strings.ToLower(strings.TrimSpace(serverCfg.Protocol))
	if serverCfg.Protocol == "" {
		serverCfg.Protocol = "tcp"
	}
	if serverCfg.KCP.HeartbeatInterval == 0 {
		// 默认 20s 心跳，适合移动网络保持活性；<=0 可显式关闭。
		serverCfg.KCP.HeartbeatInterval = 20
	}
	if serverCfg.KCP.HeartbeatTimeout == 0 {
		// 默认 60s 超时，避免网络抖动造成误判。
		serverCfg.KCP.HeartbeatTimeout = 60
	}
	return &Server{
		cfg:              serverCfg,
		logger:           logger,
		controlListeners: make([]net.Listener, 0, 2),
		pending:          make(map[string]*pendingTunnel),
		proxyListeners:   make(map[int]*proxyListener),
		stopCh:           make(chan struct{}),
	}
}

// Start 启动内网穿透服务端。
func (s *Server) Start() error {
	if !s.cfg.Enabled {
		return nil
	}
	if strings.TrimSpace(s.cfg.AuthKey) == "" {
		return errors.New("traversal auth_key is required")
	}
	controlAddr := net.JoinHostPort(s.cfg.ListenHost, strconv.Itoa(s.cfg.ListenPort))

	hasTCP, hasKCP, err := s.parseProtocols()
	if err != nil {
		return err
	}

	var controlListeners []net.Listener
	if hasTCP {
		ln, err := listenTCP(controlAddr)
		if err != nil {
			return fmt.Errorf("listen tcp %s failed: %w", controlAddr, err)
		}
		controlListeners = append(controlListeners, ln)
	}
	if hasKCP {
		ln, err := listenKCP(controlAddr)
		if err != nil {
			// 关闭已创建的监听，避免泄露。
			for _, item := range controlListeners {
				_ = item.Close()
			}
			return fmt.Errorf("listen kcp %s failed: %w", controlAddr, err)
		}
		controlListeners = append(controlListeners, ln)
	}
	if len(controlListeners) == 0 {
		return fmt.Errorf("traversal protocol %q is not supported, only tcp, kcp, tcp+kcp are supported", s.cfg.Protocol)
	}
	s.controlListeners = controlListeners

	s.logger.Info().
		Str("control_addr", controlAddr).
		Str("protocol", s.cfg.Protocol).
		Msg("traversal server started")

	errCh := make(chan error, 1)
	for _, ln := range s.controlListeners {
		s.stopWg.Add(1)
		go s.serveControl(ln, errCh)
	}

	err = <-errCh
	if err == nil || errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

// Stop 停止内网穿透服务端。
func (s *Server) Stop() error {
	if !s.cfg.Enabled {
		return nil
	}

	var stopErr error
	s.stopMux.Do(func() {
		close(s.stopCh)
		for _, ln := range s.controlListeners {
			if err := ln.Close(); err != nil && !errors.Is(err, net.ErrClosed) && stopErr == nil {
				stopErr = err
			}
		}
		s.controlListeners = nil
		if err := s.closeProxyListeners(); err != nil && stopErr == nil {
			stopErr = err
		}
		s.clientMu.Lock()
		if s.client != nil {
			_ = s.client.conn.Close()
			s.client = nil
		}
		s.clientMu.Unlock()
		s.closeAllPending()
	})
	s.stopWg.Wait()
	return stopErr
}

func (s *Server) serveControl(ln net.Listener, errCh chan<- error) {
	defer s.stopWg.Done()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if s.isStopping() || errors.Is(err, net.ErrClosed) {
				errCh <- nil
				return
			}
			errCh <- err
			return
		}
		go s.handleControlConn(conn)
	}
}

func (s *Server) serveProxy(ln net.Listener) {
	defer s.stopWg.Done()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if s.isStopping() || errors.Is(err, net.ErrClosed) {
				return
			}
			s.logger.Error().Err(err).Msg("traversal proxy accept error")
			continue
		}
		go s.handlePublicConn(conn, ln)
	}
}

func (s *Server) handleControlConn(conn net.Conn) {
	reader := bufio.NewReader(conn)
	msg, err := ReadMessage(reader)
	if err != nil {
		_ = conn.Close()
		return
	}
	switch msg.Type {
	case MessageTypeRegister:
		s.handleRegister(conn, reader, msg)
	case MessageTypeJoin:
		s.handleJoin(conn, reader, msg)
	default:
		_ = WriteMessage(conn, Message{Type: MessageTypeError, Message: "unsupported message type"})
		_ = conn.Close()
	}
}

func (s *Server) handleRegister(conn net.Conn, reader *bufio.Reader, msg Message) {
	if msg.AuthKey != s.cfg.AuthKey {
		_ = WriteMessage(conn, Message{
			Type:    MessageTypeRegisterAck,
			OK:      false,
			Message: "auth failed",
		})
		_ = conn.Close()
		return
	}
	if len(msg.ProxyServers) == 0 {
		_ = WriteMessage(conn, Message{
			Type:    MessageTypeRegisterAck,
			OK:      false,
			Message: "proxy_server is required",
		})
		_ = conn.Close()
		return
	}

	listeners, proxyAddrDesc, err := s.tryCreateProxyListeners(msg.ProxyServers)
	if err != nil {
		_ = WriteMessage(conn, Message{
			Type:    MessageTypeRegisterAck,
			OK:      false,
			Message: err.Error(),
		})
		_ = conn.Close()
		return
	}

	client := &registeredClient{
		conn:             conn,
		writer:           bufio.NewWriter(conn),
		lastSeenUnixNano: time.Now().UnixNano(),
	}

	s.clientMu.Lock()
	if s.client != nil {
		_ = s.client.conn.Close()
	}
	s.client = client
	s.clientMu.Unlock()

	s.swapProxyListeners(listeners)

	_ = client.send(Message{
		Type:    MessageTypeRegisterAck,
		OK:      true,
		Message: "registered, proxy addresses: " + proxyAddrDesc,
	})
	s.logger.Info().
		Str("remote", conn.RemoteAddr().String()).
		Str("proxy_addr", proxyAddrDesc).
		Msg("traversal client registered")

	// KCP 控制通道心跳：服务端主动 ping，客户端需回 pong。
	// 仅对 KCP 控制连接启用（TCP 不需要）。
	if session, ok := conn.(*kcp.UDPSession); ok {
		session.SetStreamMode(true)
		interval := time.Duration(s.cfg.KCP.HeartbeatInterval) * time.Second
		timeout := time.Duration(s.cfg.KCP.HeartbeatTimeout) * time.Second
		if interval > 0 && timeout > 0 {
			done := make(chan struct{})
			defer close(done)
			go s.kcpHeartbeatLoop(client, interval, timeout, done)
		}
	}

	for {
		m, err := ReadMessage(reader)
		if err != nil {
			s.clientMu.Lock()
			if s.client != nil && s.client.conn == conn {
				s.client = nil
				_ = s.closeProxyListeners()
			}
			s.clientMu.Unlock()
			_ = conn.Close()
			s.logger.Info().Str("remote", conn.RemoteAddr().String()).Msg("traversal client disconnected")
			return
		}
		atomic.StoreInt64(&client.lastSeenUnixNano, time.Now().UnixNano())

		switch m.Type {
		case MessageTypePing:
			_ = client.send(Message{Type: MessageTypePong})
		case MessageTypePong:
		default:
		}
	}
}

func (s *Server) kcpHeartbeatLoop(client *registeredClient, interval, timeout time.Duration, done <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			last := time.Unix(0, atomic.LoadInt64(&client.lastSeenUnixNano))
			if time.Since(last) > timeout {
				_ = client.conn.Close()
				return
			}
			_ = client.send(Message{Type: MessageTypePing})
		case <-s.stopCh:
			return
		case <-done:
			return
		}
	}
}

func (s *Server) handleJoin(conn net.Conn, reader *bufio.Reader, msg Message) {
	if msg.AuthKey != s.cfg.AuthKey {
		_ = conn.Close()
		return
	}
	if strings.TrimSpace(msg.ConnID) == "" {
		_ = conn.Close()
		return
	}

	buffered := &bufferedConn{
		Conn:   conn,
		reader: reader,
	}

	s.pendingMu.Lock()
	item, ok := s.pending[msg.ConnID]
	s.pendingMu.Unlock()
	if !ok {
		_ = buffered.Close()
		return
	}

	select {
	case item.joinCh <- buffered:
	default:
		_ = buffered.Close()
	}
}

func (s *Server) handlePublicConn(publicConn net.Conn, ln net.Listener) {
	client, ok := s.getClient()
	if !ok {
		_ = publicConn.Close()
		return
	}

	mappingName, remotePort := s.resolveMappingInfo(ln)

	connID := s.nextConnID()
	item := &pendingTunnel{
		publicConn: publicConn,
		joinCh:     make(chan net.Conn, 1),
	}
	s.pendingMu.Lock()
	s.pending[connID] = item
	s.pendingMu.Unlock()

	if err := client.send(Message{
		Type:        MessageTypeOpen,
		ConnID:      connID,
		MappingName: mappingName,
		RemotePort:  remotePort,
	}); err != nil {
		s.removePending(connID)
		_ = publicConn.Close()
		return
	}

	var dataConn net.Conn
	select {
	case dataConn = <-item.joinCh:
	case <-time.After(tunnelJoinTimeout):
		s.removePending(connID)
		_ = publicConn.Close()
		return
	case <-s.stopCh:
		s.removePending(connID)
		_ = publicConn.Close()
		return
	}
	s.removePending(connID)

	s.bridge(publicConn, dataConn)
}

func (s *Server) bridge(a net.Conn, b net.Conn) {
	defer a.Close()
	defer b.Close()

	errCh := make(chan error, 2)
	go func() {
		_ = a.SetReadDeadline(time.Now().Add(relayIdleTimeout))
		_ = b.SetWriteDeadline(time.Now().Add(relayIdleTimeout))
		_, err := copyWithPooledBuffer(b, a)
		closeWrite(b)
		errCh <- err
	}()
	go func() {
		_ = b.SetReadDeadline(time.Now().Add(relayIdleTimeout))
		_ = a.SetWriteDeadline(time.Now().Add(relayIdleTimeout))
		_, err := copyWithPooledBuffer(a, b)
		closeWrite(a)
		errCh <- err
	}()

	for i := 0; i < 2; i++ {
		err := <-errCh
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
			return
		}
	}
}

func copyWithPooledBuffer(dst io.Writer, src io.Reader) (int64, error) {
	bufPtr := relayBufferPool.Get().(*[]byte)
	defer relayBufferPool.Put(bufPtr)
	return io.CopyBuffer(dst, src, *bufPtr)
}

func closeWrite(conn net.Conn) {
	type closeWriter interface {
		CloseWrite() error
	}
	if cw, ok := conn.(closeWriter); ok {
		_ = cw.CloseWrite()
	}
}

// parseProtocols 解析并校验控制协议配置。
// 支持的取值：
//   - "tcp"      : 仅启用 TCP 控制监听
//   - "kcp"      : 仅启用 KCP 控制监听
//   - "tcp+kcp"  : 同时启用 TCP 和 KCP 控制监听
func (s *Server) parseProtocols() (hasTCP, hasKCP bool, err error) {
	raw := strings.TrimSpace(s.cfg.Protocol)
	if raw == "" || raw == "tcp" {
		return true, false, nil
	}
	parts := strings.Split(raw, "+")
	for _, part := range parts {
		p := strings.TrimSpace(part)
		switch p {
		case "tcp":
			hasTCP = true
		case "kcp":
			hasKCP = true
		case "":
		default:
			return false, false, fmt.Errorf("traversal protocol %q is not supported, only tcp, kcp, tcp+kcp are supported", s.cfg.Protocol)
		}
	}
	if !hasTCP && !hasKCP {
		return false, false, fmt.Errorf("traversal protocol %q is not supported, only tcp, kcp, tcp+kcp are supported", s.cfg.Protocol)
	}
	return hasTCP, hasKCP, nil
}

func (s *Server) tryCreateProxyListeners(bindings []ProxyServerBinding) ([]*proxyListener, string, error) {
	nameSet := make(map[string]struct{}, len(bindings))
	portSet := make(map[int]struct{}, len(bindings))
	listeners := make([]*proxyListener, 0, len(bindings))
	descs := make([]string, 0, len(bindings))

	closeCreated := func() {
		for _, item := range listeners {
			_ = item.listener.Close()
		}
	}

	for _, binding := range bindings {
		name := strings.TrimSpace(binding.Name)
		if name == "" {
			closeCreated()
			return nil, "", errors.New("proxy_server.name cannot be empty")
		}
		if binding.RemotePort <= 0 {
			closeCreated()
			return nil, "", fmt.Errorf("proxy_server.remote_port must be greater than 0: %s", name)
		}
		if _, ok := nameSet[name]; ok {
			closeCreated()
			return nil, "", fmt.Errorf("proxy_server.name duplicated: %s", name)
		}
		nameSet[name] = struct{}{}
		if _, ok := portSet[binding.RemotePort]; ok {
			closeCreated()
			return nil, "", fmt.Errorf("proxy_server.remote_port duplicated: %d", binding.RemotePort)
		}
		portSet[binding.RemotePort] = struct{}{}

		addr := net.JoinHostPort(s.cfg.ListenHost, strconv.Itoa(binding.RemotePort))
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			closeCreated()
			return nil, "", fmt.Errorf("[%s] remote_port %d bind failed: %w", name, binding.RemotePort, err)
		}
		listeners = append(listeners, &proxyListener{
			name:       name,
			remotePort: binding.RemotePort,
			listener:   ln,
		})
		descs = append(descs, fmt.Sprintf("[%s]%s", name, addr))
	}

	return listeners, strings.Join(descs, ", "), nil
}

func (s *Server) swapProxyListeners(newListeners []*proxyListener) {
	s.proxyMu.Lock()
	old := s.proxyListeners
	next := make(map[int]*proxyListener, len(newListeners))
	for _, item := range newListeners {
		next[item.remotePort] = item
	}
	s.proxyListeners = next
	s.proxyMu.Unlock()

	for _, item := range old {
		_ = item.listener.Close()
	}

	for _, item := range newListeners {
		s.stopWg.Add(1)
		go s.serveProxy(item.listener)
	}
}

func (s *Server) closeProxyListeners() error {
	s.proxyMu.Lock()
	listeners := s.proxyListeners
	s.proxyListeners = make(map[int]*proxyListener)
	s.proxyMu.Unlock()

	var firstErr error
	for _, item := range listeners {
		err := item.listener.Close()
		if err != nil && !errors.Is(err, net.ErrClosed) && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *Server) resolveMappingInfo(ln net.Listener) (string, int) {
	s.proxyMu.Lock()
	defer s.proxyMu.Unlock()
	for _, item := range s.proxyListeners {
		if item.listener == ln {
			return item.name, item.remotePort
		}
	}
	return "", 0
}

func (s *Server) getClient() (*registeredClient, bool) {
	s.clientMu.RLock()
	defer s.clientMu.RUnlock()
	if s.client == nil {
		return nil, false
	}
	return s.client, true
}

func (s *Server) nextConnID() string {
	next := atomic.AddUint64(&s.idSeed, 1)
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), next)
}

func (s *Server) removePending(connID string) {
	s.pendingMu.Lock()
	delete(s.pending, connID)
	s.pendingMu.Unlock()
}

func (s *Server) closeAllPending() {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	for key, item := range s.pending {
		_ = item.publicConn.Close()
		close(item.joinCh)
		delete(s.pending, key)
	}
}

func (s *Server) isStopping() bool {
	select {
	case <-s.stopCh:
		return true
	default:
		return false
	}
}

func (c *registeredClient) send(msg Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := WriteMessage(c.writer, msg); err != nil {
		return err
	}
	return c.writer.Flush()
}

type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}
