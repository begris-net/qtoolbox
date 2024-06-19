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
