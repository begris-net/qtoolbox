package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/BooleanCat/go-functional/iter"
	"github.com/YoshikiShibata/gostream"
	"github.com/begris-net/qtoolbox/internal/cache"
	"github.com/begris-net/qtoolbox/internal/candidate"
	"github.com/begris-net/qtoolbox/internal/log"
	"github.com/begris-net/qtoolbox/internal/types"
	"github.com/begris-net/qtoolbox/internal/util"
	"github.com/google/go-github/v57/github"
	"strconv"
	"strings"
	"time"
)

type GithubDistribution struct {
	pageSize  int
	cacheTtl  time.Duration
	cachePath string
	cache     *cache.Cache[[]*github.RepositoryRelease]
}

func (d *GithubDistribution) getCachedReleases(provider candidate.CandidateProvider) []*github.RepositoryRelease {
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

func (d *GithubDistribution) UpdateProviderSettings(settings types.ProviderSettings) {
	pageSize, err := strconv.Atoi(settings.Setting["page-size"])
	if err == nil {
		d.pageSize = pageSize
	} else {
		d.pageSize = 100
	}
	ttl, err := time.ParseDuration(settings.Setting["version-cache-ttl"])
	if err == nil {
		d.cacheTtl = ttl
		log.Logger.Debug(fmt.Sprintf("Cache TTL for %T is %v.", d, d.cacheTtl))
	} else {
		d.cacheTtl = 1 * time.Hour
		log.Logger.Debug(fmt.Sprintf("Cache TTL for %T is %v.", d, d.cacheTtl))
	}
	d.cachePath = settings.CachePath

	if d.cache == nil {
		d.cache = &cache.Cache[[]*github.RepositoryRelease]{}
	}

	// Update cache settings
	d.cache.SetCachePath(&d.cachePath)
	d.cache.SetTTL(&d.cacheTtl)
}

func (d *GithubDistribution) ListReleases(multipleProviders bool, provider candidate.CandidateProvider) []candidate.Candidate {
	releases := d.getCachedReleases(provider)
	log.Logger.Info(fmt.Sprintf("Fetched %d releases from provider %s.", len(releases), provider.ProviderRepoId))

	var candidates []candidate.Candidate
	iter.Lift(releases).Filter(func(v *github.RepositoryRelease) bool {
		return !(*v.Prerelease && provider.PreRelease)
	}).
		ForEach(func(release *github.RepositoryRelease) {
			candidateVersion, err := parseVersion(provider.VersionCleanupRegex, release.GetName())
			if err == nil {
				candidate := candidate.Candidate{
					Version:  util.SafeDeref(candidateVersion),
					Provider: provider,
				}
				candidate.DisplayName = renderDisplayName(multipleProviders, candidate)
				candidates = append(candidates, candidate)
			} else {
				log.Logger.Warn(fmt.Sprintf("Skipping invalid version %s releases from provider %s.", release.GetName(), provider.ProviderRepoId))
			}
		})
	return candidates
}

func (d *GithubDistribution) Download(candidate candidate.Candidate) error {
	releases := d.getCachedReleases(candidate.Provider)
	log.Logger.Info(fmt.Sprintf("Fetched %d releases from provider %s.", len(releases), candidate.Provider.ProviderRepoId))

	gostream.Of(releases...).ForEach(func(t *github.RepositoryRelease) {
		//github.RepositoryRelease.GetAssetsURL()
	})

	return errors.New("not yet implemented")
}
