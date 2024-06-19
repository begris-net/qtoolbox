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

package repository

import (
	"fmt"
	"github.com/YoshikiShibata/gostream"
	"github.com/begris-net/qtoolbox/internal/candidate"
	"github.com/begris-net/qtoolbox/internal/config"
	"github.com/begris-net/qtoolbox/internal/log"
	candidateProvider "github.com/begris-net/qtoolbox/internal/provider"
	"github.com/mariomac/gostream/stream"
	"github.com/pterm/pterm"
	"regexp"
)

var repository *Repository
var candidateInstallationBasePath string

func initRepository(metadataPath string) *Repository {
	return LoadRepositoryConfig(metadataPath)
}

func GetRepository() *Repository {
	if repository == nil {
		currentConfig, err := config.GetCurrentConfig()
		if err != nil {
			panic(err)
		}
		repository = initRepository(currentConfig.GetRepositoryConfigPath())
		candidateInstallationBasePath = currentConfig.GetCandidatesBathPath()
	}
	return repository
}

func (repository *Repository) ListCandidates() []candidate.CandidateDescription {

	mapCandidateInfo := func(t CandidateInfo) candidate.CandidateDescription {
		return candidate.CandidateDescription{
			Name:        t.Name,
			DisplayName: &t.DisplayName,
			Description: &t.Description,
		}
	}

	candidates := []candidate.CandidateDescription{}
	for _, candidate := range repository.Candidates {
		candidates = append(candidates, mapCandidateInfo(candidate))
	}

	return candidates
}

func (repository *Repository) FetchCandidateProvider(candidateName string) (
	candidate.CandidateDescription,
	[]candidate.CandidateProvider,
	// has multiple provider Ids (vendors)
	bool) {

	candidateInfo := repository.FindCandidate(candidateName)

	candidateDescription := candidate.CandidateDescription{
		Name:              candidateInfo.Name,
		DisplayName:       &candidateInfo.DisplayName,
		Description:       &candidateInfo.Description,
		DefaultProviderId: &candidateInfo.DefaultProviderId,
	}

	mapCandidate := func(repoId string, t ProviderInfo) candidate.CandidateProvider {
		return candidate.CandidateProvider{
			ProviderRepoId:       repoId,
			Product:              candidateName,
			Id:                   t.ID,
			Vendor:               t.Vendor,
			Type:                 t.Type,
			Endpoint:             t.Endpoint,
			PreRelease:           t.PreReleases,
			VersionCleanupRegex:  regexp.MustCompile(t.VersionCleanup),
			Settings:             t.Settings,
			InstallationBasePath: candidateInstallationBasePath,
		}
	}

	candidateProvider := []candidate.CandidateProvider{}
	for repoId, provider := range candidateInfo.Provider {
		candidateProvider = append(candidateProvider, mapCandidate(repoId, provider))
	}

	candidateProviders := gostream.Of(candidateProvider...)
	uniqueProviderIds := gostream.Distinct(gostream.FlatMap(candidateProviders,
		func(provider candidate.CandidateProvider) gostream.Stream[string] {
			return gostream.Of(provider.Id)
		}))
	hasMultipleProviderIds := uniqueProviderIds.Count() > 1

	return candidateDescription, candidateProvider, hasMultipleProviderIds
}

func (repository *Repository) ListCandidateVersions(candidateName string) (
	candidate.CandidateDescription,
	[]candidate.Candidate,
	// has multiple provider Ids (vendors)
	bool) {
	cadidateDescription, candidateProviders, hasMultipleProviderIds := repository.FetchCandidateProvider(candidateName)

	if log.Logger.CanPrint(pterm.LogLevelDebug) {
		gostream.Of(candidateProviders...).ForEach(func(t candidate.CandidateProvider) {
			log.Logger.Debug(fmt.Sprintf("Found candidate provider %s.", t.ProviderRepoId))
		})
	}

	fetchReleases := func(provider candidate.CandidateProvider) gostream.Stream[candidate.Candidate] {
		return gostream.Of(candidateProvider.Distributor(provider.Type).
			ListReleases(hasMultipleProviderIds, provider)...)
	}

	candidates := gostream.FlatMap(gostream.Of(candidateProviders...), fetchReleases)
	return cadidateDescription, candidates.ToSlice(), hasMultipleProviderIds
}

func (repository *Repository) FindCandidate(candidateName string) CandidateInfo {
	slice := stream.OfSlice(repository.Candidates)
	first, _ := slice.Filter(func(v CandidateInfo) bool {
		return v.Name == candidateName
	}).FindFirst()

	return first
}
