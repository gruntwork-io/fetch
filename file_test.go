package main

import (
	"os"
	"testing"
	"path/filepath"
	"io/ioutil"
)

func TestDownloadZipFile(t *testing.T) {
	t.Parallel()

	cases := []struct {
		repoOwner   string
		repoName    string
		gitTag      string
		githubToken string
	}{
		{"gruntwork-io", "fetch-test-public", "v0.0.1", ""},
		{"gruntwork-io", "fetch-test-private", "v0.0.2", os.Getenv("GITHUB_OAUTH_TOKEN")},
	}

	for _, tc := range cases {
		githubCommit := gitHubCommit{
			repo: gitHubRepo{
				Owner: tc.repoOwner,
				Name: tc.repoName,
			},
			gitTag: tc.gitTag,
		}

		zipFilePath, err := downloadGithubZipFile(githubCommit, tc.githubToken)
		if err != nil {
			t.Fatalf("Failed to download file: %s", err)
		}

		if _, err := os.Stat(zipFilePath); os.IsNotExist(err) {
			t.Fatalf("Downloaded file doesn't exist at the expected path of %s", zipFilePath)
		}
	}
}

func TestDownloadZipFileWithBadRepoValues(t *testing.T) {
	t.Parallel()

	cases := []struct {
		repoOwner   string
		repoName    string
		gitTag      string
		githubToken string
	}{
		{"https://github.com/gruntwork-io/fetch-test-public/archive/does-not-exist.zip", "MyNameIsWhat", "x.y.z", ""},
	}

	for _, tc := range cases {
		githubCommit := gitHubCommit{
			repo: gitHubRepo{
				Owner: tc.repoOwner,
				Name: tc.repoName,
			},
			gitTag: tc.gitTag,
		}

		_, err := downloadGithubZipFile(githubCommit, tc.githubToken)
		if err == nil {
			t.Fatalf("Expected error for bad repo values: %s/%s:%s", tc.repoOwner, tc.repoName, tc.gitTag)
		}
	}
}

func TestExtractFiles(t *testing.T) {
	t.Parallel()

	cases := []struct {
		localFilePath     string
		filePathToExtract string
		expectedNumFiles  int
	}{
		{"test-fixtures/fetch-test-public-0.0.1.zip", "/", 1},
		{"test-fixtures/fetch-test-public-0.0.2.zip", "/", 2},
		{"test-fixtures/fetch-test-public-0.0.3.zip", "/", 4},
		{"test-fixtures/fetch-test-public-0.0.3.zip", "/folder", 2},
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
	}
}
