package provider

import (
	"encoding/xml"
	"fmt"
	"github.com/BooleanCat/go-functional/iter"
	"github.com/begris-net/qtoolbox/internal/cache"
	"github.com/begris-net/qtoolbox/internal/candidate"
	"github.com/begris-net/qtoolbox/internal/installer"
	"github.com/begris-net/qtoolbox/internal/log"
	"github.com/begris-net/qtoolbox/internal/provider/platform"
	"github.com/begris-net/qtoolbox/internal/types"
	"github.com/begris-net/qtoolbox/internal/util"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	fallbackMavenRepoUrl = "https://repo1.maven.org/maven2"
	mavenMetadata        = "maven-metadata.xml"
)

type MavenRelease struct {
	baseUrl                *url.URL
	cacheTtl               time.Duration
	cachePath              string
	cache                  *cache.Cache[*Metadata]
	candidatesBathPath     string
	candidatesDownloadPath string
}

func (d *MavenRelease) UpdateProviderSettings(settings types.ProviderSettings) {
	bUrl, err := url.Parse(settings.Setting["base-url"])
	if err == nil {
		d.baseUrl = bUrl
	} else {
		parse, err := url.Parse(fallbackMavenRepoUrl)
		if err != nil {
			panic(err)
		}
		d.baseUrl = parse
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
		d.cache = &cache.Cache[*Metadata]{}
	}

	// Update cache settings
	d.cache.SetCachePath(&d.cachePath)
	d.cache.SetTTL(&d.cacheTtl)
}

func (d *MavenRelease) getArtifactBaseUrl(provider candidate.CandidateProvider) *url.URL {
	gav := strings.Split(provider.Endpoint, ":")
	gav = append(strings.Split(gav[0], "."), gav[1], mavenMetadata)
	return d.baseUrl.JoinPath(gav...)
}

func (d *MavenRelease) ListReleases(multipleProviders bool, provider candidate.CandidateProvider) []candidate.Candidate {
	refresh := func() *Metadata {
		resp, err := http.Get(d.getArtifactBaseUrl(provider).String())
		if err != nil {
			panic(err)
		}

		if resp.StatusCode >= http.StatusBadRequest {
			log.Logger.Fatal(fmt.Sprintf("Error on lookup for %s with provider %s: %s\nurl: %s", provider.Endpoint,
				provider.Type, resp.Status, resp.Request.URL))
		}
		var metadata Metadata
		err = xml.NewDecoder(resp.Body).Decode(&metadata)

		if err != nil {
			panic(err)
		}
		return &metadata
	}

	releases := d.cache.GetCachedReleases(provider, refresh).Versioning.Versions.Version

	var candidates []candidate.Candidate
	iter.Lift(releases).Filter(func(v string) bool {
		return provider.PreRelease || !strings.Contains(v, "-")
	}).
		ForEach(func(release string) {
			candidateVersion, _ := parseVersion(nil, release)
			candidate := candidate.Candidate{
				Version:  util.SafeDeref(candidateVersion),
				Provider: provider,
			}
			candidate.DisplayName = renderDisplayName(multipleProviders, candidate)
			candidates = append(candidates, candidate)
		})

	return candidates
}

func (d *MavenRelease) Download(installCandidate candidate.Candidate) (*installer.CandidateDownload, error) {
	//releases := d.getCachedReleases(installCandidate.Provider)
	//log.Logger.Info(fmt.Sprintf("Fetched %d releases from provider %s.", len(releases), installCandidate.Provider.ProviderRepoId))

	//os := runtime.GOOS
	//arch := runtime.GOARCH

	//log.Logger.Debug(fmt.Sprintf("Going to install version %s of candidate %s.", installCandidate.Version.Original(), installCandidate.Provider.Product))
	//log.Logger.Trace("System detection for download determination", log.Logger.Args("os", runtime.GOOS, "arch", runtime.GOARCH, "GOROOT", runtime.GOROOT()))
	//
	//platformHandler := platform.NewPlatformHandler(installCandidate.Provider.Settings)
	//properties := d.applyProviderMappings(platformHandler, templateProperties{
	//	OS:      os,
	//	Arch:    arch,
	//	Version: installCandidate.Version.Original(),
	//})

	platformHandler := platform.NewPlatformHandler(installCandidate.Provider.Settings)

	return &installer.CandidateDownload{
		Candidate:    installCandidate,
		DownloadUrl:  nil,
		DownloadPath: d.candidatesDownloadPath,
		InstallPath:  installCandidate.GetCandidateInstallationDir(d.candidatesBathPath),
		FileMode:     platformHandler.GetSetting(platform.FileMode),
	}, nil
}

type Metadata struct {
	XMLName    xml.Name `xml:"metadata"`
	GroupId    string   `xml:"groupId"`
	ArtifactId string   `xml:"artifactId"`
	Versioning struct {
		Latest   string `xml:"latest"`
		Release  string `xml:"release"`
		Versions struct {
			Version []string `xml:"version"`
		} `xml:"versions"`
		LastUpdated string `xml:"lastUpdated"`
	} `xml:"versioning"`
}
