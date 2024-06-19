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

package candidate

import (
	"fmt"
	"github.com/begris-net/qtoolbox/internal/types"
	"github.com/begris-net/qtoolbox/internal/util"
	"github.com/hashicorp/go-version"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type Candidate struct {
	Provider    CandidateProvider
	Version     version.Version
	DisplayName string
	Installed   bool
	Default     bool
	ExportPath  *string
}

type CandidateProvider struct {
	ProviderRepoId       string
	Product              string
	Id                   string
	Vendor               string
	Type                 types.ProviderType
	Endpoint             string
	PreRelease           bool
	VersionCleanupRegex  *regexp.Regexp
	Settings             map[string]any
	InstallationBasePath string
}

type CandidateDescription struct {
	Name              string
	DisplayName       *string
	Description       *string
	DefaultProviderId *string
}

func (c CandidateProvider) GetCandidateInstallationBasePath() string {
	return path.Join(c.InstallationBasePath, c.Product)
}

func (c Candidate) CandidateId() {

}

func (c Candidate) GetCandidateInstallationDir() string {
	return path.Join(c.Provider.GetCandidateInstallationBasePath(), c.DisplayName)
}

func (c Candidate) GetCurrentCandidate() string {
	return path.Join(c.Provider.GetCandidateInstallationBasePath(), "current")
}

func (c Candidate) MakeDefault() error {
	_, err := os.Stat(c.GetCurrentCandidate())
	if err == nil {
		os.Remove(c.GetCurrentCandidate())
	}
	return os.Symlink(c.GetCandidateInstallationDir(), c.GetCurrentCandidate())
}

func (c *Candidate) GetCandidateStatus() {
	stat, err := os.Stat(c.GetCandidateInstallationDir())
	if err == nil && stat.IsDir() {
		c.Installed = true
		currentLink, err2 := os.Readlink(c.GetCurrentCandidate())
		if err2 == nil {
			c.Default = path.Base(filepath.ToSlash(currentLink)) == stat.Name()
		}
	}
}

func (c Candidate) Show() {
	println(util.OrElse(c.DisplayName, c.Version.Original()))
}

func (c CandidateDescription) Show() {
	fmt.Printf("%s\n", strings.Repeat("-", 80))
	fmt.Printf("%s", util.SafeDeref(c.Description))
	cmd := fmt.Sprintf("$ toolbox install %s", c.Name)
	fmt.Printf("\n%s%s\n", strings.Repeat(" ", 80-len(cmd)), cmd)
}
