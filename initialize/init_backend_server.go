package initialize

import "Hamburger/backend_proxy"

func (i *Initializer) InitBackendServer() Runner {
	return Runner{
		fn: func() error {
			bp := backend_proxy.NewBackendProxy(i.cfg, i.logger)
			i.BackendServer = bp
			i.logger.Info().Msg("backend proxy initialized")
			return nil
		},
	}
}
