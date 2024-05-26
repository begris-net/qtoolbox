/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/begris-net/qtoolbox/internal/log"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"os"
)

var useCmd = &cobra.Command{
	Use:               "use",
	Short:             "use a specific version of a candidate only in the current shell",
	Aliases:           []string{"u"},
	Args:              cobra.RangeArgs(1, 2),
	Run:               useCandidateVersion,
	ValidArgsFunction: validInstalledCandidateVersions,
}

func useCandidateVersion(cmd *cobra.Command, args []string) {
	candidateName := args[0]
	var candidateVersion string

	if len(args) == 2 {
		candidateVersion = args[1]

		log.Logger.CanPrint(pterm.LogLevelInfo)
		{
			log.Logger.Debug(fmt.Sprintf("Setting version %s as temporary default for candidate %s.", candidateVersion, candidateName))
		}

		selectedCandidate := validateCandidate(candidateName, candidateVersion)

		if selectedCandidate.IsPresent() {
			pterm.Info.WithWriter(os.Stderr).Printfln("Setting %s %s as the default version for current shell.", candidateName, candidateVersion)
			log.Logger.Debug("Selecteed candidate", log.Logger.Args("candidate", selectedCandidate))

			fmt.Fprintf(os.Stdout, "%s %s %s", "use", candidateName, candidateVersion)
		} else {
			pterm.Error.WithWriter(os.Stderr).Printfln("Selected version %s for candidate %s is not installed.", candidateVersion, candidateName)
			os.Exit(1)
		}
	}
}

func init() {
	rootCmd.AddCommand(useCmd)
}
