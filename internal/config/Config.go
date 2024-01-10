package config

import (
	"errors"
	"github.com/begris-net/qtoolbox/internal/provider"
	"github.com/begris-net/qtoolbox/internal/types"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v3"
	"os"
	"path"
	"path/filepath"
)

const (
	QToolboxDirectory  string = ".qtoolbox"
	ConfigDir          string = QToolboxDirectory + "/config"
	CandidatesDir      string = "/candidates"
	HooksDir           string = QToolboxDirectory + "/hooks"
	VarDir             string = "var"
	CacheDir           string = VarDir + "/cache"
	RepositoryCacheDir string = CacheDir + "/repository"
	CandidateCacheDir  string = "tmp"
)

type Config struct {
	ConfigFile         string                                   `yaml:",omitempty"`
	RepositoryMetadata string                                   `yaml:"repository-metadata"`
	OS                 string                                   `yaml:"os,omitempty"`
	Platform           string                                   `yaml:"platform,omitempty"`
	ProviderSettings   map[types.ProviderType]map[string]string `yaml:"provider-settings"`
	basePath           string
	RepositoryCacheDir string
}

var currentConfig *Config

func GetCurrentConfig() (*Config, error) {
	if currentConfig != nil {
		return currentConfig, nil
	}
	return nil, errors.New("No configuration available.")
}

func LoadConfig(cfgFile string) *Config {

	var basepath string
	if len(cfgFile) <= 0 {

		homeDir, errHomeDir := homedir.Dir()
		if errHomeDir != nil {
			panic(errHomeDir)
		}

		cfgFile = filepath.Join(homeDir, ConfigDir, "config.yaml")
		basepath = filepath.Join(homeDir, QToolboxDirectory)
	}

	data, err := os.ReadFile(cfgFile)
	if err != nil {
		panic(err)
	}

	var config Config

	if err := yaml.Unmarshal(data, &config); err != nil {
		panic(err)
	}
	config.ConfigFile = cfgFile
	config.basePath = basepath
	currentConfig = &config

	config.UpdateProviderSettings()

	return &config
}

func (c *Config) GetRepositoryConfigPath() string {
	return filepath.Join(c.basePath, c.RepositoryMetadata)
}

func (c *Config) GetCandidatesBathPath() string {
	return filepath.Join(c.basePath, CandidatesDir)
}

func (c *Config) GetCandidateCachePath() string {
	return filepath.Join(c.basePath, CandidateCacheDir)
}

func (c *Config) UpdateProviderSettings() {
	for providerType, settings := range c.ProviderSettings {
		provider.Distributor(providerType).UpdateProviderSettings(types.ProviderSettings{
			CachePath:              path.Join(c.basePath, RepositoryCacheDir),
			CandidatesBathPath:     path.Join(c.basePath, CandidatesDir),
			CandidatesDownloadPath: path.Join(c.basePath, CandidateCacheDir),
			Setting:                settings,
		})
	}
}

func (c *Config) GetProviderSettings(provider types.ProviderType) types.ProviderSettings {
	setting := c.ProviderSettings[provider]
	return types.ProviderSettings{
		Setting: setting,
	}
}
