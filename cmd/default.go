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
	"github.com/YoshikiShibata/gostream"
	"github.com/begris-net/qtoolbox/internal/candidate"
	"github.com/begris-net/qtoolbox/internal/log"
	"github.com/begris-net/qtoolbox/internal/repository"
	"github.com/begris-net/qtoolbox/internal/types"
	"github.com/davecgh/go-spew/spew"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

// defaultCmd represents the default command
var defaultCmd = &cobra.Command{
	Use:               "default",
	Short:             "set the default version to use for a candidate",
	Aliases:           []string{"d"},
	Args:              cobra.RangeArgs(1, 2),
	Run:               makeCandidateVersionDefault,
	ValidArgsFunction: validInstalledCandidateVersions,
}

func validInstalledCandidateVersions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cmd.Flags().Parse(args)
	cleanedArgs := cmd.Flags().Args()

	if len(cleanedArgs) >= 2 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	if len(cleanedArgs) >= 1 {
		return installedCandidateVersions(cleanedArgs[0], toComplete)
	} else {
		return ValidCandidates(cmd, cleanedArgs, toComplete)
	}
}

func installedCandidateVersions(candidateName string, toComplete string) ([]string, cobra.ShellCompDirective) {
	repositoryConfig := repository.GetRepository()
	if log.Logger.CanPrint(pterm.LogLevelTrace) {
		log.Logger.Trace("Repository config dump:")
		spew.Fdump(log.Logger.Writer, repositoryConfig)
	}

	_, candidateVersions, _ := repositoryConfig.ListCandidateVersions(candidateName)

	completionCandidates := gostream.FlatMap(gostream.Of(candidateVersions...).Filter(func(v candidate.Candidate) bool {
		return v.Installed && strings.HasPrefix(v.DisplayName, toComplete)
	}).Sorted(func(a, b candidate.Candidate) int {
		return b.Version.Compare(&a.Version)
	}), func(v candidate.Candidate) gostream.Stream[string] {
		return gostream.Of(v.DisplayName)
	}).ToSlice()

	return completionCandidates, cobra.ShellCompDirectiveKeepOrder
}

func makeCandidateVersionDefault(cmd *cobra.Command, args []string) {
	candidateName := args[0]
	var candidateVersion string
	if len(args) == 2 {
		candidateVersion = args[1]

		log.Logger.CanPrint(pterm.LogLevelInfo)
		{
			log.Logger.Debug(fmt.Sprintf("Setting version %s as default for candidate %s.", candidateVersion, candidateName))
		}

		selectedCandidate := validateCandidate(candidateName, candidateVersion)

		if selectedCandidate.IsPresent() {
			pterm.Info.WithWriter(os.Stderr).Printfln("Setting %s %s as the default version for all shells.", candidateName, candidateVersion)
			log.Logger.Debug("Selecteed candidate", log.Logger.Args("candidate", selectedCandidate))

			selectedCandidate.Get().MakeDefault()
			fmt.Fprintf(os.Stdout, "%s %s %s %s", types.PostprocessingFlag, "default", candidateName, candidateVersion)
		} else {
			pterm.Error.WithWriter(os.Stderr).Printfln("Selected version %s for candidate %s is not installed.", candidateVersion, candidateName)
			os.Exit(1)
		}
	}
}

func validateCandidate(candidateName string, candidateVersion string) *gostream.Optional[candidate.Candidate] {
	repositoryConfig := repository.GetRepository()
	if log.Logger.CanPrint(pterm.LogLevelTrace) {
		log.Logger.Trace("Repository config dump:")
		spew.Fdump(log.Logger.Writer, repositoryConfig)
	}
	_, candidateVersions, _ := repositoryConfig.ListCandidateVersions(candidateName)
	selectedCandidate := gostream.Of(candidateVersions...).Filter(func(t candidate.Candidate) bool {
		return t.DisplayName == candidateVersion
	}).Sorted(func(a, b candidate.Candidate) int {
		return b.Version.Compare(&a.Version)
	}).FindFirst()
	return selectedCandidate
}

func init() {
	rootCmd.AddCommand(defaultCmd)
}
