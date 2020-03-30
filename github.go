package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/dustin/go-humanize"
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
	BaseUrl string
	ApiUrl  string
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
	Id     int
	Url    string
	Name   string
	Assets []GitHubReleaseAsset
}

// The "assets" portion of the GitHubReleaseApiResponse. Modeled directly after the api.github.com response (but only
// includes the fields we care about). For more info, see:
// https://developer.github.com/v3/repos/releases/#get-a-release-by-tag-name
type GitHubReleaseAsset struct {
	Id   int
	Url  string
	Name string
}

func ParseUrlIntoGithubInstance(repoUrl string, apiv string) (GitHubInstance, *FetchError) {
	var instance GitHubInstance

	u, err := url.Parse(repoUrl)
	if err != nil {
		return instance, newError(GITHUB_REPO_URL_MALFORMED_OR_NOT_PARSEABLE, fmt.Sprintf("GitHub Repo URL %s is malformed.", repoUrl))
	}

	baseUrl := u.Host
	apiUrl := "api.github.com"
	if baseUrl != "github.com" && baseUrl != "www.github.com" {
		fmt.Printf("Assuming GitHub Enterprise since the provided url (%s) does not appear to be for GitHub.com\n", repoUrl)
		apiUrl = baseUrl + "/api/" + apiv
	}

	instance = GitHubInstance{
		BaseUrl: baseUrl,
		ApiUrl:  apiUrl,
	}

	return instance, nil
}

// Fetch all tags from the given GitHub repo
func FetchTags(githubRepoUrl string, githubToken string, instance GitHubInstance) ([]string, *FetchError) {
	var tagsString []string

	repo, err := ParseUrlIntoGitHubRepo(githubRepoUrl, githubToken, instance)
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
	_, goErr := buf.ReadFrom(resp.Body)
	if goErr != nil {
		return tagsString, wrapError(goErr)
	}
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
func ParseUrlIntoGitHubRepo(url string, token string, instance GitHubInstance) (GitHubRepo, *FetchError) {
	var gitHubRepo GitHubRepo

	regex, regexErr := regexp.Compile("https?://(?:www\\.)?" + instance.BaseUrl + "/(.+?)/(.+?)(?:$|\\?|#|/)")
	if regexErr != nil {
		return gitHubRepo, newError(GITHUB_REPO_URL_MALFORMED_OR_NOT_PARSEABLE, fmt.Sprintf("GitHub Repo URL %s is malformed.", url))
	}

	matches := regex.FindStringSubmatch(url)
	if len(matches) != 3 {
		return gitHubRepo, newError(GITHUB_REPO_URL_MALFORMED_OR_NOT_PARSEABLE, fmt.Sprintf("GitHub Repo URL %s could not be parsed correctly", url))
	}

	gitHubRepo = GitHubRepo{
		Url:     url,
		BaseUrl: instance.BaseUrl,
		ApiUrl:  instance.ApiUrl,
		Owner:   matches[1],
		Name:    matches[2],
		Token:   token,
	}

	return gitHubRepo, nil
}

// Download the release asset with the given id and return its body
func DownloadReleaseAsset(repo GitHubRepo, assetId int, destPath string, withProgress bool) *FetchError {
	url := createGitHubRepoUrlForPath(repo, fmt.Sprintf("releases/assets/%d", assetId))
	resp, err := callGitHubApi(repo, url, map[string]string{"Accept": "application/octet-stream"})
	if err != nil {
		return err
	}
	return writeResonseToDisk(resp, destPath, withProgress)
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
	_, goErr := buf.ReadFrom(resp.Body)
	if goErr != nil {
		return release, wrapError(goErr)
	}
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

	request, err := http.NewRequest("GET", fmt.Sprintf("https://"+repo.ApiUrl+"/%s", path), nil)
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
		_, goErr := buf.ReadFrom(resp.Body)
		if goErr != nil {
			return nil, wrapError(goErr)
		}
		respBody := buf.String()

		// We leverage the HTTP Response Code as our ErrorCode here.
		return nil, newError(resp.StatusCode, fmt.Sprintf("Received HTTP Response %d while fetching releases for GitHub URL %s. Full HTTP response: %s", resp.StatusCode, repo.Url, respBody))
	}

	return resp, nil
}

type writeCounter struct {
	written uint64
	suffix string // contains " / SIZE MB" if size is known, otherwise empty
}

func newWriteCounter(total int64) *writeCounter {
	if total > 0 {
		return &writeCounter{
			suffix: fmt.Sprintf(" / %s", humanize.Bytes(uint64(total))),
		}
	}
	return &writeCounter{}
}

func (wc *writeCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.written += uint64(n)
	wc.PrintProgress()
	return n, nil
}

func (wc writeCounter) PrintProgress() {
	// Clear the line by using a character return to go back to the start and remove
	// the remaining characters by filling it with spaces
	fmt.Printf("\r%s", strings.Repeat(" ", 35))

	// Return again and print current status of download
	// We use the humanize package to print the bytes in a meaningful way (e.g. 10 MB)
	fmt.Printf("\rDownloading... %s%s", humanize.Bytes(wc.written), wc.suffix)
}

// Write the body of the given HTTP response to disk at the given path
func writeResonseToDisk(resp *http.Response, destPath string, withProgress bool) *FetchError {
	out, err := os.Create(destPath)
	if err != nil {
		return wrapError(err)
	}

	defer out.Close()
	defer resp.Body.Close()

	var readCloser io.Reader
	if withProgress{
		readCloser = io.TeeReader(resp.Body, newWriteCounter(resp.ContentLength))
	} else {
		readCloser = resp.Body
	}
	_, err = io.Copy(out, readCloser)
	return wrapError(err)
}
