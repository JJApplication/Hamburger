package initialize

import (
	"Hamburger/gateway/runtime"
	"time"
)

// 初始化运行时数据

func (i *Initializer) InitRuntime() Runner {
	return Runner{
		fn: func() error {
			runtime.InitRuntimeDomains(i.appConf)
			runtime.InitDomainPortsMap()
			i.logger.Info().Msg("init app runtime data success")

			go func() {
				// 延迟加载
				time.Sleep(5 * time.Second)
				runtime.InitRuntimeSyncer(i.appConf, i.logger)
			}()

			return nil
		},
	}
}
