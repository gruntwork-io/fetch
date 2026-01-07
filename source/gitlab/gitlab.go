package gitlab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gruntwork-io/fetch/source"
	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
)

// GitLabSource implements source.Source for GitLab
type GitLabSource struct {
	config source.Config
	logger *logrus.Entry
}

// NewGitLabSource creates a new GitLab source
func NewGitLabSource(config source.Config) source.Source {
	return &GitLabSource{
		config: config,
		logger: config.Logger,
	}
}

// Type returns the source type
func (s *GitLabSource) Type() source.SourceType {
	return source.SourceTypeGitLab
}

// ParseUrl parses a GitLab repo URL into a Repo struct
// Supports nested subgroups: gitlab.com/group/subgroup/project
func (s *GitLabSource) ParseUrl(repoUrl, token string) (source.Repo, error) {
	var repo source.Repo

	u, err := url.Parse(repoUrl)
	if err != nil {
		return repo, fmt.Errorf("GitLab repo URL %s is malformed", repoUrl)
	}

	baseUrl := u.Host
	apiUrl := baseUrl // GitLab API is at same host with /api/v4 path

	// Parse path to extract owner (namespace) and project name
	// Path format: /group/subgroup/.../project or /user/project
	path := strings.Trim(u.Path, "/")

	// Remove common suffixes
	path = strings.TrimSuffix(path, ".git")
	path = strings.TrimSuffix(path, "/")

	// Split into parts
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return repo, fmt.Errorf("GitLab repo URL %s could not be parsed (need at least owner/project)", repoUrl)
	}

	// Last part is the project name, everything before is the namespace (owner)
	name := parts[len(parts)-1]
	owner := strings.Join(parts[:len(parts)-1], "/")

	if owner == "" || name == "" {
		return repo, fmt.Errorf("GitLab repo URL %s could not be parsed", repoUrl)
	}

	repo = source.Repo{
		Url:     repoUrl,
		BaseUrl: baseUrl,
		ApiUrl:  apiUrl,
		Owner:   owner, // Can be nested: group/subgroup
		Name:    name,
		Token:   token,
		Type:    source.SourceTypeGitLab,
	}

	return repo, nil
}

// FetchTags returns all semver tags from the repository
func (s *GitLabSource) FetchTags(repoUrl, token string) ([]string, error) {
	var tags []string

	repo, err := s.ParseUrl(repoUrl, token)
	if err != nil {
		return tags, err
	}

	// GitLab requires URL-encoded project path
	projectId := encodeProjectPath(repo.Owner, repo.Name)

	// Set per_page to 100 (max) to reduce network calls
	tagsUrl := fmt.Sprintf("https://%s/api/v4/projects/%s/repository/tags?per_page=100", repo.ApiUrl, projectId)

	for tagsUrl != "" {
		resp, err := callGitLabApiRaw(tagsUrl, "GET", repo.Token, map[string]string{})
		if err != nil {
			return tags, err
		}

		buf := new(bytes.Buffer)
		_, goErr := buf.ReadFrom(resp.Body)
		resp.Body.Close()
		if goErr != nil {
			return tags, goErr
		}
		jsonResp := buf.Bytes()

		var apiTags []GitLabTagResponse
		if err := json.Unmarshal(jsonResp, &apiTags); err != nil {
			return tags, err
		}

		for _, tag := range apiTags {
			// Skip non-semver tags
			if _, err := version.NewVersion(tag.Name); err == nil {
				tags = append(tags, tag.Name)
			}
		}

		// Handle pagination (GitLab uses same Link header format)
		tagsUrl = getNextUrl(resp.Header.Get("link"))
	}

	return tags, nil
}

