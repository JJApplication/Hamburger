package main

import "github.com/fatih/color"

var (
	promptDollar = color.New(color.FgHiGreen).SprintFunc()
	commandText  = color.New(color.FgBlue).SprintFunc()
	flagText     = color.New(color.FgHiBlack).SprintFunc()
	errorText    = color.New(color.FgHiRed).SprintFunc()
)

