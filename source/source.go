package source

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

// SourceType identifies the provider type
type SourceType string

const (
	SourceTypeGitHub SourceType = "github"
	SourceTypeGitLab SourceType = "gitlab"
	SourceTypeAuto   SourceType = "auto"
)

// Repo represents a repository on any source
type Repo struct {
	Url     string     // Full repo URL
	BaseUrl string     // Host (github.com, gitlab.com, enterprise host)
	ApiUrl  string     // API endpoint base URL
	Owner   string     // Account/namespace (can be nested for GitLab: group/subgroup)
	Name    string     // Repository name
	Token   string     // Auth token
	Type    SourceType // Provider type
}

// Commit represents a git reference
type Commit struct {
	Repo       Repo
	GitRef     string // Fallback reference
	GitTag     string // Semantic version tag
	BranchName string // Branch name
	CommitSha  string // Exact commit SHA
}

// ReleaseAsset represents a downloadable release asset
type ReleaseAsset struct {
	Id   int    // Asset ID
	Url  string // Direct download URL
	Name string // Asset filename
}

// Release represents release info
type Release struct {
	Id     int
	Url    string
	Name   string
	Assets []ReleaseAsset
}

// Config holds source-specific configuration
type Config struct {
	ApiVersion string        // v3 for GitHub, v4 for GitLab
	Logger     *logrus.Entry // Logger instance
}

// Source interface defines operations every provider must implement
type Source interface {
	// Type returns the source type identifier
	Type() SourceType

	// ParseUrl parses repo URL into Repo struct
	ParseUrl(repoUrl, token string) (Repo, error)

	// FetchTags returns all semver tags from the repository
	FetchTags(repoUrl, token string) ([]string, error)

	// GetReleaseInfo returns release information for a specific tag
	GetReleaseInfo(repo Repo, tag string) (Release, error)

	// DownloadReleaseAsset downloads a release asset to destPath
	DownloadReleaseAsset(repo Repo, asset ReleaseAsset, destPath string, withProgress bool) error

	// MakeArchiveRequest creates HTTP request to download repo archive
	MakeArchiveRequest(commit Commit, token string) (*http.Request, error)
}
