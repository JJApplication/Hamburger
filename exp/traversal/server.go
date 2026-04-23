package traversal

import (
	"Hamburger/internal/config"
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
)

const (
	defaultTraversalHost = "0.0.0.0"
	defaultTraversalPort = 19090
	tunnelJoinTimeout    = 10 * time.Second
	relayIdleTimeout     = 90 * time.Second
)

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
}

// Server 内网穿透服务端。
type Server struct {
	cfg            config.TraversalServerConfig
	logger         *zerolog.Logger
	listener       net.Listener
	proxyListeners map[int]*proxyListener
	proxyMu        sync.Mutex

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
	return &Server{
		cfg:            serverCfg,
		logger:         logger,
		pending:        make(map[string]*pendingTunnel),
		proxyListeners: make(map[int]*proxyListener),
		stopCh:         make(chan struct{}),
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
	if s.cfg.Protocol != "tcp" {
		return fmt.Errorf("traversal protocol %q is not supported, only tcp is supported", s.cfg.Protocol)
	}

	controlAddr := net.JoinHostPort(s.cfg.ListenHost, strconv.Itoa(s.cfg.ListenPort))
	controlLn, err := net.Listen("tcp", controlAddr)
	if err != nil {
		return err
	}
	s.listener = controlLn

	s.logger.Info().
		Str("control_addr", controlAddr).
		Str("protocol", s.cfg.Protocol).
		Msg("traversal server started")

	errCh := make(chan error, 1)
	s.stopWg.Add(1)

	go s.serveControl(errCh)

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
		if s.listener != nil {
			if err := s.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) && stopErr == nil {
				stopErr = err
			}
		}
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

func (s *Server) serveControl(errCh chan<- error) {
	defer s.stopWg.Done()
	for {
		conn, err := s.listener.Accept()
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
		conn:   conn,
		writer: bufio.NewWriter(conn),
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

	for {
		if _, err := ReadMessage(reader); err != nil {
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
		_, err := io.Copy(b, a)
		closeWrite(b)
		errCh <- err
	}()
	go func() {
		_ = b.SetReadDeadline(time.Now().Add(relayIdleTimeout))
		_ = a.SetWriteDeadline(time.Now().Add(relayIdleTimeout))
		_, err := io.Copy(a, b)
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

func closeWrite(conn net.Conn) {
	type closeWriter interface {
		CloseWrite() error
	}
	if cw, ok := conn.(closeWriter); ok {
		_ = cw.CloseWrite()
	}
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
