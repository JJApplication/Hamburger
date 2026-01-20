package app

import "Hamburger/internal/data"

func (app *HamburgerApp) InitMongo() {
	data.InitMongo()
}
