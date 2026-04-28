package server

import (
	"Hamburger/gateway/tls"
	"Hamburger/internal/config/core_config"
	"context"
	"fmt"
	"github.com/lesismal/nbio/nbhttp"
	"github.com/rs/zerolog"
	"net"
	"net/http"
	"sync"
	"time"
)

// 使用NBIO作为高性能服务器

type nbioServerRunner struct {
	server   *nbhttp.Server
	stopCh   chan struct{}
	stopOnce sync.Once
}

func newNbioServerRunner(server *nbhttp.Server) *nbioServerRunner {
	return &nbioServerRunner{
		server: server,
		stopCh: make(chan struct{}),
	}
}

func (r *nbioServerRunner) Serve(_ net.Listener) error {
	if err := r.server.Start(); err != nil {
		return err
	}
	<-r.stopCh
	return http.ErrServerClosed
}

func (r *nbioServerRunner) Shutdown(ctx context.Context) error {
	err := r.server.Shutdown(ctx)
	r.stopOnce.Do(func() {
		close(r.stopCh)
	})
	return err
}

func (r *nbioServerRunner) Close() error {
	r.server.Stop()
	r.stopOnce.Do(func() {
		close(r.stopCh)
	})
	return nil
}

func CommonNbioServer(serverConfig core_config.ServerConfig, logger *zerolog.Logger, h http.Handler, tlsManager *tls.TLSManager) (*ServerInstance, error) {
	instance := &ServerInstance{
		Name:   serverConfig.Name,
		Config: serverConfig,
		TLS:    serverConfig.Protocol == "https" || serverConfig.Protocol == "http3",
		Logger: logger,
		lock:   sync.RWMutex{},
	}

	addr := fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port)

	var originHandler http.Handler
	if instance.TLS {
		originHandler = wrapHandlerWithTag(h, "https443")
	} else {
		originHandler = wrapHandlerWithTag(h, "http80")
	}

	if serverConfig.MaxRequestBody > 0 {
		originHandler = wrapHandlerWithMaxBody(originHandler, logger, serverConfig)
	}
	if serverConfig.Protocol == "http" {
		originHandler = wrapHandlerWithAutoHttpsRedirect(originHandler, logger, serverConfig)
	}

	conf := nbhttp.Config{
		Name:    serverConfig.Name,
		Network: "tcp",
		Addrs:   []string{addr},
		Handler: originHandler,
	}
	if serverConfig.MaxRequestBody > 0 {
		conf.MaxHTTPBodySize = int(serverConfig.MaxRequestBody)
	}
	if serverConfig.WriteTimeout > 0 {
		conf.WriteTimeout = time.Second * time.Duration(serverConfig.WriteTimeout)
	}
	if serverConfig.IdleTimeout > 0 {
		conf.KeepaliveTime = time.Second * time.Duration(serverConfig.IdleTimeout)
	}

	var nbioSrv *nbhttp.Server
	if instance.TLS {
		if tlsManager == nil {
			return nil, fmt.Errorf("tls manager not initialized")
		}
		tlsCfg, err := tlsManager.GetNbioTLSConfig(instance.Config.TLS)
		if err != nil {
			return nil, err
		}
		nbioSrv = nbhttp.NewServerTLS(conf, originHandler, nil, tlsCfg)
	} else {
		nbioSrv = nbhttp.NewServer(conf, originHandler)
	}

	instance.Server = newNbioServerRunner(nbioSrv)
	logger.Info().Msgf("server %s starting listener on %s (protocol: %s)",
		serverConfig.Name, addr, serverConfig.Protocol)

	return instance, nil
}
