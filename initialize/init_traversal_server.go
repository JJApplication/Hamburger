package initialize

import (
	"Hamburger/exp/traversal"
)

// InitTraversalServer 初始化内网穿透服务端。
func (i *Initializer) InitTraversalServer() Runner {
	return Runner{
		Priority: PriorityLow,
		fn: func() error {
			i.TraversalServer = traversal.NewServer(i.cfg, i.logger)
			return nil
		},
	}
}
