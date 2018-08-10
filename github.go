package main

import (
	"net/http"
	"fmt"
	"bytes"
	"encoding/json"
	"regexp"
	"os"
	"io"
)

type GitHubRepo struct {
	Url     string // The URL of the GitHub repo
	BaseUrl string // The Base URL of the GitHub Instance
	ApiUrl  string // The API Url of the GitHub Instance
	Owner   string // The GitHub account name under which the repo exists
	Name    string // The GitHub repo name
	Token   string // The personal access token to access this repo (if it's a private repo)
}

type GitHubInstance struct {
  BaseUrl    string
  ApiUrl     string
}

// Represents a specific git commit.
// Note that code using GitHub Commit should respect the following hierarchy:
// - CommitSha > BranchName > GitTag
// - Example: GitTag and BranchName are both specified; use the GitTag
// - Example: GitTag and CommitSha are both specified; use the CommitSha
// - Example: BranchName alone is specified; use BranchName
type GitHubCommit struct {
	Repo       GitHubRepo // The GitHub repo where this release lives
	GitTag     string     // The specific git tag for this release
	BranchName string     // If specified, indicates that this commit should be the latest commit on the given branch
	CommitSha  string     // If specified, indicates that this commit should be exactly this Git Commit SHA.
}

// Modeled directly after the api.github.com response
type GitHubTagsApiResponse struct {
	Name       string // The tag name
	ZipBallUrl string // The URL where a ZIP of the release can be downloaded
	TarballUrl string // The URL where a Tarball of the release can be downloaded
	Commit     GitHubTagsCommitApiResponse
}

// Modeled directly after the api.github.com response
type GitHubTagsCommitApiResponse struct {
	Sha string // The SHA of the commit associated with a given tag
	Url string // The URL at which additional API information can be found for the given commit
}

// Modeled directly after the api.github.com response (but only includes the fields we care about). For more info, see:
// https://developer.github.com/v3/repos/releases/#get-a-release-by-tag-name
type GitHubReleaseApiResponse struct {
	Id      int
	Url     string
	Name    string
	Assets  []GitHubReleaseAsset
}

// The "assets" portion of the GitHubReleaseApiResponse. Modeled directly after the api.github.com response (but only
// includes the fields we care about). For more info, see:
// https://developer.github.com/v3/repos/releases/#get-a-release-by-tag-name
type GitHubReleaseAsset struct {
	Id   int
	Url  string
	Name string
}

func ParseUrlIntoGithubInstance(url string, apiv string) (GitHubInstance, *FetchError) {
  var instance GitHubInstance

  regex, regexErr := regexp.Compile("https?://(?:www\\.)?(.+?\\.com).*")
  if regexErr != nil {
		return instance, newError(GITHUB_REPO_URL_MALFORMED_OR_NOT_PARSEABLE, fmt.Sprintf("GitHub Repo URL %s is malformed.", url))
  }

  matches := regex.FindStringSubmatch(url)
  if len(matches) != 2 {
		return instance, newError(GITHUB_REPO_URL_MALFORMED_OR_NOT_PARSEABLE, fmt.Sprintf("GitHub Repo URL %s could not be parsed correctly", url))
  }

  baseUrl := matches[1]
  apiUrl := "api.github.com"
  if baseUrl != "github.com" && baseUrl != "www.github.com" {
    fmt.Printf("Assuming GitHub Enterprise since the provided url (%s) does not appear to be for GitHub.com\n", url)
    apiUrl = baseUrl + "/api/" + apiv
  }

  instance = GitHubInstance{
    BaseUrl: baseUrl,
    ApiUrl: apiUrl,
  }

  return instance, nil
}

// Fetch all tags from the given GitHub repo
func FetchTags(githubRepoUrl string, githubBaseUrl string, githubApiUrl string, githubToken string) ([]string, *FetchError) {
	var tagsString []string

	repo, err := ParseUrlIntoGitHubRepo(githubRepoUrl, githubBaseUrl, githubApiUrl, githubToken)
	if err != nil {
		return tagsString, wrapError(err)
	}

	url := createGitHubRepoUrlForPath(repo, "tags")
	resp, err := callGitHubApi(repo, url, map[string]string{})
	if err != nil {
		return tagsString, err
	}

	// Convert the response body to a byte array
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	jsonResp := buf.Bytes()

	// Extract the JSON into our array of gitHubTagsCommitApiResponse's
	var tags []GitHubTagsApiResponse
	if err := json.Unmarshal(jsonResp, &tags); err != nil {
		return tagsString, wrapError(err)
	}

	for _, tag := range tags {
		tagsString = append(tagsString, tag.Name)
	}

	return tagsString, nil
}

