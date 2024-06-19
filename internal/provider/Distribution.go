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

package provider

import (
	"fmt"
	"github.com/begris-net/qtoolbox/internal/candidate"
	"github.com/begris-net/qtoolbox/internal/installer"
	"github.com/begris-net/qtoolbox/internal/types"
	"github.com/begris-net/qtoolbox/internal/util"
	"log"
)

type Distribution interface {
	ListReleases(multipleProviders bool, provider candidate.CandidateProvider) []candidate.Candidate
	Download(installCandidate candidate.Candidate) (*installer.CandidateDownload, error)
	UpdateProviderSettings(settings types.ProviderSettings)
}

type DummyDistributor struct{}

func renderDisplayName(multipleProviders bool, candidate candidate.Candidate) string {
	if multipleProviders {
		return fmt.Sprintf("%s-%s", candidate.Version.Original(), candidate.Provider.Id)
	} else {
		return candidate.Version.Original()
	}
}

func (d DummyDistributor) UpdateProviderSettings(settings types.ProviderSettings) {
}

func (d DummyDistributor) ListReleases(multipleProviders bool, provider candidate.CandidateProvider) []candidate.Candidate {
	log.Printf("No provider available for provider %s with type %s.",
		util.OrElse(provider.Vendor, provider.Id), provider.Type)
	return make([]candidate.Candidate, 0)
}

func (d DummyDistributor) Download(installCandidate candidate.Candidate) (*installer.CandidateDownload, error) {
	log.Printf("No provider available for provider %s with type %s.",
		util.OrElse(installCandidate.Provider.Vendor, installCandidate.Provider.Id), installCandidate.Provider.Type)
	return nil, nil
}

var dummy = &DummyDistributor{}

var providerMap = map[types.ProviderType]Distribution{
	types.GitHubRelease:         &GithubDistribution{},
	types.GitHubTagsDownloadUrl: &GitHubTagsDownloadUrl{},
	types.MavenRelease:          &MavenRelease{},
	types.Zulu:                  &ZuluDistribution{},
}

func Distributor(provider types.ProviderType) Distribution {
	distribution := providerMap[provider]

	if distribution == nil {
		return dummy
	}
	return distribution
}

type templateProperties struct {
	OS, Arch, Version, OSArchiveExt string
}
