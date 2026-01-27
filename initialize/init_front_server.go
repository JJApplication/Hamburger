package initialize

import "Hamburger/frontend_proxy"

func (i *Initializer) InitFrontServer() Runner {
	return Runner{
		Priority: PriorityNormal,
		fn: func() error {
			fs, err := frontend_proxy.NewFrontServer(i.cfg, i.logger)
			if err != nil {
				return err
			}
			i.FrontServer = fs
			i.logger.Info().Msg("frontend proxy initialized")
			return nil
		},
	}
}
