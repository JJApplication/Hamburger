package app

import "Hamburger/frontend_proxy"

func (app *HamburgerApp) InitFrontServer(e *error) {
	fs, err := frontend_proxy.NewFrontServer(app.conf, app.logger)
	if err != nil {
		*e = err
		return
	}
	app.FrontServer = fs
	app.logger.Info().Msg("init app front server success")
}
