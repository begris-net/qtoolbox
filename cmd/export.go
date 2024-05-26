/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/begris-net/qtoolbox/internal/repository"
	"github.com/begris-net/qtoolbox/internal/util"
	"github.com/spf13/cobra"
	"os"
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:               "export",
	Short:             "get the additional path to export for a candidate, if required",
	Hidden:            true,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: validCandidates,

	Run: func(cmd *cobra.Command, args []string) {
		CandidatePreRun(cmd, args)
		repositoryConfig := repository.GetRepository()
		candidate := repositoryConfig.FindCandidate(CandidateName)
		if candidate.ExportPath != nil {
			fmt.Fprintf(os.Stdout, "/%s", util.SafeDeref(candidate.ExportPath))
		}
	},
}

func init() {
	candidateCmd.AddCommand(exportCmd)
}
