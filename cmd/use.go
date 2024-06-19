/*
 * Copyright (c) 2024 Bjoern Beier.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of
 * this software and associated documentation files (the "Software"), to deal in
 * the Software without restriction, including without limitation the rights to
 * use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
 * the Software, and to permit persons to whom the Software is furnished to do so,
 * subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
 * FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
 * COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
 * IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
 * CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */
package cmd

import (
	"fmt"
	"github.com/begris-net/qtoolbox/internal/log"
	"github.com/begris-net/qtoolbox/internal/types"
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

			fmt.Fprintf(os.Stdout, "%s %s %s %s", types.PostprocessingFlag, "use", candidateName, candidateVersion)
		} else {
			pterm.Error.WithWriter(os.Stderr).Printfln("Selected version %s for candidate %s is not installed.", candidateVersion, candidateName)
			os.Exit(1)
		}
	}
}

func init() {
	rootCmd.AddCommand(useCmd)
}
