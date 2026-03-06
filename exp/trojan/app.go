package trojan

import (
	"context"

	"Hamburger/exp/trojan/core/config"
	"Hamburger/exp/trojan/core/statistic/memory"
	"Hamburger/exp/trojan/core/tunnel"
	"Hamburger/exp/trojan/core/tunnel/freedom"
	"Hamburger/exp/trojan/core/tunnel/mux"
	"Hamburger/exp/trojan/core/tunnel/router"
	"Hamburger/exp/trojan/core/tunnel/simplesocks"
	"Hamburger/exp/trojan/core/tunnel/tls"
	"Hamburger/exp/trojan/core/tunnel/transport"
	"Hamburger/exp/trojan/core/tunnel/trojan"
)

type liteTrojan struct {
	engine *relayEngine
}

func (l *liteTrojan) Run() error {
	return l.engine.Run()
}

func (l *liteTrojan) Close() error {
	return l.engine.Close()
}

func newLiteTrojan(parent context.Context, cfg *liteConfig) (*liteTrojan, error) {
	ctx, cancel := context.WithCancel(parent)
	ctx = config.WithConfig(ctx, transport.Name, &transport.Config{
		LocalHost:  cfg.Server.ListenHost,
		LocalPort:  cfg.Server.ListenPort,
		RemoteHost: cfg.Fallback.Host,
		RemotePort: cfg.Fallback.Port,
		TransportPlugin: transport.TransportPluginConfig{
			Enabled: cfg.Plugin.Enabled,
			Type:    cfg.Plugin.Type,
			Command: cfg.Plugin.Command,
			Option:  cfg.Plugin.Option,
			Arg:     cfg.Plugin.Arg,
			Env:     cfg.Plugin.Env,
		},
	})
	ctx = config.WithConfig(ctx, tls.Name, &tls.Config{
		RemoteHost: cfg.Fallback.Host,
		RemotePort: cfg.Fallback.Port,
		TLS: tls.TLSConfig{
			CertPath:             cfg.TLS.CertPath,
			KeyPath:              cfg.TLS.KeyPath,
			KeyPassword:          cfg.TLS.KeyPassword,
			Cipher:               cfg.TLS.Cipher,
			PreferServerCipher:   cfg.TLS.PreferServerCipher,
			SNI:                  cfg.TLS.SNI,
			ALPN:                 cfg.TLS.ALPN,
			Fingerprint:          cfg.TLS.Fingerprint,
			Curves:               cfg.TLS.Curves,
			KeyLogPath:           cfg.TLS.KeyLogPath,
			CertCheckRate:        cfg.TLS.CertCheckRate,
			FallbackHost:         cfg.TLS.FallbackHost,
			FallbackPort:         cfg.TLS.FallbackPort,
			HTTPResponseFileName: cfg.TLS.PlainHTTPResponse,
		},
	})
	ctx = config.WithConfig(ctx, trojan.Name, &trojan.Config{
		LocalHost:        cfg.Server.ListenHost,
		LocalPort:        cfg.Server.ListenPort,
		RemoteHost:       cfg.Fallback.Host,
		RemotePort:       cfg.Fallback.Port,
		DisableHTTPCheck: cfg.Server.DisableHTTPCheck,
	})
	ctx = config.WithConfig(ctx, mux.Name, &mux.Config{
		Mux: mux.MuxConfig{
			Enabled:     cfg.Mux.Enabled,
			IdleTimeout: cfg.Mux.IdleTimeout,
			Concurrency: cfg.Mux.Concurrency,
		},
	})
	ctx = config.WithConfig(ctx, memory.Name, &memory.Config{
		Passwords: cfg.Auth.Passwords,
	})
	ctx = config.WithConfig(ctx, freedom.Name, &freedom.Config{})

	transportServer, err := transport.NewServer(ctx, nil)
	if err != nil {
		cancel()
		return nil, err
	}
	tlsServer, err := tls.NewServer(ctx, transportServer)
	if err != nil {
		cancel()
		return nil, err
	}
	trojanServer, err := trojan.NewServer(ctx, tlsServer)
	if err != nil {
		cancel()
		return nil, err
	}

	sources := []connSource{trojanServer}
	if cfg.Mux.Enabled {
		muxServer, muxErr := mux.NewServer(ctx, trojanServer)
		if muxErr != nil {
			cancel()
			return nil, muxErr
		}
		simplesocksServer, simpleErr := simplesocks.NewServer(ctx, muxServer)
		if simpleErr != nil {
			cancel()
			return nil, simpleErr
		}
		sources = append(sources, simplesocksServer)
	}

	var sink tunnel.Client
	sink, err = freedom.NewClient(ctx, nil)
	if err != nil {
		cancel()
		return nil, err
	}
	if cfg.Router.Enabled {
		ctx = config.WithConfig(ctx, router.Name, &router.Config{
			Router: router.RouterConfig{
				Enabled:         cfg.Router.Enabled,
				Bypass:          cfg.Router.Bypass,
				Proxy:           cfg.Router.Proxy,
				Block:           cfg.Router.Block,
				DomainStrategy:  cfg.Router.DomainStrategy,
				DefaultPolicy:   cfg.Router.DefaultPolicy,
				GeoIPFilename:   cfg.Router.GeoIP,
				GeoSiteFilename: cfg.Router.GeoSite,
			},
		})
		sink, err = router.NewClient(ctx, sink)
		if err != nil {
			cancel()
			return nil, err
		}
	}

	engine := newRelayEngine(ctx, cancel, sources, sink)
	return &liteTrojan{engine: engine}, nil
}
