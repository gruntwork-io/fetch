package main

import (
	"os"
	"testing"
	"path/filepath"
	"io/ioutil"
	"fmt"
	"strings"
)

// Although other tests besides those in this file require this env var, this init() func will cover all tests.
func init() {
	if os.Getenv("GITHUB_OAUTH_TOKEN") == "" {
		fmt.Println("ERROR: These tests require that env var GITHUB_OAUTH_TOKEN be set to a GitHub Personal Access Token.")
		fmt.Println("See the tests cases to see which GitHub repos the oAuth token needs access to.")
		os.Exit(1)
	}
}

func TestDownloadGitTagZipFile(t *testing.T) {
	t.Parallel()

	publicGitHub := GitHubInstance{
		BaseUrl: "github.com",
		ApiUrl:  "api.github.com",
	}

	enterpriseGitHubExample := GitHubInstance{
		BaseUrl: "github.acme.com",
		ApiUrl: "github.acme.com/api/v3",
	}

	cases := []struct {
		instance 	GitHubInstance
		repoOwner   string
		repoName    string
		gitTag      string
		githubToken string
	}{
		{publicGitHub, "gruntwork-io", "fetch-test-public", "v0.0.1", ""},
		{publicGitHub, "gruntwork-io", "fetch-test-private", "v0.0.2", os.Getenv("GITHUB_OAUTH_TOKEN")},
		{enterpriseGitHubExample, "temp-internal-org", "bash-commons", "v0.0.4", os.Getenv("GITHUB_OAUTH_TOKEN")},
	}

	for _, tc := range cases {
		gitHubCommit := GitHubCommit{
			Repo: GitHubRepo{
				Owner: tc.repoOwner,
				Name: tc.repoName,
			},
			GitTag: tc.gitTag,
		}

		zipFilePath, err := downloadGithubZipFile(gitHubCommit, tc.githubToken, tc.instance)

		defer os.RemoveAll(zipFilePath)

		// We don't have a running instance of GitHub Enterprise against which to validate tests as we do for GitHub public,
		// so this test will only validate that fetch attempted to download from the expected URL. The download itself
		// will fail.

		githubEnterpriseDownloadUrl := fmt.Sprintf("https://%s/repos/%s/%s/zipball/%s", tc.instance.ApiUrl, tc.repoOwner, tc.repoName, tc.gitTag)

		// TODO: The awkwardness of this test makes it clear that a better structure for this program would be to refactor
		// the downloadGithubZipFile() function to a function called downloadGithubFile() that would accept a URL as a
		// param. We could then test explicitly that the URL is as expected, which would make GitHub Enterprise test cases
		// simpler to handle.

		if err != nil && strings.Contains(err.Error(), "no such host") {
			if strings.Contains(err.Error(), githubEnterpriseDownloadUrl) {
				t.Logf("Found expected download URL %s. Download itself failed as expected because no GitHub Enterprise instance exists at the given URL.", githubEnterpriseDownloadUrl)
				return
			} else {
				t.Fatalf("Attempted to download from URL other than the expected download URL of %s. Full error: %s", githubEnterpriseDownloadUrl, err.Error())
			}
		}

		if err != nil {
			t.Fatalf("Failed to download file: %s", err)
		}

		if _, err := os.Stat(zipFilePath); os.IsNotExist(err) {
			t.Fatalf("Downloaded file doesn't exist at the expected path of %s", zipFilePath)
		}
	}
}

func TestDownloadGitBranchZipFile(t *testing.T) {
	t.Parallel()

	publicGitHub := GitHubInstance{
		BaseUrl: "github.com",
		ApiUrl:  "api.github.com",
	}

	cases := []struct {
		instance 	GitHubInstance
		repoOwner   string
		repoName    string
		branchName  string
		githubToken string
	}{
		{publicGitHub, "gruntwork-io", "fetch-test-public", "sample-branch", ""},
		{publicGitHub, "gruntwork-io", "fetch-test-private", "sample-branch", os.Getenv("GITHUB_OAUTH_TOKEN")},
	}

	for _, tc := range cases {
		gitHubCommit := GitHubCommit{
			Repo: GitHubRepo{
				Owner: tc.repoOwner,
				Name: tc.repoName,
			},
			BranchName: tc.branchName,
		}

		zipFilePath, err := downloadGithubZipFile(gitHubCommit, tc.githubToken, tc.instance)
		defer os.RemoveAll(zipFilePath)
		if err != nil {
			t.Fatalf("Failed to download file: %s", err)
		}

		if _, err := os.Stat(zipFilePath); os.IsNotExist(err) {
			t.Fatalf("Downloaded file doesn't exist at the expected path of %s", zipFilePath)
		}
	}
}