// Convert a URL into a GitHubRepo struct
func ParseUrlIntoGitHubRepo(url string, githubBaseUrl string, githubApiUrl string, token string) (GitHubRepo, *FetchError) {
	var gitHubRepo GitHubRepo

	regex, regexErr := regexp.Compile("https?://(?:www\\.)?" + githubBaseUrl + "/(.+?)/(.+?)(?:$|\\?|#|/)")
	if regexErr != nil {
		return gitHubRepo, newError(GITHUB_REPO_URL_MALFORMED_OR_NOT_PARSEABLE, fmt.Sprintf("GitHub Repo URL %s is malformed.", url))
	}

	matches := regex.FindStringSubmatch(url)
	if len(matches) != 3 {
		return gitHubRepo, newError(GITHUB_REPO_URL_MALFORMED_OR_NOT_PARSEABLE, fmt.Sprintf("GitHub Repo URL %s could not be parsed correctly", url))
	}

	gitHubRepo = GitHubRepo{
		Url:     url,
		BaseUrl: githubBaseUrl,
		ApiUrl:  githubApiUrl,
		Owner:   matches[1],
		Name:    matches[2],
		Token:   token,
	}

	return gitHubRepo, nil
}

// Download the release asset with the given id and return its body
func DownloadReleaseAsset(repo GitHubRepo, assetId int, destPath string) *FetchError {
	url := createGitHubRepoUrlForPath(repo, fmt.Sprintf("releases/assets/%d", assetId))
	resp, err := callGitHubApi(repo, url, map[string]string{"Accept": "application/octet-stream"})
	if err != nil {
		return err
	}

	return writeResonseToDisk(resp, destPath)
}

// Get information about the GitHub release with the given tag
func GetGitHubReleaseInfo(repo GitHubRepo, tag string) (GitHubReleaseApiResponse, *FetchError) {
	release := GitHubReleaseApiResponse{}

	url := createGitHubRepoUrlForPath(repo, fmt.Sprintf("releases/tags/%s", tag))
	resp, err := callGitHubApi(repo, url, map[string]string{})
	if err != nil {
		return release, err
	}

	// Convert the response body to a byte array
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	jsonResp := buf.Bytes()

	if err := json.Unmarshal(jsonResp, &release); err != nil {
		return release, wrapError(err)
	}

	return release, nil
}

// Craft a URL for the GitHub repos API of the form repos/:owner/:repo/:path
func createGitHubRepoUrlForPath(repo GitHubRepo, path string) string {
	return fmt.Sprintf("repos/%s/%s/%s", repo.Owner, repo.Name, path)
}

// Call the GitHub API at the given path and return the HTTP response
func callGitHubApi(repo GitHubRepo, path string, customHeaders map[string]string) (*http.Response, *FetchError) {
	httpClient := &http.Client{}

	request, err := http.NewRequest("GET", fmt.Sprintf("https://" + repo.ApiUrl + "/%s", path), nil)
	if err != nil {
		return nil, wrapError(err)
	}

	if repo.Token != "" {
		request.Header.Set("Authorization", fmt.Sprintf("token %s", repo.Token))
	}

	for headerName, headerValue := range customHeaders {
		request.Header.Set(headerName, headerValue)
	}

	resp, err := httpClient.Do(request)
	if err != nil {
		return nil, wrapError(err)
	}
	if resp.StatusCode != 200 {
		// Convert the resp.Body to a string
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		respBody := buf.String()

		// We leverage the HTTP Response Code as our ErrorCode here.
		return nil, newError(resp.StatusCode, fmt.Sprintf("Received HTTP Response %d while fetching releases for GitHub URL %s. Full HTTP response: %s", resp.StatusCode, repo.Url, respBody))
	}

	return resp, nil
}

// Write the body of the given HTTP response to disk at the given path
func writeResonseToDisk(resp *http.Response, destPath string) *FetchError {
	out, err := os.Create(destPath)
	if err != nil  {
		return wrapError(err)
	}

	defer out.Close()
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	return wrapError(err)
}