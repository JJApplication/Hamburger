package initialize

import (
	"Hamburger/exp/vpn_proxy"
)

func (i *Initializer) InitVpnServer() Runner {
	return Runner{
		Priority: PriorityLow,
		fn: func() error {
			i.VpnServer = vpn_proxy.NewVpnServer(i.cfg, i.logger)
			return nil
		},
	}
}
