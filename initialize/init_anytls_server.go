package initialize

import (
	"Hamburger/exp/any_tls"
)

func (i *Initializer) InitAnyTLSServer() Runner {
	return Runner{
		Priority: PriorityLow,
		fn: func() error {
			i.AnyTLSServer = any_tls.NewAnyTLSServer(i.cfg, i.logger)
			return nil
		},
	}
}
