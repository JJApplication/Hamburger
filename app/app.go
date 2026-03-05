package app

import (
	appgrpc "Hamburger/app/grpc"
	"Hamburger/backend_proxy"
	"Hamburger/exp/any_tls"
	"Hamburger/exp/vpn_proxy"
	"Hamburger/frontend_proxy"
	"Hamburger/gateway/core"
	"Hamburger/gateway/latency"
	"Hamburger/gateway/manager"
	"Hamburger/gateway/modifier"
	"Hamburger/gateway/stat"
	"Hamburger/grpc_server"
	"Hamburger/initialize"
	"Hamburger/internal/config"
	grpc_proxy "Hamburger/internal/grpc"
	"Hamburger/internal/logger"
	"Hamburger/static_direct"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

type HamburgerApp struct {
	appConf *config.AppConfig
	conf    *config.Config

	logger      *zerolog.Logger // APP日志
	pidFile     string
	serverCount int

	// Proxy
	FrontServer     *frontend_proxy.HeliosServer
	BackendServer   *backend_proxy.BackendProxy
	Gateway         *core.Proxy
	Manager         *manager.Manager
	GrpcProxy       *grpc_proxy.GrpcProxy
	ModifierManager *modifier.ModifierManager
	StatServer      *stat.StatServer
	LatencyServer   *latency.LatencyServer
	StaticDirectSvr *static_direct.StaticDirectServer
	VpnServer       *vpn_proxy.VpnServer
	AnyTLSServer    *any_tls.AnyTLSServer

	grpcServer   *grpc.Server
	grpcListener net.Listener
}

const (
	DefaultConfigFile = "config.json"
)

func NewHamburgerApp(configFile string) (*HamburgerApp, error) {
	if configFile == "" {
		configFile = DefaultConfigFile
	}
	logger.InitLogger()
	appCfg, err := config.LoadConfig(configFile)
	if err != nil {
		return nil, err
	}

	cfg := config.Merge(appCfg)
	config.Set(cfg)

	return &HamburgerApp{
		appConf: appCfg,
		conf:    cfg,
		logger:  logger.GetLogger(),
	}, nil
}

func (app *HamburgerApp) registerServerCount(assign func() int) {
	if assign() > 0 {
		app.serverCount++
	}
}

func (app *HamburgerApp) InitApp() error {
	i, err := initialize.Initialize(app.appConf, app.conf)
	if err != nil {
		return err
	}
	app.registerServerCount(func() int { app.FrontServer = i.FrontServer; return 1 })
	app.registerServerCount(func() int { app.BackendServer = i.BackendServer; return 1 })
	app.registerServerCount(func() int { app.Gateway = i.Gateway; return 0 })
	app.registerServerCount(func() int { app.Manager = i.Manager; return 1 })
	app.registerServerCount(func() int { app.GrpcProxy = i.GrpcProxy; return 0 })
	app.registerServerCount(func() int { app.ModifierManager = i.ModifierManager; return 0 })
	app.registerServerCount(func() int { app.StatServer = i.StatServer; return 1 })
	app.registerServerCount(func() int { app.LatencyServer = i.LatencyServer; return 1 })
	app.registerServerCount(func() int { app.StaticDirectSvr = i.StaticDirectSvr; return 1 })
	app.registerServerCount(func() int { app.VpnServer = i.VpnServer; return 1 })
	app.registerServerCount(func() int { app.AnyTLSServer = i.AnyTLSServer; return 1 })
	app.logger = i.GetLogger()

	return nil
}

func (app *HamburgerApp) Run() {
	if err := app.InitApp(); err != nil {
		app.logger.Fatal().Err(err).Msg("init app failed")
		return
	}

	app.Status()
	app.LifeCycle()

	wg := sync.WaitGroup{}
	wg.Add(app.serverCount)

	grpcAddr := ""
	if app.conf.GRPC.Enabled {
		grpcAddr = strings.TrimSpace(app.conf.GRPC.Address)
	}
	if grpcAddr != "" {
		listener, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			app.logger.Fatal().Err(err).Str("address", grpcAddr).Msg("grpc server listen error")
		}
		app.grpcListener = listener
		app.grpcServer = grpc.NewServer()
		appgrpc.RegisterAppServiceServer(app.grpcServer, grpc_server.NewAppService(
			func() *manager.Manager { return app.Manager },
			func() *frontend_proxy.HeliosServer { return app.FrontServer },
			func() *modifier.ModifierManager { return app.ModifierManager },
			func() *stat.StatServer { return app.StatServer },
		))
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := app.grpcServer.Serve(listener); err != nil {
				app.logger.Fatal().Err(err).Msg("grpc server error")
			}
		}()
		app.logger.Info().Str("address", grpcAddr).Msg("grpc server started")
	}

	go func() {
		defer wg.Done()
		if err := app.FrontServer.Start(); err != nil {
			app.logger.Fatal().Err(err).Msg("frontend server error")
		}
	}()

	go func() {
		defer wg.Done()
		if err := app.Manager.Start(); err != nil {
			app.logger.Fatal().Err(err).Msg("gateway server error")
		}
	}()

	go func() {
		defer wg.Done()
		app.BackendServer.Start()
	}()

	go func() {
		defer wg.Done()
		if err := app.StatServer.Start(); err != nil {
			app.logger.Fatal().Err(err).Msg("stat server error")
		}
	}()

	go func() {
		defer wg.Done()
		if err := app.LatencyServer.Start(); err != nil {
			app.logger.Fatal().Err(err).Msg("latency server error")
		}
	}()

	go func() {
		defer wg.Done()
		if err := app.StaticDirectSvr.Start(); err != nil {
			app.logger.Fatal().Err(err).Msg("static direct server error")
		}
	}()

	go func() {
		defer wg.Done()
		if err := app.VpnServer.Start(); err != nil {
			app.logger.Fatal().Err(err).Msg("vpn server error")
		}
	}()

	go func() {
		defer wg.Done()
		if err := app.AnyTLSServer.Start(); err != nil {
			app.logger.Fatal().Err(err).Msg("any tls server error")
		}
	}()

	wg.Wait()
	app.removePidFile()
}

