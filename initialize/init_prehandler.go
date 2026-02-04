package initialize

import (
	"Hamburger/gateway/prehandler"
)

func (i *Initializer) InitPreHandlerManager() Runner {
	return Runner{
		Priority: PriorityLow,
		fn: func() error {
			prehandler.InitPreHandlerManager()
			return nil
		},
	}
}
