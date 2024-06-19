/*
 * Copyright (c) 2024 Bjoern Beier.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of
 * this software and associated documentation files (the "Software"), to deal in
 * the Software without restriction, including without limitation the rights to
 * use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
 * the Software, and to permit persons to whom the Software is furnished to do so,
 * subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
 * FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
 * COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
 * IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
 * CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

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
	ExportPath        *string                 `yaml:"export-path,omitempty"`
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
