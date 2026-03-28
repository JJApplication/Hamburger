package dns

import (
	"Hamburger/internal/config"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"
)

type DNSServer struct {
	cfg      *config.Config
	logger   *zerolog.Logger
	enabled  bool
	dohPath  string
	upstream string
	timeout  time.Duration

	dnsUDPServer *dns.Server
	dnsTCPServer *dns.Server
	dohServer    *http.Server

	mu      sync.Mutex
	started bool
}

func NewDNSServer(cfg *config.Config, logger *zerolog.Logger) *DNSServer {
	conf := cfg.ExpConfig.DNSServer
	host := conf.Host
	if host == "" {
		host = "0.0.0.0"
	}
	port := conf.Port
	if port <= 0 {
		port = 53
	}
	upstream := conf.Upstream
	if upstream == "" {
		upstream = "8.8.8.8:53"
	}
	timeout := time.Duration(conf.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	dohPath := conf.DOH.Path
	if dohPath == "" {
		dohPath = "/dns-query"
	}
	if !strings.HasPrefix(dohPath, "/") {
		dohPath = "/" + dohPath
	}

	dnsAddr := net.JoinHostPort(host, strconv.Itoa(port))
	return &DNSServer{
		cfg:          cfg,
		logger:       logger,
		enabled:      conf.Enabled,
		dohPath:      dohPath,
		upstream:     upstream,
		timeout:      timeout,
		dnsUDPServer: &dns.Server{Addr: dnsAddr, Net: "udp"},
		dnsTCPServer: &dns.Server{Addr: dnsAddr, Net: "tcp"},
	}
}

func (s *DNSServer) Start() error {
	if !s.enabled {
		return nil
	}

	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return nil
	}
	s.started = true
	s.mu.Unlock()

	handler := dns.HandlerFunc(s.handleDNSQuery)
	s.dnsUDPServer.Handler = handler
	s.dnsTCPServer.Handler = handler

	errCh := make(chan error, 3)
	go s.serveDNS(s.dnsUDPServer, "udp", errCh)
	go s.serveDNS(s.dnsTCPServer, "tcp", errCh)

	if s.cfg.ExpConfig.DNSServer.DOH.Enabled {
		dohServer, err := s.newDOHServer()
		if err != nil {
			return err
		}
		s.dohServer = dohServer
		go s.serveDOH(errCh)
	}

	err := <-errCh
	if err == nil || errors.Is(err, http.ErrServerClosed) || errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func (s *DNSServer) Stop() error {
	if !s.enabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		return nil
	}
	s.started = false

	var firstErr error
	if s.dnsUDPServer != nil {
		if err := s.dnsUDPServer.Shutdown(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if s.dnsTCPServer != nil {
		if err := s.dnsTCPServer.Shutdown(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if s.dohServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
		defer cancel()
		if err := s.dohServer.Shutdown(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *DNSServer) serveDNS(server *dns.Server, netType string, errCh chan<- error) {
	s.logger.Info().Str("network", netType).Str("address", server.Addr).Msg("dns server started")
	if err := server.ListenAndServe(); err != nil {
		errCh <- err
		return
	}
	errCh <- nil
}

func (s *DNSServer) newDOHServer() (*http.Server, error) {
	conf := s.cfg.ExpConfig.DNSServer.DOH
	host := conf.Host
	if host == "" {
		host = "0.0.0.0"
	}
	port := conf.Port
	if port <= 0 {
		port = 8443
	}
	if conf.CertFile == "" || conf.KeyFile == "" {
		return nil, errors.New("doh cert_file and key_file are required")
	}

	cert, err := tls.LoadX509KeyPair(conf.CertFile, conf.KeyFile)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc(s.dohPath, s.handleDOHQuery)

	return &http.Server{
		Addr:         net.JoinHostPort(host, strconv.Itoa(port)),
		Handler:      mux,
		ReadTimeout:  s.timeout,
		WriteTimeout: s.timeout,
		TLSConfig: &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
		},
	}, nil
}

func (s *DNSServer) serveDOH(errCh chan<- error) {
	s.logger.Info().Str("path", s.dohPath).Str("address", s.dohServer.Addr).Msg("doh server started")
	if err := s.dohServer.ListenAndServeTLS("", ""); err != nil {
		errCh <- err
		return
	}
	errCh <- nil
}

func (s *DNSServer) handleDNSQuery(w dns.ResponseWriter, r *dns.Msg) {
	reply, err := s.exchange(r)
	if err != nil {
		fallback := new(dns.Msg)
		fallback.SetRcode(r, dns.RcodeServerFailure)
		_ = w.WriteMsg(fallback)
		return
	}
	_ = w.WriteMsg(reply)
}

func (s *DNSServer) handleDOHQuery(w http.ResponseWriter, r *http.Request) {
	msg, err := s.unpackDOHRequest(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	reply, err := s.exchange(msg)
	if err != nil {
		http.Error(w, "upstream failed", http.StatusBadGateway)
		return
	}
	raw, err := reply.Pack()
	if err != nil {
		http.Error(w, "encode failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/dns-message")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func (s *DNSServer) unpackDOHRequest(r *http.Request) (*dns.Msg, error) {
	switch r.Method {
	case http.MethodGet:
		encoded := r.URL.Query().Get("dns")
		if encoded == "" {
			return nil, errors.New("missing dns query")
		}
		raw, err := base64.RawURLEncoding.DecodeString(encoded)
		if err != nil {
			return nil, err
		}
		msg := new(dns.Msg)
		if err := msg.Unpack(raw); err != nil {
			return nil, err
		}
		return msg, nil
	case http.MethodPost:
		if ct := r.Header.Get("Content-Type"); !strings.Contains(strings.ToLower(ct), "application/dns-message") {
			return nil, errors.New("invalid content type")
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 4096))
		if err != nil {
			return nil, err
		}
		msg := new(dns.Msg)
		if err := msg.Unpack(raw); err != nil {
			return nil, err
		}
		return msg, nil
	default:
		return nil, errors.New("method not allowed")
	}
}

func (s *DNSServer) exchange(req *dns.Msg) (*dns.Msg, error) {
	network := "udp"
	if req.Len() > 1232 {
		network = "tcp"
	}
	client := &dns.Client{
		Net:     network,
		Timeout: s.timeout,
	}
	reply, _, err := client.Exchange(req, s.upstream)
	if err != nil {
		return nil, err
	}
	if reply == nil {
		return nil, errors.New("empty dns response")
	}
	return reply, nil
}
