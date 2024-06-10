/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"fmt"
	"github.com/begris-net/qtoolbox/internal/config"
	"github.com/begris-net/qtoolbox/internal/config/defaults"
	"github.com/begris-net/qtoolbox/internal/log"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"io"
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
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
	},
}

func setup(cmd *cobra.Command, args []string) {
	homeDir, _ := homedir.Dir()
	log.Logger.Info("Creating setup in", log.Logger.Args("directory", fmt.Sprintf("%v", homeDir)))

	if _, err := os.Stat(filepath.Join(homeDir, config.QToolboxDirectory)); err == nil {
		msg := "Found existing config"
		if !force {
			log.Logger.Error(msg + "... aborting.")
			os.Exit(1)
		} else {
			log.Logger.Warn(msg + "... Overriding current configuration.")
		}
	}

	dirs, _ := defaults.Default.ReadDir(".")
	log.Logger.Info("Extracting qtoolbox installation.")
	extractInstallation(dirs, homeDir, ".", 0)
	qToolboxBinary := filepath.Join(homeDir, config.QToolboxDirectory, "bin", filepath.Base(os.Args[0]))
	log.Logger.Info("Installing qtoolbox binary...")
	err := installQtoolbox(os.Args[0], qToolboxBinary)
	if err != nil {
		log.Logger.Fatal("Error installing qtoolbox binary.", log.Logger.Args("err", err))
	}
	log.Logger.Info("Integrating qtoolbox in shell...")
	integrateShell(homeDir)
	log.Logger.Warn("Restart your shell or call manually", log.Logger.Args("cmd", "source ~/.qtoolbox/bin/qtoolbox-init.sh"))
}

func installQtoolbox(src string, dst string) error {
	err := copyBinary(src, dst)
	if err != nil {
		return err
	}
	permExec := os.FileMode(0750)
	err = os.Chmod(dst, permExec)
	if err != nil {
		return err
	}
	return nil
}

func copyBinary(src string, dst string) error {
	log.Logger.Info("Copying qtoolbox binary.", log.Logger.Args("src", src, "dst", dst))
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return nil
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
		log.Logger.Trace(fmt.Sprint(strings.Repeat(" ", indent*2), entry.Name()))
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

const shell_integration_line = "[[ -s \"$HOME/.qtoolbox/bin/qtoolbox-init.sh\" ]] && source \"$HOME/.qtoolbox/bin/qtoolbox-init.sh\""

func updateShellRC(rcfile string) {
	stat, err2 := os.Stat(rcfile)
	if err2 != nil {
		log.Logger.Error(fmt.Sprintf("Error opening recource file for shell. (stat)", log.Logger.Args("err", err2, "file-name", rcfile)))
	}
	var file, err = os.OpenFile(rcfile, os.O_APPEND|os.O_RDWR, stat.Mode())
	if err != nil {
		log.Logger.Error(fmt.Sprintf("Error opening recource file for shell.", log.Logger.Args("err", err, "file-name", rcfile)))
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), shell_integration_line) {
			log.Logger.Info(fmt.Sprintf("Already integrated into shell."))
			return
		}
	}
	if err = scanner.Err(); err != nil {
		log.Logger.Error(fmt.Sprintf("Error while scanning recource file.", log.Logger.Args("err", err, "file-name", rcfile)))
	} else {
		bytesWritten, err := file.WriteString(fmt.Sprintf("\n%s\n", shell_integration_line))
		if err != nil {
			log.Logger.Error(fmt.Sprintf("Error during updating recource file.", log.Logger.Args("err", err, "bytes-written", bytesWritten)))
		}
	}
}

func integrateShell(homedir string) {
	shell := os.Getenv("SHELL")
	shell = filepath.Base(shell)
	switch shell {
	case "zsh":
		log.Logger.WithWriter(os.Stderr).Info(fmt.Sprintf("Found zsh."))
		updateShellRC(path.Join(homedir, ".zshrc"))
	case "bash":
		log.Logger.WithWriter(os.Stderr).Info(fmt.Sprintf("Found bash."))
		updateShellRC(path.Join(homedir, ".bashrc"))
	case "fish", "powershell":
		log.Logger.Warn(fmt.Sprintf("Automatic integration for %s not supported yet. Please ensure to include the init command into your shell.", shell), log.Logger.Args("cmd", "source ~/.qtoolbox/bin/qtoolbox-init.sh"))
	case "cmd", "csh":
		log.Logger.Error(fmt.Sprintf("%s is not supported.", shell))
	}
}

var force bool

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&force, "force", "f", false, "Override existing configuration")
}
