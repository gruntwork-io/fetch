package main

import (
	"net/http"
	"fmt"
	"strings"
	"bytes"
	"encoding/json"
)

type GitHubRepo struct {
	Owner string // The GitHub account name under which the repo exists
	Name  string // The GitHub repo name
}

// Modeled directly after the api.github.com response
type gitHubTagsApiResponse struct {
	Name       string // The tag name
	ZipBallUrl string // The URL where a ZIP of the release can be downloaded
	TarballUrl string // The URL where a Tarball of the release can be downloaded
	Commit     gitHubTagsCommitApiResponse
}

// Modeled directly after the api.github.com response
type gitHubTagsCommitApiResponse struct {
	Sha string // The SHA of the commit associated with a given tag
	Url string // The URL at which additional API information can be found for the given commit
}

func FetchReleases(githubRepoUrl string) ([]string, error) {
	repo, err := ExtractUrlIntoGitHubRepo(githubRepoUrl)
	if err != nil {
		return []string{}, err
	}

	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/%s/tags", repo.Owner, repo.Name))
	if err != nil {
		return []string{}, err
	}
	if resp.StatusCode != 200 {
		// Convert the resp.Body to a string
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		respBody := buf.String()

		return []string{}, fmt.Errorf("Received HTTP Response %d while fetching releases for GitHub URL %s. Full HTTP response: %s", resp.StatusCode, githubRepoUrl, respBody)
	}

	// Convert the response body to a byte array
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	jsonResp := buf.Bytes()

	// Extract the JSON into our array of gitHubTagsCommitApiResponse's
	var tags []gitHubTagsApiResponse
	err = json.Unmarshal(jsonResp, &tags)
	if err != nil {
		return []string{}, err
	}

	var tagsString []string
	for _, tag := range tags {
		tagsString = append(tagsString, tag.Name)
	}

	return tagsString, nil
}

func ExtractUrlIntoGitHubRepo(url string) (GitHubRepo, error) {
	if url[0:17] == "http://github.com" {
		tokens := strings.Split(url[18:], "/")
		return GitHubRepo{
			Owner: tokens[0],
			Name: tokens[1],
		}, nil
	} else if url[0:18] == "https://github.com" {
		tokens := strings.Split(url[19:], "/")
		return GitHubRepo{
			Owner: tokens[0],
			Name: tokens[1],
		}, nil
	} else {
		return GitHubRepo{}, fmt.Errorf("GitHub Repo URL %s did not begin with http://github.com or https://github.com", url)
	}
}

