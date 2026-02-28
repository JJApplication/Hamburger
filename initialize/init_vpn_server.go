package initialize

import "Hamburger/vpn_proxy"

func (i *Initializer) InitVpnServer() Runner {
	return Runner{
		Priority: PriorityNormal,
		fn: func() error {
			i.VpnServer = vpn_proxy.NewVpnServer(i.cfg, i.logger)
			return nil
		},
	}
}
