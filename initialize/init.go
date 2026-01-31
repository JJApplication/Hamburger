package initialize

import (
	"Hamburger/backend_proxy"
	"Hamburger/frontend_proxy"
	"Hamburger/gateway/core"
	"Hamburger/gateway/manager"
	"Hamburger/gateway/modifier"
	"Hamburger/internal/config"
	grpc_proxy "Hamburger/internal/grpc"
	"Hamburger/internal/logger"
	"github.com/rs/zerolog"
	"slices"
)

type Initializer struct {
	appConf *config.AppConfig
	cfg     *config.Config
	logger  *zerolog.Logger
	runners []Runner

	FrontServer     *frontend_proxy.HeliosServer
	BackendServer   *backend_proxy.BackendProxy
	Gateway         *core.Proxy
	Manager         *manager.Manager
	GrpcProxy       *grpc_proxy.GrpcProxy
	ModifierManager *modifier.ModifierManager
}

type Runner struct {
	Priority int // 优先级
	fn       func() error
}

const (
	PriorityHigh = iota
	PriorityNormal
	PriorityLow
)

func Initialize(appConf *config.AppConfig, cfg *config.Config) (*Initializer, error) {
	err := new(error)
	i := new(Initializer)
	i.appConf = appConf
	i.cfg = cfg
	i.logger = logger.GetLogger()

	i.Register(i.InitLogger())
	i.Register(i.InitMongo())
	i.Register(i.InitRuntime())
	i.Register(i.InitFrontServer())
	i.Register(i.InitGateway())
	i.Register(i.InitGatewayManager())
	i.Register(i.InitBackendServer())
	i.Register(i.InitGrpcProxy())
	i.Register(i.InitModifierManager())
	i.Register(i.InitPProf())

	// 按优先级排序
	slices.SortFunc(i.runners, func(a Runner, b Runner) int {
		return a.Priority - b.Priority
	})

	for _, runner := range i.runners {
		if e := runner.fn(); e != nil {
			err = &e
		}
	}
	return i, *err
}

func (i *Initializer) Register(runner Runner) {
	i.runners = append(i.runners, runner)
}

func (i *Initializer) GetLogger() *zerolog.Logger {
	return i.logger
}
