package initialize

import grpc_proxy "Hamburger/internal/grpc"

func (i *Initializer) InitGrpcProxy() Runner {
	return Runner{
		Priority: PriorityLow,
		fn: func() error {
			grpc_proxy.InitGrpcProxy(&i.cfg.Features.GrpcProxy, i.logger)
			i.logger.Info().Msg("grpc proxy initialized")
			i.GrpcProxy = grpc_proxy.GetGrpcProxy()
			return nil
		},
	}
}
