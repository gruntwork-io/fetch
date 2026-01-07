package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/gruntwork-io/fetch/source"
	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
)

// GitHubSource implements source.Source for GitHub
type GitHubSource struct {
	config source.Config
	logger *logrus.Entry
}

// NewGitHubSource creates a new GitHub source
func NewGitHubSource(config source.Config) source.Source {
	return &GitHubSource{
		config: config,
		logger: config.Logger,
	}
}

// Type returns the source type
func (s *GitHubSource) Type() source.SourceType {
	return source.SourceTypeGitHub
}

// ParseUrl parses a GitHub repo URL into a Repo struct
func (s *GitHubSource) ParseUrl(repoUrl, token string) (source.Repo, error) {
	var repo source.Repo

	u, err := url.Parse(repoUrl)
	if err != nil {
		return repo, fmt.Errorf("GitHub repo URL %s is malformed", repoUrl)
	}

	baseUrl := u.Host
	apiUrl := "api.github.com"
	apiVersion := s.config.ApiVersion
	if apiVersion == "" {
		apiVersion = "v3"
	}

	if baseUrl != "github.com" && baseUrl != "www.github.com" {
		if s.logger != nil {
			s.logger.Infof("Assuming GitHub Enterprise for URL: %s", repoUrl)
		}
		apiUrl = baseUrl + "/api/" + apiVersion
	}

	// Parse owner and repo name
	regex, err := regexp.Compile(`https?://(?:www\.)?` + regexp.QuoteMeta(baseUrl) + `/(.+?)/(.+?)(?:$|\?|#|/)`)
	if err != nil {
		return repo, fmt.Errorf("GitHub repo URL %s is malformed", repoUrl)
	}

	matches := regex.FindStringSubmatch(repoUrl)
	if len(matches) != 3 {
		return repo, fmt.Errorf("GitHub repo URL %s could not be parsed", repoUrl)
	}

	repo = source.Repo{
		Url:     repoUrl,
		BaseUrl: baseUrl,
		ApiUrl:  apiUrl,
		Owner:   matches[1],
		Name:    matches[2],
		Token:   token,
		Type:    source.SourceTypeGitHub,
	}

	return repo, nil
}

// FetchTags returns all semver tags from the repository
func (s *GitHubSource) FetchTags(repoUrl, token string) ([]string, error) {
	var tags []string

	repo, err := s.ParseUrl(repoUrl, token)
	if err != nil {
		return tags, err
	}

	// Set per_page to 100 (max) to reduce network calls
	tagsUrl := fmt.Sprintf("https://%s/repos/%s/%s/tags?per_page=100", repo.ApiUrl, repo.Owner, repo.Name)

	for tagsUrl != "" {
		resp, err := callGitHubApiRaw(tagsUrl, "GET", repo.Token, map[string]string{})
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

		var apiTags []GitHubTagsApiResponse
		if err := json.Unmarshal(jsonResp, &apiTags); err != nil {
			return tags, err
		}

		for _, tag := range apiTags {
			// Skip non-semver tags
			if _, err := version.NewVersion(tag.Name); err == nil {
				tags = append(tags, tag.Name)
			}
		}

		// Handle pagination
		tagsUrl = getNextUrl(resp.Header.Get("link"))
	}

	return tags, nil
}

// GetReleaseInfo returns release information for a specific tag
func (s *GitHubSource) GetReleaseInfo(repo source.Repo, tag string) (source.Release, error) {
	var release source.Release

	path := fmt.Sprintf("repos/%s/%s/releases/tags/%s", repo.Owner, repo.Name, tag)
	resp, err := callGitHubApi(repo.ApiUrl, path, repo.Token, map[string]string{})
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

	var apiRelease GitHubReleaseApiResponse
	if err := json.Unmarshal(jsonResp, &apiRelease); err != nil {
		return release, err
	}

	// Convert to generic Release type
	release = source.Release{
		Id:   apiRelease.Id,
		Url:  apiRelease.Url,
		Name: apiRelease.Name,
	}

	for _, asset := range apiRelease.Assets {
		release.Assets = append(release.Assets, source.ReleaseAsset{
			Id:   asset.Id,
			Url:  asset.Url,
			Name: asset.Name,
		})
	}

	return release, nil
}

// DownloadReleaseAsset downloads a release asset to destPath
func (s *GitHubSource) DownloadReleaseAsset(repo source.Repo, asset source.ReleaseAsset, destPath string, withProgress bool) error {
	path := fmt.Sprintf("repos/%s/%s/releases/assets/%d", repo.Owner, repo.Name, asset.Id)
	resp, err := callGitHubApi(repo.ApiUrl, path, repo.Token, map[string]string{"Accept": "application/octet-stream"})
	if err != nil {
		return err
	}
	return writeResponseToDisk(resp, destPath, withProgress)
}

// MakeArchiveRequest creates HTTP request to download repo archive
func (s *GitHubSource) MakeArchiveRequest(commit source.Commit, token string) (*http.Request, error) {
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

	archiveUrl := fmt.Sprintf("https://%s/repos/%s/%s/zipball/%s",
		commit.Repo.ApiUrl, commit.Repo.Owner, commit.Repo.Name, gitRef)

	request, err := http.NewRequest("GET", archiveUrl, nil)
	if err != nil {
		return nil, err
	}

	if token != "" {
		request.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	}

	return request, nil
}

// DownloadSourceZip downloads repo archive for a git ref and returns temp file path
func (s *GitHubSource) DownloadSourceZip(repo source.Repo, gitRef string) (string, error) {
	commit := source.Commit{
		Repo:   repo,
		GitRef: gitRef,
	}

	req, err := s.MakeArchiveRequest(commit, repo.Token)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error downloading source zip from %s: %v", req.URL.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received HTTP %d from %s", resp.StatusCode, req.URL.String())
	}

	return writeResponseToTempFile(resp)
}

func init() {
	// Register the factory function
	source.NewGitHubSource = NewGitHubSource
}
