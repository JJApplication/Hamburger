package app

import "Hamburger/internal/logger"

func (app *HamburgerApp) InitLogger() {
	logger.ReloadLogger(&app.conf.Log)
	app.logger = logger.GetLogger()
	app.logger.Info().Msg("init app logger success")
}
