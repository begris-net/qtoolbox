package provider

import (
	"context"
	"fmt"
	"github.com/BooleanCat/go-functional/iter"
	"github.com/begris-net/qtoolbox/internal/cache"
	"github.com/begris-net/qtoolbox/internal/candidate"
	"github.com/begris-net/qtoolbox/internal/installer"
	"github.com/begris-net/qtoolbox/internal/log"
	"github.com/begris-net/qtoolbox/internal/provider/platform"
	"github.com/begris-net/qtoolbox/internal/types"
	"github.com/begris-net/qtoolbox/internal/util"
	"github.com/google/go-github/v57/github"
	"html/template"
	"net/url"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type GitHubTagsDownloadUrl struct {
	pageSize               int
	cacheTtl               time.Duration
	cachePath              string
	cache                  *cache.Cache[[]*github.RepositoryRelease]
	candidatesBathPath     string
	candidatesDownloadPath string
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
	d.candidatesBathPath = settings.CandidatesBathPath
	d.candidatesDownloadPath = settings.CandidatesDownloadPath

	if d.cache == nil {
		d.cache = &cache.Cache[[]*github.RepositoryRelease]{}
	}

	// Update cache settings
	d.cache.SetCachePath(&d.cachePath)
	d.cache.SetTTL(&d.cacheTtl)
}

func (d *GitHubTagsDownloadUrl) ListReleases(multipleProviders bool, provider candidate.CandidateProvider) []candidate.Candidate {
	releases := d.getCachedReleases(provider)
	log.Logger.Info(fmt.Sprintf("Fetched %d releases from provider %s.", len(releases), provider.ProviderRepoId))

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
			candidate.GetCandidateStatus()
			candidates = append(candidates, candidate)
		} else {
			log.Logger.Warn(fmt.Sprintf("Skipping invalid version %s releases from provider %s.", release.GetName(), provider.ProviderRepoId))
		}
	})
	return candidates
}

func (d *GitHubTagsDownloadUrl) Download(installCandidate candidate.Candidate) (*installer.CandidateDownload, error) {
	releases := d.getCachedReleases(installCandidate.Provider)
	log.Logger.Info(fmt.Sprintf("Fetched %d releases from provider %s.", len(releases), installCandidate.Provider.ProviderRepoId))

	os := runtime.GOOS
	arch := runtime.GOARCH

	log.Logger.Debug(fmt.Sprintf("Going to install version %s of candidate %s.", installCandidate.Version.Original(), installCandidate.Provider.Product))
	log.Logger.Trace("System detection for download determination", log.Logger.Args("os", runtime.GOOS, "arch", runtime.GOARCH, "GOROOT", runtime.GOROOT()))

	platformHandler := platform.NewPlatformHandler(installCandidate.Provider.Settings)
	properties := d.applyProviderMappings(platformHandler, templateProperties{
		OS:      os,
		Arch:    arch,
		Version: installCandidate.Version.Original(),
	})

	downloadUrl, err := d.renderUrlTemplate(installCandidate, properties)
	if err != nil {
		log.Logger.Fatal("Error during endpoint template creation.", log.Logger.Args("err", err))
	}
	log.Logger.Debug("Download", log.Logger.Args("url", downloadUrl))

	return &installer.CandidateDownload{
		Candidate:    installCandidate,
		DownloadUrl:  downloadUrl,
		DownloadPath: d.candidatesDownloadPath,
		InstallPath:  installCandidate.GetCandidateInstallationDir(),
		FileMode:     platformHandler.GetSetting(platform.FileMode),
	}, nil
}

func (d *GitHubTagsDownloadUrl) applyProviderMappings(ph *platform.PlatformHandler, properties templateProperties) templateProperties {
	properties.OS = ph.MapOS(properties.OS)
	properties.Arch = ph.MapArchitecture(properties.Arch)
	properties.OSArchiveExt = ph.MapExtension(properties.OS)
	return properties
}

func (d *GitHubTagsDownloadUrl) renderUrlTemplate(candidate candidate.Candidate, properties templateProperties) (*url.URL, error) {
	downloadUrlTemplate, err := template.New("endpoint").Parse(reflect.ValueOf(candidate.Provider.Settings[platform.Url_Template]).String())
	if err != nil {
		return nil, err
	}
	var urlBuilder strings.Builder
	err = downloadUrlTemplate.Execute(&urlBuilder, properties)
	if err != nil {
		return nil, err
	}
	url, err := url.Parse(urlBuilder.String())
	return url, err
}
