/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/begris-net/qtoolbox/internal/log"
	"github.com/begris-net/qtoolbox/internal/repository"
	"github.com/davecgh/go-spew/spew"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// candidateCmd represents the candidate command
var candidateCmd = &cobra.Command{
	Use:   "candidate",
	Short: "commands internally used for candidate management",
}

func CandidatePreRun(cmd *cobra.Command, args []string) {
	rootCmd.PersistentPreRun(cmd, args)
	log.Logger.Debug("list called with arguments", log.Logger.Args("args", args))
	hasCandidate := false
	if len(args) > 0 {
		CandidateName = args[0]
		hasCandidate = true
		log.Logger.Debug("Called with candidate", log.Logger.Args("candicate", CandidateName), log.Logger.Args("hasCandicate", hasCandidate))
	}

	repositoryConfig := repository.GetRepository()
	if log.Logger.CanPrint(pterm.LogLevelTrace) {
		log.Logger.Trace("Repository config dump:")
		spew.Fdump(log.Logger.Writer, repositoryConfig)
	}

	if !hasCandidate {
		// show all candidate descriptions
		// TODO need some fancy UI stuff here - scrollable view
		for _, c := range repositoryConfig.ListCandidates() {
			c.Show()
		}
	}
}

func init() {
	rootCmd.AddCommand(candidateCmd)
}
