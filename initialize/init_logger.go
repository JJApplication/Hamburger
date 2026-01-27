package initialize

import (
	"Hamburger/internal/logger"
)

func (i *Initializer) InitLogger() Runner {
	return Runner{
		Priority: PriorityHigh,
		fn: func() error {
			logger.ReloadLogger(&i.cfg.Log)
			i.logger = logger.GetLogger()
			i.logger.Info().Msg("init app logger success")
			return nil
		},
	}
}
