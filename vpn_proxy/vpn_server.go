package vpn_proxy

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"Hamburger/internal/config"

	"github.com/rs/zerolog"
)

type VpnServer struct {
	cfg           *config.Config
	logger        *zerolog.Logger
	httpServer    *http.Server
	httpListener  net.Listener
	socksListener net.Listener
	stopCh        chan struct{}
	stopOnce      sync.Once
	started       bool
	transport     *http.Transport
}

func NewVpnServer(cfg *config.Config, logger *zerolog.Logger) *VpnServer {
	s := &VpnServer{
		cfg:    cfg,
		logger: logger,
		stopCh: make(chan struct{}),
		transport: &http.Transport{
			Proxy:               nil,
			MaxIdleConns:        256,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}
	return s
}

func (s *VpnServer) Start() error {
	if s.cfg == nil || !s.cfg.VpnServer.Enabled {
		return nil
	}
	if s.started {
		return nil
	}
	cfg := s.cfg.VpnServer
	if cfg.HttpPort > 0 {
		addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.HttpPort)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		if cfg.TLS.Enabled && cfg.TLS.EnableHTTP {
			tlsConfig, err := s.loadTLSConfig()
			if err != nil {
				ln.Close()
				return err
			}
			ln = tls.NewListener(ln, tlsConfig)
		}
		s.httpListener = ln
		s.httpServer = &http.Server{
			Handler: http.HandlerFunc(s.handleHTTP),
		}
		go func() {
			err := s.httpServer.Serve(ln)
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				s.logger.Error().Err(err).Msg("vpn http proxy listen error")
			}
		}()
		s.logger.Info().Str("address", addr).Msg("vpn http proxy started")
	}
	if cfg.SocksPort > 0 {
		addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.SocksPort)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		if cfg.TLS.Enabled && cfg.TLS.EnableSocks {
			tlsConfig, err := s.loadTLSConfig()
			if err != nil {
				ln.Close()
				return err
			}
			ln = tls.NewListener(ln, tlsConfig)
		}
		s.socksListener = ln
		go s.acceptSocksLoop(ln)
		s.logger.Info().Str("address", addr).Msg("vpn socks5 proxy started")
	}
	s.started = true
	return nil
}

func (s *VpnServer) Stop() error {
	if !s.started {
		return nil
	}
	s.started = false
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	if s.httpServer != nil {
		_ = s.httpServer.Close()
	}
	if s.httpListener != nil {
		_ = s.httpListener.Close()
	}
	if s.socksListener != nil {
		_ = s.socksListener.Close()
	}
	return nil
}

