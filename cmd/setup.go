/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/begris-net/qtoolbox/internal/config"
	"github.com/begris-net/qtoolbox/internal/config/defaults"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:    "setup",
	Hidden: true,
	Short:  "setup qtoolbox installation",
	Long: `Prepares and installs the qtoolbox enviroment folder
and adds the toolbox command to your system PATH.`,
	Run: setup,
}

func setup(cmd *cobra.Command, args []string) {
	homeDir, _ := homedir.Dir()
	fmt.Printf("%v", homeDir)

	if _, err := os.Stat(filepath.Join(homeDir, config.QToolboxDirectory)); err == nil {
		print("Found existing config")
		if !force {
			println("... aborting.")
			os.Exit(1)
		} else {
			println("... Overriding current configuration.")
		}
	}

	dirs, _ := defaults.Default.ReadDir(".")
	extractInstallation(dirs, homeDir, ".", 0)
}

func extractInstallation(dirs []fs.DirEntry, homeDir string, parent string, indent int) {
	permDir := os.FileMode(0750)
	permExec := os.FileMode(0750)
	permConfig := os.FileMode(0640)
	err := os.MkdirAll(path.Join(homeDir, parent), permDir)
	if err != nil {
		panic(err)
	}
	for _, entry := range dirs {
		println(strings.Repeat(" ", indent*2), entry.Name())
		if entry.IsDir() {
			subpath := path.Join(parent, entry.Name())
			childs, err := defaults.Default.ReadDir(subpath)
			if err != nil {
				panic(err)
			}
			extractInstallation(childs, homeDir, subpath, indent+1)
		} else if !entry.IsDir() {
			filepath := path.Join(parent, entry.Name())

			bytes, err := defaults.Default.ReadFile(filepath)
			if err != nil {
				panic(err)
			}
			destFilepath := path.Join(homeDir, filepath)
			err = os.WriteFile(destFilepath, bytes, permConfig)
			if err != nil {
				panic(err)
			}

			var mode os.FileMode
			switch path.Ext(filepath) {
			case ".yaml", ".gitkeep":
				mode = permConfig
			default:
				mode = permExec
			}

			err = os.Chmod(destFilepath, mode)
			if err != nil {
				panic(err)
			}
		}
	}
}

var force bool

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&force, "force", "f", false, "Override existing configuration")
}
