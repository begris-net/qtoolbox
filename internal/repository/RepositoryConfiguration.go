package repository

import (
	"github.com/begris-net/qtoolbox/internal/types"
	"gopkg.in/yaml.v3"
	"os"
)

type Repository struct {
	UpdateURL  string          `yaml:"update-url"`
	Candidates []CandidateInfo `yaml:"candidates"`
}

type CandidateInfo struct {
	Name              string                  `yaml:"name"`
	DisplayName       string                  `yaml:"display-name,omitempty"`
	Description       string                  `yaml:"description,omitempty"`
	DefaultProviderId string                  `yaml:"default-provider-id,omitempty"`
	Provider          map[string]ProviderInfo `yaml:"provider,flow"`
}

type ProviderInfo struct {
	ID             string             `yaml:"id"`
	Vendor         string             `yaml:"vendor,omitempty"`
	Type           types.ProviderType `yaml:"type"`
	Endpoint       string             `yaml:"endpoint"`
	PreReleases    bool               `yaml:"pre-releases,omitempty"`
	VersionCleanup string             `yaml:"version-cleanup"`
	Settings       map[string]any     `yaml:"settings,omitempty"`
}

func LoadRepositoryConfig(configPath string) *Repository {

	data, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}

	var repository Repository
	if err := yaml.Unmarshal(data, &repository); err != nil {
		panic(err)
	}
	return &repository
}