func (s *VpnServer) loadTLSConfig() (*tls.Config, error) {
	cfg := s.cfg.VpnServer.TLS
	if cfg.CertFile == "" || cfg.KeyFile == "" {
		return nil, fmt.Errorf("vpn tls cert or key not configured")
	}
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

func (s *VpnServer) handleHTTP(w http.ResponseWriter, r *http.Request) {
	if !s.checkHTTPAuth(w, r) {
		return
	}
	if r.Method == http.MethodConnect {
		s.handleHTTPConnect(w, r)
		return
	}
	if r.URL == nil || !r.URL.IsAbs() {
		s.writeCamouflage(w)
		return
	}
	if r.URL.Scheme == "" {
		r.URL.Scheme = "http"
	}
	outReq := r.Clone(r.Context())
	outReq.RequestURI = ""
	outReq.Header = s.cloneHeader(outReq.Header)
	s.removeHopHeaders(outReq.Header)
	outReq.Header.Del("Proxy-Authorization")
	outReq.Header.Del("Proxy-Connection")

	resp, err := s.transport.RoundTrip(outReq)
	if err != nil {
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (s *VpnServer) handleHTTPConnect(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if host == "" && r.URL != nil {
		host = r.URL.Host
	}
	if host == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !strings.Contains(host, ":") {
		host = net.JoinHostPort(host, "443")
	}
	dialer := net.Dialer{
		Timeout: s.getTimeout(),
	}
	serverConn, err := dialer.Dial("tcp", host)
	if err != nil {
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		serverConn.Close()
		http.Error(w, "hijack not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		serverConn.Close()
		return
	}
	_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	s.pipeConn(clientConn, serverConn)
}

func (s *VpnServer) checkHTTPAuth(w http.ResponseWriter, r *http.Request) bool {
	auth := s.cfg.VpnServer.Auth
	if !auth.Enabled {
		return true
	}
	header := r.Header.Get("Proxy-Authorization")
	if header == "" {
		header = r.Header.Get("Authorization")
	}
	const prefix = "basic "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		s.writeProxyAuthRequired(w)
		return false
	}
	raw := strings.TrimSpace(header[len(prefix):])
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		s.writeProxyAuthRequired(w)
		return false
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 || parts[0] != auth.Username || parts[1] != auth.Password {
		s.writeProxyAuthRequired(w)
		return false
	}
	return true
}

func (s *VpnServer) writeProxyAuthRequired(w http.ResponseWriter) {
	w.Header().Set("Proxy-Authenticate", "Basic realm=\"vpn\"")
	w.WriteHeader(http.StatusProxyAuthRequired)
}

func (s *VpnServer) writeCamouflage(w http.ResponseWriter) {
	obfs := s.cfg.VpnServer.Obfs
	if !obfs.CamouflageEnabled {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	status := obfs.CamouflageStatus
	if status == 0 {
		status = http.StatusOK
	}
	body := obfs.CamouflageBody
	if body == "" {
		body = "<html><head><title>OK</title></head><body>OK</body></html>"
	}
	mime := obfs.CamouflageMimeType
	if mime == "" {
		mime = "text/html; charset=utf-8"
	}
	w.Header().Set("Content-Type", mime)
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func (s *VpnServer) acceptSocksLoop(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-s.stopCh:
				return
			default:
				s.logger.Error().Err(err).Msg("vpn socks accept error")
				continue
			}
		}
		go s.handleSocksConn(conn)
	}
}

func (s *VpnServer) handleSocksConn(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(s.getTimeout()))
	br := bufio.NewReader(conn)
	ver, err := br.ReadByte()
	if err != nil || ver != 0x05 {
		return
	}
	nMethods, err := br.ReadByte()
	if err != nil {
		return
	}
	methods := make([]byte, int(nMethods))
	if _, err := io.ReadFull(br, methods); err != nil {
		return
	}
	method := byte(0x00)
	authEnabled := s.cfg.VpnServer.Auth.Enabled
	if authEnabled {
		method = 0xFF
		for _, m := range methods {
			if m == 0x02 {
				method = 0x02
				break
			}
		}
	} else {
		method = 0xFF
		for _, m := range methods {
			if m == 0x00 {
				method = 0x00
				break
			}
		}
		if method == 0xFF {
			for _, m := range methods {
				if m == 0x02 {
					method = 0x02
					break
				}
			}
		}
	}
	if _, err := conn.Write([]byte{0x05, method}); err != nil {
		return
	}
	if method == 0xFF {
		return
	}
	if method == 0x02 {
		if !s.socksAuth(conn, br) {
			return
		}
	}
	req, err := s.readSocksRequest(br)
	if err != nil {
		return
	}
	if req.cmd != 0x01 {
		s.writeSocksReply(conn, 0x07, nil)
		return
	}
	dialer := net.Dialer{
		Timeout: s.getTimeout(),
	}
	targetConn, err := dialer.Dial("tcp", req.addr)
	if err != nil {
		s.writeSocksReply(conn, 0x05, nil)
		return
	}
	defer targetConn.Close()
	s.writeSocksReply(conn, 0x00, targetConn.LocalAddr())
	s.pipeConn(conn, targetConn)
}

func (s *VpnServer) socksAuth(conn net.Conn, br *bufio.Reader) bool {
	ver, err := br.ReadByte()
	if err != nil || ver != 0x01 {
		return false
	}
	ulen, err := br.ReadByte()
	if err != nil {
		return false
	}
	ubuf := make([]byte, int(ulen))
	if _, err := io.ReadFull(br, ubuf); err != nil {
		return false
	}
	plen, err := br.ReadByte()
	if err != nil {
		return false
	}
	pbuf := make([]byte, int(plen))
	if _, err := io.ReadFull(br, pbuf); err != nil {
		return false
	}
	auth := s.cfg.VpnServer.Auth
	if string(ubuf) != auth.Username || string(pbuf) != auth.Password {
		_, _ = conn.Write([]byte{0x01, 0x01})
		return false
	}
	_, _ = conn.Write([]byte{0x01, 0x00})
	return true
}

type socksRequest struct {
	cmd  byte
	addr string
}

func (s *VpnServer) readSocksRequest(br *bufio.Reader) (*socksRequest, error) {
	ver, err := br.ReadByte()
	if err != nil || ver != 0x05 {
		return nil, fmt.Errorf("invalid socks version")
	}
	cmd, err := br.ReadByte()
	if err != nil {
		return nil, err
	}
	_, err = br.ReadByte()
	if err != nil {
		return nil, err
	}
	atyp, err := br.ReadByte()
	if err != nil {
		return nil, err
	}
	var host string
	switch atyp {
	case 0x01:
		buf := make([]byte, 4)
		if _, err := io.ReadFull(br, buf); err != nil {
			return nil, err
		}
		host = net.IP(buf).String()
	case 0x03:
		l, err := br.ReadByte()
		if err != nil {
			return nil, err
		}
		buf := make([]byte, int(l))
		if _, err := io.ReadFull(br, buf); err != nil {
			return nil, err
		}
		host = string(buf)
	case 0x04:
		buf := make([]byte, 16)
		if _, err := io.ReadFull(br, buf); err != nil {
			return nil, err
		}
		host = net.IP(buf).String()
	default:
		return nil, fmt.Errorf("invalid address type")
	}
	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(br, portBuf); err != nil {
		return nil, err
	}
	port := int(portBuf[0])<<8 | int(portBuf[1])
	return &socksRequest{
		cmd:  cmd,
		addr: net.JoinHostPort(host, strconv.Itoa(port)),
	}, nil
}

func (s *VpnServer) writeSocksReply(conn net.Conn, rep byte, addr net.Addr) {
	bindAddr := "0.0.0.0"
	bindPort := 0
	if addr != nil {
		if tcpAddr, ok := addr.(*net.TCPAddr); ok {
			bindAddr = tcpAddr.IP.String()
			bindPort = tcpAddr.Port
		}
	}
	ip := net.ParseIP(bindAddr).To4()
	if ip == nil {
		ip = net.IPv4zero
	}
	reply := []byte{
		0x05, rep, 0x00, 0x01,
		ip[0], ip[1], ip[2], ip[3],
		byte(bindPort >> 8), byte(bindPort),
	}
	_, _ = conn.Write(reply)
}

func (s *VpnServer) pipeConn(client net.Conn, server net.Conn) {
	timeout := s.getTimeout()
	if timeout > 0 {
		_ = client.SetDeadline(time.Now().Add(timeout))
		_ = server.SetDeadline(time.Now().Add(timeout))
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = s.copyWithObfs(server, client)
	}()
	go func() {
		defer wg.Done()
		_, _ = s.copyWithObfs(client, server)
	}()
	wg.Wait()
	_ = client.Close()
	_ = server.Close()
}

func (s *VpnServer) copyWithObfs(dst io.Writer, src io.Reader) (int64, error) {
	obfs := s.cfg.VpnServer.Obfs
	if !obfs.Enabled {
		return io.Copy(dst, src)
	}
	minChunk := obfs.MinChunkSize
	maxChunk := obfs.MaxChunkSize
	if minChunk <= 0 {
		minChunk = 1024
	}
	if maxChunk < minChunk {
		maxChunk = minChunk
	}
	minDelay := obfs.MinDelayMs
	maxDelay := obfs.MaxDelayMs
	if maxDelay < minDelay {
		maxDelay = minDelay
	}
	buf := make([]byte, maxChunk)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var total int64
	for {
		n, err := src.Read(buf)
		if n > 0 {
			offset := 0
			for offset < n {
				chunk := minChunk
				if maxChunk > minChunk {
					chunk = rng.Intn(maxChunk-minChunk+1) + minChunk
				}
				if chunk > n-offset {
					chunk = n - offset
				}
				w, werr := dst.Write(buf[offset : offset+chunk])
				total += int64(w)
				if w <= 0 && werr == nil {
					return total, io.ErrUnexpectedEOF
				}
				offset += w
				if werr != nil {
					return total, werr
				}
				if maxDelay > 0 {
					delay := minDelay
					if maxDelay > minDelay {
						delay = rng.Intn(maxDelay-minDelay+1) + minDelay
					}
					if delay > 0 {
						time.Sleep(time.Duration(delay) * time.Millisecond)
					}
				}
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return total, nil
			}
			return total, err
		}
	}
}

func (s *VpnServer) removeHopHeaders(h http.Header) {
	connection := h.Get("Connection")
	for _, k := range []string{
		"Connection",
		"Proxy-Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	} {
		h.Del(k)
	}
	if connection != "" {
		for _, f := range strings.Split(connection, ",") {
			h.Del(strings.TrimSpace(f))
		}
	}
}

func (s *VpnServer) cloneHeader(h http.Header) http.Header {
	out := make(http.Header, len(h))
	for k, vv := range h {
		v2 := make([]string, len(vv))
		copy(v2, vv)
		out[k] = v2
	}
	return out
}

func (s *VpnServer) getTimeout() time.Duration {
	timeout := s.cfg.VpnServer.Timeout
	if timeout <= 0 {
		return 30 * time.Second
	}
	return time.Duration(timeout) * time.Second
}
