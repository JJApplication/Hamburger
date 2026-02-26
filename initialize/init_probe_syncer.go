package initialize

import (
	"Hamburger/gateway/health_probe"
	"time"
)

func (i *Initializer) InitProbeSyncer() Runner {
	return Runner{
		Priority: PriorityNormal,
		fn: func() error {
			go func() {
				// 延迟加载
				time.Sleep(5 * time.Second)
				health_probe.InitProbeSyncer(i.cfg, i.logger)
			}()

			return nil
		},
	}
}
