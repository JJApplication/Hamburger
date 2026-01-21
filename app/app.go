package app

import (
	"Hamburger/frontend_proxy"
	"Hamburger/gateway/core"
	"Hamburger/gateway/manager"
	"Hamburger/internal/config"
	"Hamburger/internal/logger"
	"flag"
	"github.com/rs/zerolog"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type HamburgerApp struct {
	appConf *config.AppConfig
	conf    *config.Config

	logger *zerolog.Logger // APP日志

	// Proxy
	FrontServer *frontend_proxy.HeliosServer
	Gateway     *core.Proxy
	Manager     *manager.Manager
}

const (
	DefaultConfigFile = "config.json"
)

var (
	ConfigFile     = flag.String("config", "config.json", "config file")
	GenerateConfig = flag.Bool("generate", false, "generate config file")
)

func NewHamburgerApp() *HamburgerApp {
	flag.Parse()

	if *GenerateConfig {
		_ = config.CreateConfig()
		os.Exit(0)
	}
	if *ConfigFile == "" {
		*ConfigFile = DefaultConfigFile
	}

	logger.InitLogger()
	appCfg, err := config.LoadConfig(*ConfigFile)
	if err != nil {
		logger.GetLogger().Fatal().Err(err).Msg("load config file failed")
		panic(err)
	}

	cfg := config.Merge(appCfg)
	config.Set(cfg)

	return &HamburgerApp{
		appConf: appCfg,
		conf:    cfg,
	}
}

func (app *HamburgerApp) InitApp() error {
	err := new(error)
	// init logger
	app.InitLogger()
	// init database
	app.InitMongo()
	// init runtime
	app.InitRuntime()
	// init front proxy
	app.InitFrontServer(err)
	// init gateway proxy
	app.InitGateway()
	// init gw server manager
	app.InitGatewayManager()
	return *err
}

func (app *HamburgerApp) Run() {
	if err := app.InitApp(); err != nil {
		app.logger.Fatal().Err(err).Msg("init app failed")
		return
	}

	app.Status()
	app.LifeCycle()

	wg := sync.WaitGroup{}
	wg.Add(2)

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

	wg.Wait()
}

// Status 输出服务器状态信息
func (app *HamburgerApp) Status() {
	app.FrontServer.Status()

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
		if err := app.Manager.Stop(); err != nil {
			app.logger.Error().Err(err).Msg("gateway server shutdown failed")
		}
		os.Exit(0)
	}()
}
