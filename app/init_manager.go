package app

import (
	"Hamburger/gateway/manager"
)

func (app *HamburgerApp) InitGatewayManager() {
	mg := manager.NewManager(app.conf, app.logger, app.Gateway.Create())
	app.Manager = mg
}
