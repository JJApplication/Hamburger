package app

import (
	"Hamburger/gateway/breaker"
	"Hamburger/gateway/core"
)

func (app *HamburgerApp) InitGateway() {
	gw := core.NewProxy(app.conf, app.logger)
	app.Gateway = gw
	breaker.InitBreaker()
	app.logger.Info().Msg("init app gw proxy success")
}
