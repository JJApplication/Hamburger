package initialize

import (
	"Hamburger/gateway/manager"
)

func (i *Initializer) InitGatewayManager() Runner {
	return Runner{
		fn: func() error {
			mg := manager.NewManager(i.cfg, i.logger, i.Gateway.Create())
			i.Manager = mg
			return nil
		}}
}
