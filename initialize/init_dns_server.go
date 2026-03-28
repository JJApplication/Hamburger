package initialize

import (
	exp_dns "Hamburger/exp/dns"
)

func (i *Initializer) InitDNSServer() Runner {
	return Runner{
		Priority: PriorityLow,
		fn: func() error {
			i.DNSServer = exp_dns.NewDNSServer(i.cfg, i.logger)
			return nil
		},
	}
}
