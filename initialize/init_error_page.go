package initialize

import (
	"Hamburger/gateway/error_page"
)

func (i *Initializer) InitPxyErrorPage() Runner {
	return Runner{
		Priority: PriorityLow,
		fn: func() error {
			error_page.InitErrorPageManager(i.cfg, i.logger)
			i.logger.Info().Msg("proxy error page initialized")
			return nil
		},
	}
}
