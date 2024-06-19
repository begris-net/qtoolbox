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
	log.Logger.Trace(fmt.Sprintf("Extracted version: %v", extractedVersion), log.Logger.Args("original", version))
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
