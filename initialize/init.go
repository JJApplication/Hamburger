package initialize

import (
	"Hamburger/backend_proxy"
	"Hamburger/frontend_proxy"
	"Hamburger/gateway/core"
	"Hamburger/gateway/manager"
	"Hamburger/internal/config"
	"github.com/rs/zerolog"
	"slices"
)

type Initializer struct {
	appConf *config.AppConfig
	cfg     *config.Config
	logger  *zerolog.Logger
	runners []Runner

	FrontServer   *frontend_proxy.HeliosServer
	BackendServer *backend_proxy.BackendProxy
	Gateway       *core.Proxy
	Manager       *manager.Manager
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

func Initialize() (*Initializer, error) {
	err := new(error)
	i := new(Initializer)

	i.Register(i.InitLogger(), PriorityHigh)
	i.Register(i.InitMongo(), PriorityHigh)
	i.Register(i.InitRuntime(), PriorityHigh)
	i.Register(i.InitFrontServer(), PriorityNormal)
	i.Register(i.InitGateway(), PriorityNormal)
	i.Register(i.InitGatewayManager(), PriorityLow)
	i.Register(i.InitBackendServer(), PriorityLow)

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

func (i *Initializer) Register(runner Runner, priority int) {
	runner.Priority = priority
	i.runners = append(i.runners, runner)
}
