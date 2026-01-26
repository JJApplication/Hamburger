package initialize

import (
	"Hamburger/gateway/breaker"
	"Hamburger/gateway/core"
)

func (i *Initializer) InitGateway() Runner {
	return Runner{
		fn: func() error {
			gw := core.NewProxy(i.cfg, i.logger)
			i.Gateway = gw
			breaker.InitBreaker()
			i.logger.Info().Msg("init app gw proxy success")
			return nil
		},
	}
}
