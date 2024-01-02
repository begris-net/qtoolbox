/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/BooleanCat/go-functional/iter"
	"github.com/begris-net/qtoolbox/internal/candidate"
	"github.com/begris-net/qtoolbox/internal/config"
	"github.com/begris-net/qtoolbox/internal/log"
	"github.com/begris-net/qtoolbox/internal/repository"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "qtoolbox",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Aliases: []string{"qtb", "tb"},
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log.DebugLevel = Debug
		log.SetLogLevel(log.DebugLevel)
		config.LoadConfig(cfgFile)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var cfgFile string
var Debug int8

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "",
		"config file (default is $HOME/.qtoolbox/config/config.yaml)")
	rootCmd.PersistentFlags().Int8VarP(&Debug, "debug", "v", 1, "Debug messages")
}

func ValidCandidates(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	repositoryConfig := repository.GetRepository()
	candidates := repositoryConfig.ListCandidates()

	var completionCandidates []string
	iter.Lift(candidates).Filter(func(v candidate.CandidateDescription) bool {
		return strings.HasPrefix(v.Name, toComplete)
	}).ForEach(func(description candidate.CandidateDescription) {
		completionCandidates = append(completionCandidates, description.Name)
	})

	return completionCandidates, cobra.ShellCompDirectiveNoFileComp
}
