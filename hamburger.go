package main

import (
	"fmt"
	"os"

	"Hamburger/app/cli"
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
	if err := cli.Execute(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
