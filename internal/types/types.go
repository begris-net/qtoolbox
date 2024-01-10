package types

type ProviderType string

const (
	Zulu                  ProviderType = "Zulu"
	GitHubRelease         ProviderType = "GitHubRelease"
	GitHubTagsDownloadUrl ProviderType = "GitHubTagsDownloadUrl"
	Download              ProviderType = "Download"
	MavenRelease          ProviderType = "MavenRelease"
)

type ProviderSettings struct {
	CachePath              string
	CandidatesBathPath     string
	CandidatesDownloadPath string
	Setting                map[string]string `yaml:"*,flow"`
}

//func MapProviderTypeByName(providerTypeName string) ProviderType {
//
//}
