package initialize

import (
	"Hamburger/frontend_proxy"
	"Hamburger/gateway/api"
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
				func() *api.Server { return i.APIServer })
			if err != nil {
				return err
			}
			i.GrpcServer = gs
			i.logger.Info().Msg("grpc server initialized")
			return nil
		},
	}
}
