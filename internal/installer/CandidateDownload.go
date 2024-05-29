package installer

import (
	"github.com/begris-net/qtoolbox/internal/candidate"
	"net/url"
)

type CandidateDownload struct {
	Candidate    candidate.Candidate
	DownloadUrl  *url.URL
	Checksum     string
	DownloadPath string
	InstallPath  string
	FileMode     string
}
