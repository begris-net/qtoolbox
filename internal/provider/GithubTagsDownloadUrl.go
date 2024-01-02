package provider

import (
	"context"
	"github.com/BooleanCat/go-functional/iter"
	"github.com/begris-net/qtoolbox/internal/cache"
	"github.com/begris-net/qtoolbox/internal/candidate"
	"github.com/begris-net/qtoolbox/internal/types"
	"github.com/begris-net/qtoolbox/internal/util"
	"github.com/google/go-github/v57/github"
	"log"
	"strconv"
	"strings"
	"time"
)

type GitHubTagsDownloadUrl struct {
	pageSize  int
	cacheTtl  time.Duration
	cachePath string
	cache     *cache.Cache[[]*github.RepositoryRelease]
}

func (d *GitHubTagsDownloadUrl) getCachedReleases(provider candidate.CandidateProvider) []*github.RepositoryRelease {
	refresh := func() []*github.RepositoryRelease {
		repo := strings.Split(provider.Endpoint, "/")
		client := github.NewClient(nil)
		releases, _, err := client.Repositories.ListReleases(context.Background(),
			repo[0], repo[1], &github.ListOptions{PerPage: d.pageSize})

		if err != nil {
			panic(err)
		}
		return releases
	}

	return d.cache.GetCachedReleases(provider, refresh)
}

func (d *GitHubTagsDownloadUrl) UpdateProviderSettings(settings types.ProviderSettings) {
	pageSize, err := strconv.Atoi(settings.Setting["page-size"])
	if err == nil {
		d.pageSize = pageSize
	} else {
		d.pageSize = 100
	}
	ttl, err := time.ParseDuration(settings.Setting["version-cache-ttl"])
	if err == nil {
		d.cacheTtl = ttl
	} else {
		d.cacheTtl = 1 * time.Hour
	}
	d.cachePath = settings.CachePath

	if d.cache == nil {
		d.cache = &cache.Cache[[]*github.RepositoryRelease]{}
	}

	// Update cache settings
	d.cache.SetCachePath(&d.cachePath)
	d.cache.SetTTL(&d.cacheTtl)
}

func (d *GitHubTagsDownloadUrl) ListReleases(multipleProviders bool, provider candidate.CandidateProvider) []candidate.Candidate {
	releases := d.getCachedReleases(provider)
	log.Printf("Fetched %d releases from provider %s.", len(releases), provider.ProviderRepoId)

	var candidates []candidate.Candidate
	iter.Lift(releases).Filter(func(v *github.RepositoryRelease) bool {
		return !*v.Prerelease || (*v.Prerelease == provider.PreRelease)
	}).ForEach(func(release *github.RepositoryRelease) {
		candidateVersion, err := parseVersion(provider.VersionCleanupRegex, release.GetName())
		if err == nil {
			candidate := candidate.Candidate{
				Version:  util.SafeDeref(candidateVersion),
				Provider: provider,
			}
			candidate.DisplayName = renderDisplayName(multipleProviders, candidate)
			candidates = append(candidates, candidate)
		} else {
			log.Printf("Skipping invalid version %s releases from provider %s.", release.GetName(), provider.ProviderRepoId)
		}
	})
	return candidates
}

func (d *GitHubTagsDownloadUrl) Download(candidate candidate.Candidate) error {
	//TODO implement me
	panic("implement me")
}
