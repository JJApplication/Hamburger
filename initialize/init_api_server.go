package initialize

import "Hamburger/gateway/api"

func (i *Initializer) InitAPIServer() Runner {
	return Runner{
		Priority: PriorityLow,
		fn: func() error {
			i.APIServer = api.NewAPIServer(i.cfg, i.logger)
			return nil
		}}
}
