package main

import (
	"fmt"

	"Hamburger/app"
)

var (
	AppName     = "Hamburger"
	Description = "Next Generation Gateway for JJApps."
	Version     string
	BuildHash   string
)

func main() {
	fmt.Printf("%s - %s\n", AppName, Description)
	fmt.Printf("Version: %s\nBuildHash: %s\n\n", Version, BuildHash)
	hamburger := app.NewHamburgerApp()
	hamburger.Run()

	select {}
}