func TestDownloadBadGitBranchZipFile(t *testing.T) {
	t.Parallel()

	publicGitHub := GitHubInstance{
		BaseUrl: "github.com",
		ApiUrl:  "api.github.com",
	}

	cases := []struct {
		instance 	GitHubInstance
		repoOwner   string
		repoName    string
		branchName  string
		githubToken string
	}{
		{publicGitHub, "gruntwork-io", "fetch-test-public", "branch-that-doesnt-exist", ""},
	}

	for _, tc := range cases {
		gitHubCommit := GitHubCommit{
			Repo: GitHubRepo{
				Owner: tc.repoOwner,
				Name: tc.repoName,
			},
			BranchName: tc.branchName,
		}

		zipFilePath, err := downloadGithubZipFile(gitHubCommit, tc.githubToken, tc.instance)
		defer os.RemoveAll(zipFilePath)
		if err == nil {
			t.Fatalf("Expected that attempt to download repo %s/%s for branch \"%s\" would fail, but received no error.", tc.repoOwner, tc.repoName, tc.branchName)
		}
	}
}

func TestDownloadGitCommitFile(t *testing.T) {
	t.Parallel()

	publicGitHub := GitHubInstance{
		BaseUrl: "github.com",
		ApiUrl:  "api.github.com",
	}

	cases := []struct {
		instance 	GitHubInstance
		repoOwner   string
		repoName    string
		commitSha   string
		githubToken string
	}{
		{publicGitHub, "gruntwork-io", "fetch-test-public", "d2de34edb4c6564e0674b3f390b3b1fb0468183a", ""},
		{publicGitHub, "gruntwork-io", "fetch-test-public", "57752e7f1df0acbd3c1e61545d5c4d0e87699d84", ""},
		{publicGitHub, "gruntwork-io", "fetch-test-public", "f32a08313e30f116a1f5617b8b68c11f1c1dbb61", ""},
		{publicGitHub, "gruntwork-io", "fetch-test-private", "676cfb92b54d33538c756c7a9479bfc3f6b44de2", os.Getenv("GITHUB_OAUTH_TOKEN")},
	}

	for _, tc := range cases {
		gitHubCommit := GitHubCommit{
			Repo: GitHubRepo{
				Owner: tc.repoOwner,
				Name: tc.repoName,
			},
			CommitSha: tc.commitSha,
		}

		zipFilePath, err := downloadGithubZipFile(gitHubCommit, tc.githubToken, tc.instance)
		defer os.RemoveAll(zipFilePath)
		if err != nil {
			t.Fatalf("Failed to download file: %s", err)
		}

		if _, err := os.Stat(zipFilePath); os.IsNotExist(err) {
			t.Fatalf("Downloaded file doesn't exist at the expected path of %s", zipFilePath)
		}
	}
}

