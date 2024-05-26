package platform

import (
	"errors"
	"fmt"
	"github.com/YoshikiShibata/gostream"
	"github.com/begris-net/qtoolbox/internal/log"
	"reflect"
	"regexp"
)

const (
	Url_Template      = "url"
	FileMode          = "file-mode"
	OS_Mapping        = "os-mapping"
	Arch_Mapping      = "arch-mapping"
	Extention_Mapping = "ext-mapping"
	Archive           = "archive"
)

type PlatformHandler struct {
	settings map[string]any
}

func NewPlatformHandler(settings map[string]any) *PlatformHandler {
	return &PlatformHandler{
		settings,
	}
}

func (p *PlatformHandler) GetSetting(setting string) string {
	value := p.settings[setting]
	if value != nil {
		return reflect.ValueOf(value).String()
	}
	return ""
}

func (p *PlatformHandler) GetArchitectureRegex(arch string) (*regexp.Regexp, error) {
	switch arch {
	case "amd64":
		return regexp.MustCompile(".*(amd64|x86_64).*"), nil
	case "386":
		return regexp.MustCompile(".*(i386|x86)[\\d].*"), nil
	case "arm64":
		return regexp.MustCompile(".*(arm64).*"), nil
	case "arm":
		return regexp.MustCompile(".*(arm)[\\d].*"), nil
	}
	return nil, errors.New(fmt.Sprintf("No matcher available for architecture %s", arch))
}

func (p *PlatformHandler) GetExtensionRegex(os string) (*regexp.Regexp, error) {
	switch os {
	case "darwin":
		return regexp.MustCompile(".*(\\.tar\\.gz)$"), nil
	case "linux":
		return regexp.MustCompile(".*(\\.tar\\.gz)$"), nil
	case "windows":
		return regexp.MustCompile(".*(\\.zip)$"), nil
	}
	return nil, errors.New(fmt.Sprintf("No matcher available for os %s", os))
}

func (p *PlatformHandler) MapOS(os string) string {
	if p.settings[OS_Mapping] != nil {
		log.Logger.Debug("OS mapping:", log.Logger.Args("mappingTable", OS_Mapping, "mappings", p.settings[OS_Mapping]))
		return p.MapOriginalValue(p.settings[OS_Mapping], os, os)
	}
	return os
}

func (p *PlatformHandler) MapArchitecture(arch string) string {
	if p.settings[Arch_Mapping] != nil {
		log.Logger.Debug("Arch mapping:", log.Logger.Args("mappingTable", Arch_Mapping, "mappings", p.settings[Arch_Mapping]))
		return p.MapOriginalValue(p.settings[Arch_Mapping], arch, arch)
	}
	return arch
}

func (p *PlatformHandler) MapExtension(ext string) string {
	if p.settings[Extention_Mapping] != nil {
		log.Logger.Debug("Extention mapping:", log.Logger.Args("mappingTable", Extention_Mapping, "mappings", p.settings[Extention_Mapping]))
		return p.MapOriginalValue(p.settings[Extention_Mapping], ext, "")
	}
	return ext
}

func (p *PlatformHandler) MapOriginalValue(mappingTable any, value string, defaultValue string) string {
	log.Logger.Trace("Mapping:", log.Logger.Args("mappingTable", mappingTable, "original_value", value))
	var mappings = reflect.ValueOf(mappingTable)
	if mappings.IsValid() {
		isPresent := gostream.Of(mappings.MapKeys()...).
			Filter(func(t reflect.Value) bool {
				return t.IsValid() && t.String() == value
			}).FindFirst().IsPresent()

		if isPresent {
			mapping := mappings.MapIndex(reflect.ValueOf(value))
			if mapping.IsValid() {
				log.Logger.Debug(fmt.Sprintf("Replacing original value %s with %s", value, mapping.Elem().String()))
				return mapping.Elem().String()
			}
		}
	}
	return defaultValue
}
