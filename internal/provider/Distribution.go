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
