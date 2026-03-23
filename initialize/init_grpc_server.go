package initialize

import (
	"Hamburger/backend_proxy"
	"Hamburger/exp/any_tls"
	"Hamburger/exp/trojan"
	"Hamburger/exp/vpn_proxy"
	"Hamburger/frontend_proxy"
	"Hamburger/gateway/api"
	"Hamburger/gateway/latency"
	"Hamburger/gateway/manager"
	"Hamburger/gateway/modifier"
	"Hamburger/grpc_server"
)

func (i *Initializer) InitGRPCServer() Runner {
	return Runner{
		Priority: PriorityFinal,
		fn: func() error {
			gs, err := grpc_server.NewServer(i.cfg, i.logger,
				func() *manager.Manager { return i.Manager },
				func() *frontend_proxy.HeliosServer { return i.FrontServer },
				func() *modifier.ModifierManager { return i.ModifierManager },
				func() *api.Server { return i.APIServer },
				func() *backend_proxy.BackendProxy { return i.BackendServer },
				func() *latency.LatencyServer { return i.LatencyServer },
				func() *vpn_proxy.VpnServer { return i.VpnServer },
				func() *trojan.TrojanServer { return i.TrojanServer },
				func() *any_tls.AnyTLSServer { return i.AnyTLSServer })
			if err != nil {
				return err
			}
			i.GrpcServer = gs
			i.logger.Info().Msg("grpc server initialized")
			return nil
		},
	}
}
