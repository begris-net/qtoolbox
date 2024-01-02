package provider

import (
	"fmt"
	"github.com/begris-net/qtoolbox/internal/log"
	v "github.com/hashicorp/go-version"
	"regexp"
)

func parseVersion(cleanupRegex *regexp.Regexp, version string) (*v.Version, error) {
	var candidateVersion *v.Version
	var extractedVersion string
	if cleanupRegex != nil {
		extractedVersion = cleanupRegex.ReplaceAllString(version, "")
	} else {
		extractedVersion = version
	}
	log.Logger.Debug(fmt.Sprintf("Extracted version: %v", extractedVersion), log.Logger.Args("original", version))
	candidateVersion, err := v.NewSemver(extractedVersion)
	if err != nil {
		log.Logger.Warn(fmt.Sprintf("Could not retrieve semantic version for release %s. Trying with more relaxed version instead. %v", version, err))
		candidateVersion, err = v.NewVersion(extractedVersion)
		if err != nil {
			log.Logger.Error(fmt.Sprintf("Could not retrieve version for release %s. %v", version, err))
		}
	}
	return candidateVersion, err
}