// Status 输出服务器状态信息
func (app *HamburgerApp) Status() {
	app.FrontServer.Status()

	app.BackendServer.Status()

	gwServerStatus := app.Manager.GetServerStatus()
	gwHttp3ServerStatus := app.Manager.GetHttp3ServerStatus()

	for _, server := range gwServerStatus {
		app.logger.Info().Str("name", server.Name).Bool("running", server.Started).Msg("[gateway proxy] server status")
	}
	for _, server := range gwHttp3ServerStatus {
		app.logger.Info().Str("name", server.Name).Bool("running", server.IsStarted()).Msg("[gateway proxy] server status")
	}
}

func (app *HamburgerApp) LifeCycle() {
	// 设置信号处理
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// 启动优雅关闭监听
	go func() {
		<-c
		app.logger.Info().Msg("received shutdown signal, gracefully shutting down...")
		app.FrontServer.Shutdown()
		app.BackendServer.Stop()
		if err := app.Manager.Stop(); err != nil {
			app.logger.Error().Err(err).Msg("gateway server shutdown failed")
		}
		if err := app.StatServer.Stop(); err != nil {
			app.logger.Error().Err(err).Msg("stat server shutdown failed")
		}
		if err := app.LatencyServer.Stop(); err != nil {
			app.logger.Error().Err(err).Msg("latency server shutdown failed")
		}
		if app.VpnServer != nil {
			_ = app.VpnServer.Stop()
		}
		if app.grpcServer != nil {
			app.grpcServer.GracefulStop()
		}
		if app.grpcListener != nil {
			_ = app.grpcListener.Close()
		}
		app.removePidFile()
		os.Exit(0)
	}()
}

func (app *HamburgerApp) SetPidFile(pidFile string) {
	app.pidFile = pidFile
}

func (app *HamburgerApp) removePidFile() {
	if app.pidFile == "" {
		return
	}
	_ = os.Remove(app.pidFile)
}
