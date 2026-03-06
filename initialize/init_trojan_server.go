package initialize

import (
	"Hamburger/exp/trojan"
)

func (i *Initializer) InitTrojanServer() Runner {
	return Runner{
		Priority: PriorityLow,
		fn: func() error {
			svr, err := trojan.NewTrojanServer(i.cfg, i.logger)
			if err != nil {
				return err
			}
			i.TrojanServer = svr
			return nil
		},
	}
}
