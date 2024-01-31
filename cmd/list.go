/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/YoshikiShibata/gostream"
	"github.com/begris-net/qtoolbox/internal/candidate"
	"github.com/begris-net/qtoolbox/internal/log"
	"github.com/begris-net/qtoolbox/internal/repository"
	"github.com/begris-net/qtoolbox/internal/ui"
	"github.com/begris-net/qtoolbox/internal/util"
	"github.com/davecgh/go-spew/spew"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list [candidate]",
	Short: "List all candidates or candidate versions",
	Long: `Invoke the subcommand without a candidate to see a comprehensive list of all
candidates with name, URL, detailed description and an installation command.
If the candidate qualifier is specified, the subcommand will display a list
of all available and local versions for that candidate. In addition, the
version list view marks all versions that are installed or currently in use. 
They appear as follows:

* - installed
> - currently in use

Java has a custom list view with vendor-specific details.`,
	Aliases:           []string{"ls"},
	Args:              cobra.MaximumNArgs(1),
	Run:               list,
	ValidArgsFunction: validCandidates,
}

var CandidateName string

func validCandidates(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return ValidCandidates(cmd, args, toComplete)
}

func list(cmd *cobra.Command, args []string) {
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

	if hasCandidate {
		candidateDescription, candidateVersions, hasMultipleProviderIds := repositoryConfig.ListCandidateVersions(CandidateName)

		if hasMultipleProviderIds {
			// TODO Display MultiProvider Table
			log.Logger.Debug("show multiprovider list")
			log.Logger.Debug(fmt.Sprintf("Number of candidates %d", len(candidateVersions)))
			displayName := func(d candidate.CandidateDescription) string {
				if d.DisplayName != nil && len(*d.DisplayName) > 0 {
					return util.SafeDeref(d.DisplayName)
				} else {
					return d.Name
				}
			}

			ui.NewViewItem(fmt.Sprintf("Available %s Versions", displayName(candidateDescription)),
				gostream.FlatMap(gostream.Of(candidateVersions...).Sorted(func(a, b candidate.Candidate) int {
					return b.Version.Compare(&a.Version)
				}), func(t candidate.Candidate) gostream.Stream[ui.ViewElement] {
					return gostream.Of(ui.ViewElement{
						Name:      t.DisplayName,
						Installed: t.Installed,
						Default:   t.Default,
					})
				}).ToSlice()).Show()
		} else {
			// Display simple Version Table
			log.Logger.Debug(fmt.Sprintf("Number of candidates %d", len(candidateVersions)))
			displayName := func(d candidate.CandidateDescription) string {
				if d.DisplayName != nil && len(*d.DisplayName) > 0 {
					return util.SafeDeref(d.DisplayName)
				} else {
					return d.Name
				}
			}

			ui.NewViewItem(fmt.Sprintf("Available %s Versions", displayName(candidateDescription)),
				gostream.FlatMap(gostream.Of(candidateVersions...).Sorted(func(a, b candidate.Candidate) int {
					return b.Version.Compare(&a.Version)
				}), func(t candidate.Candidate) gostream.Stream[ui.ViewElement] {
					return gostream.Of(ui.ViewElement{
						Name:      t.DisplayName,
						Installed: t.Installed,
						Default:   t.Default,
					})
				}).ToSlice()).Show()
		}
	} else {
		// show all candidate descriptions
		// TODO need some fancy UI stuff here - scrollable view
		for _, c := range repositoryConfig.ListCandidates() {
			c.Show()
		}
	}

}

func init() {
	rootCmd.AddCommand(listCmd)
}
