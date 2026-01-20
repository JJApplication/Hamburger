package main

import (
	"Hamburger/app"
	"fmt"
)

var (
	AppName     = "Hamburger"
	Description = "Next Generation Gateway for JJApps."
	Version     string
	BuildHash   string
)

func main() {
	fmt.Printf("%s - %s\n", AppName, Description)
	fmt.Printf("Version: %s\nBuildHash: %s\n", Version, BuildHash)
	hamburger := app.NewHamburgerApp()
	hamburger.Run()

	select {}
}
