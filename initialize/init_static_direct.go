package initialize

import (
	"Hamburger/static_direct"
)

func (i *Initializer) InitStaticDirect() Runner {
	return Runner{
		Priority: PriorityLow,
		fn: func() error {
			sd := static_direct.NewStaticDirectServer(i.cfg, i.logger)
			i.StaticDirectSvr = sd
			i.logger.Info().Msg("init app static direct success")
			return nil
		},
	}
}
