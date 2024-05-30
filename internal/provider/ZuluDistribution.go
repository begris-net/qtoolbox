package provider

import (
	"encoding/json"
	"fmt"
	"github.com/YoshikiShibata/gostream"
	"github.com/begris-net/qtoolbox/internal/cache"
	"github.com/begris-net/qtoolbox/internal/candidate"
	"github.com/begris-net/qtoolbox/internal/installer"
	"github.com/begris-net/qtoolbox/internal/log"
	"github.com/begris-net/qtoolbox/internal/provider/platform"
	"github.com/begris-net/qtoolbox/internal/types"
	"github.com/begris-net/qtoolbox/internal/util"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type ZuluDistribution struct {
	pageSize               int
	cacheTtl               time.Duration
	cachePath              string
	cache                  *cache.Cache[[]*ZuluRelease]
	candidatesBathPath     string
	candidatesDownloadPath string
}

func (d *ZuluDistribution) UpdateProviderSettings(settings types.ProviderSettings) {
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
		d.cache = &cache.Cache[[]*ZuluRelease]{}
	}

	// Update cache settings
	d.cache.SetCachePath(&d.cachePath)
	d.cache.SetTTL(&d.cacheTtl)
}

func (d *ZuluDistribution) endpointRender(endpoint string, properties templateProperties) (string, error) {
	endpointTemplate, err := template.New("endpoint").Parse(endpoint)
	if err != nil {
		return "", err
	}
	var endpointBuilder strings.Builder
	err = endpointTemplate.Execute(&endpointBuilder, properties)
	if err != nil {
		return "", err
	}
	return endpointBuilder.String(), err
}

func (d *ZuluDistribution) getRepositoryUrl(provider candidate.CandidateProvider) (*url.URL, error) {
	os := runtime.GOOS
	arch := runtime.GOARCH

	platformHandler := platform.NewPlatformHandler(provider.Settings)

	properties := d.applyProviderMappings(platformHandler, templateProperties{
		OS:   os,
		Arch: arch,
	})

	endpoint, err := d.endpointRender(provider.Endpoint, properties)
	if err != nil {
		log.Logger.Error(err.Error(), log.Logger.Args("endpoint-url-template", provider.Endpoint))
		return nil, err
	}
	endpointUrl, err := url.Parse(endpoint)
	if err != nil {
		log.Logger.Error(err.Error(), log.Logger.Args("endpoint-url", endpoint))
		return nil, err
	}

	return endpointUrl, nil
}

func (d *ZuluDistribution) applyProviderMappings(ph *platform.PlatformHandler, properties templateProperties) templateProperties {
	properties.OS = ph.MapOS(properties.OS)
	properties.Arch = ph.MapArchitecture(properties.Arch)
	properties.OSArchiveExt = ph.MapExtension(properties.OS)
	return properties
}

func (d *ZuluDistribution) getCachedReleases(provider candidate.CandidateProvider) []*ZuluRelease {
	refresh := func() []*ZuluRelease {
		repositoryUrl, err2 := d.getRepositoryUrl(provider)
		if err2 != nil {
			log.Logger.Fatal(fmt.Sprintf("Error on endpoint rendering for %s with provider %s.", provider.Endpoint,
				provider.Type))
		}

		log.Logger.Debug("Querying Azul Zulu repository", log.Logger.Args("rendered-endpoint-url", repositoryUrl.String()))
		resp, err := http.Get(repositoryUrl.String())
		if err != nil {
			panic(err)
		}

		if resp.StatusCode >= http.StatusBadRequest {
			log.Logger.Fatal(fmt.Sprintf("Error on lookup for %s with provider %s: %s\nurl: %s", provider.Endpoint,
				provider.Type, resp.Status, resp.Request.URL))
		}

		log.Logger.Debug("Response", log.Logger.Args("respomse", resp.Body))

		var zuluReleases []*ZuluRelease
		err = json.NewDecoder(resp.Body).Decode(&zuluReleases)

		if err != nil {
			panic(err)
		}
		return zuluReleases
	}

	return d.cache.GetCachedReleases(provider, refresh)
}

func (d *ZuluDistribution) ListReleases(multipleProviders bool, provider candidate.CandidateProvider) []candidate.Candidate {
	releases := d.getCachedReleases(provider)
	log.Logger.Info(fmt.Sprintf("Fetched %d releases from provider %s.", len(releases), provider.ProviderRepoId))

	return gostream.FlatMap(gostream.Of(releases...), func(release *ZuluRelease) gostream.Stream[candidate.Candidate] {
		candidateVersion, err := parseVersion(provider.VersionCleanupRegex, d.versionConverter(release.JavaVersion))
		if err == nil {
			candidate := candidate.Candidate{
				Version:  util.SafeDeref(candidateVersion),
				Provider: provider,
			}
			candidate.DisplayName = renderDisplayName(multipleProviders, candidate)
			candidate.GetCandidateStatus()
			return gostream.Of(candidate)
		} else {
			log.Logger.Warn(fmt.Sprintf("Skipping invalid version %s releases from provider %s.", release.Name, provider.ProviderRepoId))
			return gostream.Empty[candidate.Candidate]()
		}
	}).ToSlice()
}

func (d *ZuluDistribution) versionConverter(versionParts []int) string {
	return strings.Join(gostream.FlatMap(gostream.Of(versionParts...), func(part int) gostream.Stream[string] {
		return gostream.Of(strconv.Itoa(part))
	}).ToSlice(), ".")
}

func (d *ZuluDistribution) Download(installCandidate candidate.Candidate) (*installer.CandidateDownload, error) {
	releases := d.getCachedReleases(installCandidate.Provider)
	log.Logger.Info(fmt.Sprintf("Fetched %d releases from provider %s.", len(releases), installCandidate.Provider.ProviderRepoId))

	log.Logger.Debug("Trying to install", log.Logger.Args("candidate", installCandidate.Version.Original()))

	release := gostream.Of(releases...).Filter(func(v *ZuluRelease) bool {
		return d.versionConverter(v.JavaVersion) == installCandidate.Version.Original()
	}).FindFirst().OrElsePanic()

	platformHandler := platform.NewPlatformHandler(installCandidate.Provider.Settings)

	log.Logger.Debug("Oi", log.Logger.Args("candidate-release", release))

	downloadUrl, err := url.Parse(release.DownloadUrl)
	candidateDownload := &installer.CandidateDownload{
		Candidate:    installCandidate,
		DownloadUrl:  downloadUrl,
		DownloadPath: d.candidatesDownloadPath,
		InstallPath:  installCandidate.GetCandidateInstallationDir(),
		FileMode:     platformHandler.GetSetting(platform.FileMode),
	}

	log.Logger.Debug("Installing", log.Logger.Args("candidate", candidateDownload))

	return candidateDownload, err
}

type ZuluRelease struct {
	AvailabilityType   string `json:"availability_type"`
	DistroVersion      []int  `json:"distro_version"`
	DownloadUrl        string `json:"download_url"`
	JavaVersion        []int  `json:"java_version"`
	Latest             bool   `json:"latest"`
	Name               string `json:"name"`
	OpenjdkBuildNumber int    `json:"openjdk_build_number"`
	PackageUuid        string `json:"package_uuid"`
	Product            string `json:"product"`
}
