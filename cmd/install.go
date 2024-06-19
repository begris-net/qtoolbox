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
	"github.com/begris-net/qtoolbox/internal/config"
	"github.com/begris-net/qtoolbox/internal/log"
	"github.com/begris-net/qtoolbox/internal/provider"
	"github.com/begris-net/qtoolbox/internal/repository"
	"github.com/begris-net/qtoolbox/internal/types"
	"github.com/begris-net/qtoolbox/internal/util"
	"github.com/davecgh/go-spew/spew"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install candidate [version]",
	Short: "install a candidate version",
	Long: `TBD - ....Invoke the subcommand without a candidate to see a comprehensive list of all
candidates with name, URL, detailed description and an installation command.
If the candidate qualifier is specified, the subcommand will display a list
of all available and local versions for that candidate. In addition, the
version list view marks all versions that are installed or currently in use. 
They appear as follows:

* - installed
> - currently in use

Java has a custom list view with vendor-specific details.`,
	Aliases:           []string{"i"},
	Args:              cobra.RangeArgs(1, 2),
	Run:               installCandidate,
	ValidArgsFunction: validInstallCandidateVersions,
}

func validInstallCandidateVersions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cmd.Flags().Parse(args)
	cleanedArgs := cmd.Flags().Args()

	if len(cleanedArgs) >= 2 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	if len(cleanedArgs) >= 1 {
		return validCandidateVersions(cleanedArgs[0], toComplete)
	} else {
		return ValidCandidates(cmd, cleanedArgs, toComplete)
	}
}

func installCandidate(cmd *cobra.Command, args []string) {
	candidateName := args[0]
	latestVersion := true
	var candidateVersion string
	if len(args) == 2 {
		candidateVersion = args[1]
		latestVersion = false
	}

	log.Logger.CanPrint(pterm.LogLevelInfo)
	{
		if !latestVersion {
			log.Logger.Debug(fmt.Sprintf("Installing candidate %s with version %s.", candidateName, candidateVersion))
		} else {
			log.Logger.Debug(fmt.Sprintf("Installing candidate %s with latest version.", candidateName))
		}
	}

	repositoryConfig := repository.GetRepository()
	if log.Logger.CanPrint(pterm.LogLevelTrace) {
		log.Logger.Trace("Repository config dump:")
		spew.Fdump(log.Logger.Writer, repositoryConfig)
	}

	candidateInfo, candidateVersions, hasMultipleProviders := repositoryConfig.ListCandidateVersions(candidateName)

	selectedCandidate := gostream.Of(candidateVersions...).Filter(func(t candidate.Candidate) bool {
		return (latestVersion && (!hasMultipleProviders || (util.SafeDeref(candidateInfo.DefaultProviderId) == t.Provider.Id))) ||
			t.DisplayName == candidateVersion
	}).Sorted(func(a, b candidate.Candidate) int {
		return b.Version.Compare(&a.Version)
	}).FindFirst().OrElsePanic()

	log.Logger.Debug("Selected installation candidate.", log.Logger.Args("candidate", selectedCandidate))

	download, err := provider.Distributor(selectedCandidate.Provider.Type).Download(selectedCandidate)
	if err != nil {
		log.Logger.Fatal(err.Error())
	}

	currentConfig, err := config.GetCurrentConfig()
	if err != nil {
		log.Logger.Fatal("Error get current configuration.", log.Logger.Args("err", err))
	}

	_, checkedDownloadError := download.CheckedDownload(currentConfig.GetCandidateCachePath())
	if checkedDownloadError != nil {
		log.Logger.Error("Error during candidate installation.", log.Logger.Args("err", checkedDownloadError))
		os.Exit(1)
	}

	if !gostream.Of(candidateVersions...).AnyMatch(func(t candidate.Candidate) bool {
		return t.Default
	}) {
		log.Logger.Info("Setting default")
		selectedCandidate.MakeDefault()
	} else {
		if !selectedCandidate.Default {
			pterm.SetDefaultOutput(os.Stderr)
			interactiveConfirm := pterm.DefaultInteractiveConfirm
			result, _ := interactiveConfirm.WithDefaultValue(true).Show(fmt.Sprintf("Do you want %s %s to be set as default?", selectedCandidate.Provider.Product, selectedCandidate.Version.Original()))
			if result {
				selectedCandidate.MakeDefault()
			}
		}
	}
	fmt.Fprintf(os.Stdout, "%s %s %s %s", types.PostprocessingFlag, "install", selectedCandidate.Provider.Product, selectedCandidate.Version.String())
}

func validCandidateVersions(candidateName string, toComplete string) ([]string, cobra.ShellCompDirective) {
	repositoryConfig := repository.GetRepository()
	if log.Logger.CanPrint(pterm.LogLevelTrace) {
		log.Logger.Trace("Repository config dump:")
		spew.Fdump(log.Logger.Writer, repositoryConfig)
	}

	_, candidateVersions, _ := repositoryConfig.ListCandidateVersions(candidateName)

	completionCandidates := gostream.FlatMap(gostream.Of(candidateVersions...).Filter(func(v candidate.Candidate) bool {
		return strings.HasPrefix(v.DisplayName, toComplete)
	}).Sorted(func(a, b candidate.Candidate) int {
		return b.Version.Compare(&a.Version)
	}), func(v candidate.Candidate) gostream.Stream[string] {
		return gostream.Of(v.DisplayName)
	}).ToSlice()

	return completionCandidates, cobra.ShellCompDirectiveKeepOrder
}

func init() {
	rootCmd.AddCommand(installCmd)
}
