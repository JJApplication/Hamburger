package any_tls

import (
	"Hamburger/internal/config"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/rs/zerolog"
	"io"
	"net"
	"sync"
	"time"

	anytls "github.com/anytls/sing-anytls"
	"github.com/sagernet/sing/common/logger"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

type AnyTLSServer struct {
	mu       sync.Mutex
	cfg      *config.Config
	logger   *zerolog.Logger
	enabled  bool
	service  *anytls.Service
	listener net.Listener
}

type tcpRelayHandler struct {
	dialTimeout time.Duration
}

func (h *tcpRelayHandler) NewConnectionEx(ctx context.Context, conn net.Conn, _ M.Socksaddr, destination M.Socksaddr, onClose N.CloseHandlerFunc) {
	dialer := &net.Dialer{Timeout: h.dialTimeout}
	upstream, err := dialer.DialContext(ctx, "tcp", destination.String())
	if err != nil {
		_ = conn.Close()
		if onClose != nil {
			onClose(err)
		}
		return
	}

	errCh := make(chan error, 2)

	go func() {
		_, copyErr := io.Copy(upstream, conn)
		if tcpConn, ok := upstream.(*net.TCPConn); ok {
			_ = tcpConn.CloseWrite()
		}
		errCh <- copyErr
	}()

	go func() {
		_, copyErr := io.Copy(conn, upstream)
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			_ = tcpConn.CloseWrite()
		}
		errCh <- copyErr
	}()

	var relayErr error
	for i := 0; i < 2; i++ {
		copyErr := <-errCh
		if copyErr != nil && !errors.Is(copyErr, io.EOF) && !errors.Is(copyErr, net.ErrClosed) {
			relayErr = copyErr
			break
		}
	}

	_ = conn.Close()
	_ = upstream.Close()
	if onClose != nil {
		onClose(relayErr)
	}
}

func NewAnyTLSServer(cfg *config.Config, logger *zerolog.Logger) *AnyTLSServer {
	return &AnyTLSServer{
		cfg:     cfg,
		logger:  logger,
		enabled: cfg.ExpConfig.AnyTLSServer.Enabled,
	}
}

func (as *AnyTLSServer) Start() error {
	listener, service, err := as.open()
	if err != nil || listener == nil {
		return err
	}
	return as.serve(listener, service)
}

// StartAsync binds the configured listener before returning and serves it in
// the background. Management operations use this form so success means the
// port was actually opened rather than merely scheduling a goroutine.
func (as *AnyTLSServer) StartAsync() error {
	listener, service, err := as.open()
	if err != nil || listener == nil {
		return err
	}
	go func() {
		if serveErr := as.serve(listener, service); serveErr != nil && as.logger != nil {
			as.logger.Error().Err(serveErr).Msg("anytls error serving listener")
		}
	}()
	return nil
}

func (as *AnyTLSServer) open() (net.Listener, *anytls.Service, error) {
	if as == nil || as.cfg == nil {
		return nil, nil, errors.New("anytls configuration unavailable")
	}
	asCfg := as.cfg.ExpConfig.AnyTLSServer
	as.mu.Lock()
	defer as.mu.Unlock()
	as.enabled = asCfg.Enabled
	if !as.enabled {
		return nil, nil, nil
	}
	if as.listener != nil {
		return nil, nil, errors.New("anytls server already running")
	}
	certificate, err := tls.LoadX509KeyPair(asCfg.CertFile, asCfg.KeyFile)
	if err != nil {
		if as.logger != nil {
			as.logger.Error().Err(err).Msg("anytls error loading certificate")
		}
		return nil, nil, err
	}
	service, err := anytls.NewService(anytls.ServiceConfig{
		Users: []anytls.User{
			{
				Name:     asCfg.User,
				Password: asCfg.Password,
			},
		},
		Handler: &tcpRelayHandler{
			dialTimeout: time.Duration(asCfg.DialTimeout) * time.Second,
		},
		Logger: logger.NOP(),
	})
	if err != nil {
		if as.logger != nil {
			as.logger.Error().Err(err).Msg("anytls error creating service")
		}
		return nil, nil, err
	}
	listener, err := tls.Listen("tcp", fmt.Sprintf("%s:%d", asCfg.Host, asCfg.Port), &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{certificate},
	})
	if err != nil {
		if as.logger != nil {
			as.logger.Error().Err(err).Msg("anytls error creating listener")
		}
		return nil, nil, err
	}
	as.service = service
	as.listener = listener
	return listener, service, nil
}

func (as *AnyTLSServer) serve(listener net.Listener, service *anytls.Service) error {
	for {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			if errors.Is(acceptErr, net.ErrClosed) {
				if as.logger != nil {
					as.logger.Info().Msg("anytls listener closed")
				}
				return nil
			}
			if as.logger != nil {
				as.logger.Error().Err(acceptErr).Msg("anytls error accepting connection")
			}
			continue
		}

		go func(c net.Conn) {
			source := M.SocksaddrFromNet(c.RemoteAddr())
			serveErr := service.NewConnection(context.Background(), c, source, func(_ error) {})
			if serveErr != nil && !errors.Is(serveErr, net.ErrClosed) {
				if as.logger != nil {
					as.logger.Error().Err(serveErr).Msg("anytls error serving connection")
				}
			}
			_ = c.Close()
		}(conn)
	}
}

func (as *AnyTLSServer) Stop() error {
	if as == nil {
		return nil
	}
	as.mu.Lock()
	listener := as.listener
	as.listener = nil
	as.service = nil
	as.mu.Unlock()
	if listener == nil {
		return nil
	}
	return listener.Close()
}
