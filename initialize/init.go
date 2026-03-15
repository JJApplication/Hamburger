package initialize

import (
	"Hamburger/backend_proxy"
	"Hamburger/exp/any_tls"
	"Hamburger/exp/trojan"
	"Hamburger/exp/vpn_proxy"
	"Hamburger/frontend_proxy"
	"Hamburger/gateway/api"
	"Hamburger/gateway/core"
	"Hamburger/gateway/latency"
	"Hamburger/gateway/manager"
	"Hamburger/gateway/modifier"
	"Hamburger/gateway/notifier"
	"Hamburger/gateway/stat"
	"Hamburger/grpc_server"
	"Hamburger/internal/config"
	grpc_proxy "Hamburger/internal/grpc"
	"Hamburger/internal/logger"
	"Hamburger/static_direct"
	"slices"

	"github.com/rs/zerolog"
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
	StatManager     *stat.StatManager
	APIServer       *api.Server
	LatencyServer   *latency.LatencyServer
	Notifier        *notifier.Service
	StaticDirectSvr *static_direct.StaticDirectServer
	VpnServer       *vpn_proxy.VpnServer
	AnyTLSServer    *any_tls.AnyTLSServer
	TrojanServer    *trojan.TrojanServer
	GrpcServer      *grpc_server.AppServiceServer
}

type Runner struct {
	Priority int // 优先级
	fn       func() error
}

const (
	PriorityHigh = iota
	PriorityNormal
	PriorityLow
	PriorityFinal // 最后初始化
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
	i.Register(i.InitPreHandlerManager())
	i.Register(i.InitNotifier())
	i.Register(i.InitStatManager())
	i.Register(i.InitAPIServer())
	i.Register(i.InitLatencyServer())
	i.Register(i.InitStaticDirect())
	i.Register(i.InitVpnServer())
	i.Register(i.InitAnyTLSServer())
	i.Register(i.InitTrojanServer())
	i.Register(i.InitProbeSyncer())
	i.Register(i.InitPxyErrorPage())
	i.Register(i.InitPProf())
	i.Register(i.InitGRPCServer())

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
