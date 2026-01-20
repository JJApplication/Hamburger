package app

import (
	"Hamburger/gateway/runtime"
	"time"
)

// 初始化运行时数据

func (app *HamburgerApp) InitRuntime() {
	runtime.InitRuntimeDomains(app.appConf)
	runtime.InitDomainPortsMap()
	app.logger.Info().Msg("init app runtime data success")

	go func() {
		// 延迟加载
		time.Sleep(5 * time.Second)
		runtime.InitRuntimeSyncer(app.appConf, app.logger)
	}()
}
