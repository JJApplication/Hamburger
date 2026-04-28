package main

import (
	"fmt"
	"os"

	"Hamburger/app/cli"
	"Hamburger/internal/buildinfo"
)

func main() {
	fmt.Printf("%s - %s\n", buildinfo.AppName, buildinfo.Description)
	fmt.Println("")
	cli.OutputHamburgerLogo()
	fmt.Println("")
	fmt.Printf("Version: %s\nBuildHash: %s\n\n", buildinfo.Version, buildinfo.BuildHash)
	if err := cli.Execute(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