// GetReleaseInfo returns release information for a specific tag
func (s *GitLabSource) GetReleaseInfo(repo source.Repo, tag string) (source.Release, error) {
	var release source.Release

	projectId := encodeProjectPath(repo.Owner, repo.Name)
	path := fmt.Sprintf("projects/%s/releases/%s", projectId, url.PathEscape(tag))

	resp, err := callGitLabApi(repo.ApiUrl, path, repo.Token, map[string]string{})
	if err != nil {
		return release, err
	}

	buf := new(bytes.Buffer)
	_, goErr := buf.ReadFrom(resp.Body)
	resp.Body.Close()
	if goErr != nil {
		return release, goErr
	}
	jsonResp := buf.Bytes()

	var apiRelease GitLabReleaseResponse
	if err := json.Unmarshal(jsonResp, &apiRelease); err != nil {
		return release, err
	}

	// Convert to generic Release type
	release = source.Release{
		Name: apiRelease.Name,
		Url:  fmt.Sprintf("https://%s/%s/%s/-/releases/%s", repo.BaseUrl, repo.Owner, repo.Name, tag),
	}

	// GitLab assets are in assets.links (uploaded files)
	for _, link := range apiRelease.Assets.Links {
		release.Assets = append(release.Assets, source.ReleaseAsset{
			Id:   link.Id,
			Url:  link.Url,
			Name: link.Name,
		})
	}

	return release, nil
}

// DownloadReleaseAsset downloads a release asset to destPath
func (s *GitLabSource) DownloadReleaseAsset(repo source.Repo, asset source.ReleaseAsset, destPath string, withProgress bool) error {
	// GitLab assets have direct URLs, so we just download from the URL
	httpClient := &http.Client{}

	request, err := http.NewRequest("GET", asset.Url, nil)
	if err != nil {
		return err
	}

	if repo.Token != "" {
		request.Header.Set("PRIVATE-TOKEN", repo.Token)
	}

	resp, err := httpClient.Do(request)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return fmt.Errorf("HTTP %d downloading asset: %s", resp.StatusCode, buf.String())
	}

	return writeResponseToDisk(resp, destPath, withProgress)
}

// MakeArchiveRequest creates HTTP request to download repo archive
func (s *GitLabSource) MakeArchiveRequest(commit source.Commit, token string) (*http.Request, error) {
	// Determine git ref (hierarchy: CommitSha > BranchName > GitTag > GitRef)
	var gitRef string
	if commit.CommitSha != "" {
		gitRef = commit.CommitSha
	} else if commit.BranchName != "" {
		gitRef = commit.BranchName
	} else if commit.GitTag != "" {
		gitRef = commit.GitTag
	} else if commit.GitRef != "" {
		gitRef = commit.GitRef
	} else {
		return nil, fmt.Errorf("no git reference specified (commit, branch, tag, or ref)")
	}

	projectId := encodeProjectPath(commit.Repo.Owner, commit.Repo.Name)

	// GitLab archive endpoint: /projects/:id/repository/archive.zip?sha=ref
	archiveUrl := fmt.Sprintf("https://%s/api/v4/projects/%s/repository/archive.zip?sha=%s",
		commit.Repo.ApiUrl, projectId, url.QueryEscape(gitRef))

	request, err := http.NewRequest("GET", archiveUrl, nil)
	if err != nil {
		return nil, err
	}

	if token != "" {
		request.Header.Set("PRIVATE-TOKEN", token)
	}

	return request, nil
}

// parseGitLabUrl is a helper to extract owner and project from various URL formats
func parseGitLabUrl(repoUrl string) (owner, name string, err error) {
	// Support multiple URL formats:
	// https://gitlab.com/owner/project
	// https://gitlab.com/group/subgroup/project
	// git@gitlab.com:owner/project.git

	u, err := url.Parse(repoUrl)
	if err != nil {
		return "", "", err
	}

	path := strings.Trim(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")

	// Use regex for more flexible parsing
	regex := regexp.MustCompile(`^(.+)/([^/]+)$`)
	matches := regex.FindStringSubmatch(path)

	if len(matches) != 3 {
		return "", "", fmt.Errorf("could not parse GitLab URL: %s", repoUrl)
	}

	return matches[1], matches[2], nil
}

func init() {
	// Register the factory function
	source.NewGitLabSource = NewGitLabSource
}
