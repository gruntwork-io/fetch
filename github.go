package main

import (
	"net/http"
	"fmt"
	"bytes"
	"encoding/json"
	"regexp"
)

type GitHubRepo struct {
	Owner string // The GitHub account name under which the repo exists
	Name  string // The GitHub repo name
}

type GitHubCommit struct {
	Repo       GitHubRepo // The GitHub repo where this release lives
	GitTag     string     // The specific git tag for this release
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

// Fetch all tags from the given GitHub repo
func FetchTags(githubRepoUrl string, githubToken string) ([]string, *FetchError) {
	var tagsString []string

	repo, err := ParseUrlIntoGitHubRepo(githubRepoUrl)
	if err != nil {
		return tagsString, wrapError(err)
	}

	// Make an HTTP request, possibly with the gitHubOAuthToken in the header
	httpClient := &http.Client{}

	req, err := MakeGitHubTagsRequest(repo, githubToken)
	if err != nil {
		return tagsString, wrapError(err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return tagsString, wrapError(err)
	}
	if resp.StatusCode != 200 {
		// Convert the resp.Body to a string
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		respBody := buf.String()

		// We leverage the HTTP Response Code as our ErrorCode here.
		return tagsString, newError(resp.StatusCode, fmt.Sprintf("Received HTTP Response %d while fetching releases for GitHub URL %s. Full HTTP response: %s", resp.StatusCode, githubRepoUrl, respBody))
	}

	// Convert the response body to a byte array
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	jsonResp := buf.Bytes()

	// Extract the JSON into our array of gitHubTagsCommitApiResponse's
	var tags []GitHubTagsApiResponse
	err = json.Unmarshal(jsonResp, &tags)
	if err != nil {
		return tagsString, wrapError(err)
	}

	for _, tag := range tags {
		tagsString = append(tagsString, tag.Name)
	}

	return tagsString, nil
}

// Convert a URL into a GitHubRepo struct
func ParseUrlIntoGitHubRepo(url string) (GitHubRepo, error) {
	var gitHubRepo GitHubRepo

	regex, regexErr := regexp.Compile("https?://(?:www\\.)?github.com/(.+?)/(.+?)(?:$|\\?|#|/)")
	if regexErr != nil {
		return gitHubRepo, newError(GITHUB_REPO_URL_MALFORMED_OR_NOT_PARSEABLE, fmt.Sprintf("GitHub Repo URL %s is malformed.", url))
	}

	matches := regex.FindStringSubmatch(url)
	if len(matches) != 3 {
		return gitHubRepo, newError(GITHUB_REPO_URL_MALFORMED_OR_NOT_PARSEABLE, fmt.Sprintf("GitHub Repo URL %s could not be parsed correctly", url))
	}

	gitHubRepo = GitHubRepo{
		Owner: matches[1],
		Name: matches[2],
	}

	return gitHubRepo, nil
}


// Return an HTTP request that will fetch the given GitHub repo's tags, possibly with the gitHubOAuthToken in the header
func MakeGitHubTagsRequest(repo GitHubRepo, gitHubToken string) (*http.Request, error) {
	var request *http.Request

	request, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/%s/tags", repo.Owner, repo.Name), nil)
	if err != nil {
		return request, wrapError(err)
	}

	if gitHubToken != "" {
		request.Header.Set("Authorization", fmt.Sprintf("token %s", gitHubToken))
	}

	return request, nil
}