func TestDownloadBadGitCommitFile(t *testing.T) {
	t.Parallel()

	publicGitHub := GitHubInstance{
		BaseUrl: "github.com",
		ApiUrl:  "api.github.com",
	}

	cases := []struct {
		instance 	GitHubInstance
		repoOwner   string
		repoName    string
		commitSha   string
		githubToken string
	}{
		{publicGitHub, "gruntwork-io", "fetch-test-public", "hello-world", ""},
		{publicGitHub, "gruntwork-io", "fetch-test-public", "i-am-a-non-existent-commit", ""},
		// remove a single letter from the beginning of an otherwise legit commit sha
		// interestingly, through testing I found that GitHub will attempt to find the right commit sha if you
		// truncate the end of it.
		{publicGitHub, "gruntwork-io", "fetch-test-public", "7752e7f1df0acbd3c1e61545d5c4d0e87699d84", ""},
	}

	for _, tc := range cases {
		gitHubCommit := GitHubCommit{
			Repo: GitHubRepo{
				Owner: tc.repoOwner,
				Name: tc.repoName,
			},
			CommitSha: tc.commitSha,
		}

		zipFilePath, err := downloadGithubZipFile(gitHubCommit, tc.githubToken, tc.instance)
		defer os.RemoveAll(zipFilePath)
		if err == nil {
			t.Fatalf("Expected that attempt to download repo %s/%s at commmit sha \"%s\" would fail, but received no error.", tc.repoOwner, tc.repoName, tc.commitSha)
		}
	}
}

func TestDownloadZipFileWithBadRepoValues(t *testing.T) {
	t.Parallel()

	publicGitHub := GitHubInstance{
		BaseUrl: "github.com",
		ApiUrl:  "api.github.com",
	}

	cases := []struct {
		instance 	GitHubInstance
		repoOwner   string
		repoName    string
		gitTag      string
		githubToken string
	}{
		{publicGitHub, "https://github.com/gruntwork-io/fetch-test-public/archive/does-not-exist.zip", "MyNameIsWhat", "x.y.z", ""},
	}

	for _, tc := range cases {
		gitHubCommit := GitHubCommit{
			Repo: GitHubRepo{
				Owner: tc.repoOwner,
				Name: tc.repoName,
			},
			GitTag: tc.gitTag,
		}

		_, err := downloadGithubZipFile(gitHubCommit, tc.githubToken, tc.instance)
		if err == nil && err.errorCode != 500 {
			t.Fatalf("Expected error for bad repo values: %s/%s:%s", tc.repoOwner, tc.repoName, tc.gitTag)
		}
	}
}

func TestExtractFiles(t *testing.T) {
	t.Parallel()

	publicGitHub := GitHubInstance{
		BaseUrl: "github.com",
		ApiUrl:  "api.github.com",
	}

	cases := []struct {
		instance 		  GitHubInstance
		localFilePath     string
		filePathToExtract string
		expectedNumFiles  int
		nonemptyFiles     []string
	}{
		{publicGitHub, "test-fixtures/fetch-test-public-0.0.1.zip", "/", 1, nil},
		{publicGitHub, "test-fixtures/fetch-test-public-0.0.2.zip", "/", 2, nil},
		{publicGitHub, "test-fixtures/fetch-test-public-0.0.3.zip", "/", 4, []string{"/README.md"} },
		{publicGitHub, "test-fixtures/fetch-test-public-0.0.3.zip", "/folder", 2, nil},
	}

	for _, tc := range cases {
		// Create a temp directory
		tempDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %s", err)
		}
		defer os.RemoveAll(tempDir)

		err = extractFiles(tc.localFilePath, tc.filePathToExtract, tempDir)
		if err != nil {
			t.Fatalf("Failed to extract files: %s", err)
		}

		// Count the number of files in the directory
		var numFiles int
		filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
			if ! info.IsDir() {
				numFiles++
			}
			return nil
		})

		if (numFiles != tc.expectedNumFiles) {
			t.Fatalf("While extracting %s, expected to find %d file(s), but found %d. Local path = %s", tc.localFilePath, tc.expectedNumFiles, numFiles, tempDir)
		}

		// Ensure that files declared to be non-empty are in fact non-empty
		filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
			relativeFilename := strings.TrimPrefix(path, tempDir)

			if ! info.IsDir() && stringInSlice(relativeFilename, tc.nonemptyFiles) {
				if info.Size() == 0 {
					t.Fatalf("Expected %s in %s to have non-zero file size, but found file size = %d.\n", relativeFilename, tc.localFilePath, info.Size())
				}
			}
			return nil
		})

	}
}

// Return ture if the given slice contains the given string
func stringInSlice(s string, slice []string) bool {
	for _, val := range slice {
		if val == s {
			return true
		}
	}
	return false
}
