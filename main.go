/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"github.com/begris-net/qtoolbox/cmd"
)

var Version = "develop"

func main() {
	cmd.SetVersionInformation(Version)
	cmd.Execute()
}
