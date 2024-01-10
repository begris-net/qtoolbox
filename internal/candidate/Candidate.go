package candidate

import (
	"fmt"
	"github.com/begris-net/qtoolbox/internal/types"
	"github.com/begris-net/qtoolbox/internal/util"
	"github.com/hashicorp/go-version"
	"path"
	"regexp"
	"strings"
)

type Candidate struct {
	Provider    CandidateProvider
	Version     version.Version
	DisplayName string
	Installed   bool
	Default     bool
}

type CandidateProvider struct {
	ProviderRepoId      string
	Product             string
	Id                  string
	Vendor              string
	Type                types.ProviderType
	Endpoint            string
	PreRelease          bool
	VersionCleanupRegex *regexp.Regexp
	Settings            map[string]any
}

type CandidateDescription struct {
	Name              string
	DisplayName       *string
	Description       *string
	DefaultProviderId *string
}

func (c Candidate) CandidateId() {

}

func (c Candidate) GetCandidateInstallationDir(candidatesBathPath string) string {
	return path.Join(candidatesBathPath, c.Provider.Product, c.DisplayName)
}

func (c Candidate) Show() {
	println(util.OrElse(c.DisplayName, c.Version.Original()))
}

func (c CandidateDescription) Show() {
	fmt.Printf("%s\n", strings.Repeat("-", 80))
	fmt.Printf("%s", util.SafeDeref(c.Description))
	cmd := fmt.Sprintf("$ toolbox install %s", c.Name)
	fmt.Printf("\n%s%s\n", strings.Repeat(" ", 80-len(cmd)), cmd)
}
