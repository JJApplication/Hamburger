package initialize

import (
	"Hamburger/gateway/manager"
	"Hamburger/internal/config"
	"Hamburger/internal/connectprotocol"
)

func (i *Initializer) InitGatewayManager() Runner {
	return Runner{
		Priority: PriorityLow,
		fn: func() error {
			gatewayHandler := i.Gateway.Create()
			initial := config.ConnectProtocolConfig{}
			if i.cfg != nil {
				initial = i.cfg.ConnectProtocol
			}
			state, err := connectprotocol.NewConfigState(initial)
			if err != nil {
				return err
			}
			connectHandler, err := connectprotocol.New(gatewayHandler, connectprotocol.NewService(i.APIService), state.Load, i.logger)
			if err != nil {
				return err
			}
			i.ConnectProtocol = connectHandler
			i.ConnectConfig = state
			mg := manager.NewManager(i.cfg, i.logger, connectHandler)
			i.Manager = mg
			return nil
		}}
}
