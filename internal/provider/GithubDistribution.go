package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/BooleanCat/go-functional/iter"
	"github.com/YoshikiShibata/gostream"
	"github.com/begris-net/qtoolbox/internal/cache"
	"github.com/begris-net/qtoolbox/internal/candidate"
	"github.com/begris-net/qtoolbox/internal/installer"
	"github.com/begris-net/qtoolbox/internal/log"
	"github.com/begris-net/qtoolbox/internal/provider/platform"
	"github.com/begris-net/qtoolbox/internal/types"
	"github.com/begris-net/qtoolbox/internal/util"
	"github.com/google/go-github/v57/github"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type GithubDistribution struct {
	pageSize               int
	cacheTtl               time.Duration
	cachePath              string
	cache                  *cache.Cache[[]*github.RepositoryRelease]
	candidatesBathPath     string
	candidatesDownloadPath string
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
	d.candidatesBathPath = settings.CandidatesBathPath
	d.candidatesDownloadPath = settings.CandidatesDownloadPath

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
				candidate.GetCandidateStatus()
				candidates = append(candidates, candidate)
			} else {
				log.Logger.Warn(fmt.Sprintf("Skipping invalid version %s releases from provider %s.", release.GetName(), provider.ProviderRepoId))
			}
		})
	return candidates
}

func (d *GithubDistribution) Download(installCandidate candidate.Candidate) (*installer.CandidateDownload, error) {
	releases := d.getCachedReleases(installCandidate.Provider)
	log.Logger.Info(fmt.Sprintf("Fetched %d releases from provider %s.", len(releases), installCandidate.Provider.ProviderRepoId))

	os := runtime.GOOS
	arch := runtime.GOARCH

	platformHandler := platform.NewPlatformHandler(installCandidate.Provider.Settings)

	log.Logger.Debug(fmt.Sprintf("Going to install version %s of candidate %s.", installCandidate.Version.Original(), installCandidate.Provider.Product))
	log.Logger.Trace("System detection for download determination", log.Logger.Args("os", runtime.GOOS, "arch", runtime.GOARCH, "GOROOT", runtime.GOROOT()))

	archRegex, err := platformHandler.GetArchitectureRegex(arch)
	if err != nil {
		log.Logger.Error(err.Error(), log.Logger.Args("arch", arch))
		return nil, err
	}

	extRegex, err := platformHandler.GetExtensionRegex(os)
	if err != nil {
		log.Logger.Error(err.Error(), log.Logger.Args("os", os))
		return nil, err
	}

	release := gostream.Of(releases...).Filter(func(t *github.RepositoryRelease) bool {
		return util.SafeDeref(t.Name) == installCandidate.Version.Original()
	}).FindFirst().OrElsePanic()

	releaseAssets := gostream.Of(release.Assets...).Filter(func(t *github.ReleaseAsset) bool {
		assetName := strings.ToLower(t.GetName())
		log.Logger.Trace(fmt.Sprintf("%s --> %s", util.SafeDeref(t.Name), util.SafeDeref(t.BrowserDownloadURL)))
		return strings.Contains(assetName, platformHandler.MapOS(os)) && archRegex.MatchString(assetName) && extRegex.MatchString(assetName)
	}).ToSlice()

	if len(releaseAssets) > 1 {
		assets := make(map[string]any, len(releaseAssets))
		gostream.Of(releaseAssets...).ForEach(func(t *github.ReleaseAsset) {
			assets[t.GetName()] = util.SafeDeref(t.BrowserDownloadURL)
		})
		log.Logger.Warn("More than one download asset matched matched the criteria. First match is used for download.",
			log.Logger.Args("criteria os", platformHandler.MapOS(os), "criteria arch", arch, "criteria arch-regex", archRegex, "criteria extension-regex", extRegex),
			log.Logger.ArgsFromMap(assets))
	}

	log.Logger.Trace("System detection for download determination", log.Logger.Args("os", runtime.GOOS, "arch", runtime.GOARCH, "GOROOT", runtime.GOROOT()))
	var candidateDownload *installer.CandidateDownload
	var noAssetErr error
	gostream.Of(releaseAssets...).FindFirst().IfPresentOrElse(func(t *github.ReleaseAsset) {
		downloadUrl, _ := url.Parse(gostream.Of(releaseAssets...).FindFirst().Get().GetBrowserDownloadURL())
		candidateDownload = &installer.CandidateDownload{
			Candidate:    installCandidate,
			DownloadUrl:  downloadUrl,
			DownloadPath: d.candidatesDownloadPath,
			InstallPath:  installCandidate.GetCandidateInstallationDir(),
			FileMode:     platformHandler.GetSetting(platform.FileMode),
		}
	}, func() {
		noAssetErr = errors.New(fmt.Sprintf("No download asset for candidate %s with version %s found.", installCandidate.Provider.Product, installCandidate.DisplayName))
	})

	return candidateDownload, noAssetErr
}
