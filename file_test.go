package main

import (
	"os"
	"testing"
)

func TestDownloadZipFile(t *testing.T) {
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
		_, zipFilePath, err := downloadGithubZipFile(tc.repoOwner, tc.repoName, tc.gitTag, tc.githubToken)
		if err != nil {
			t.Fatalf("Failed to download file: %s", err)
		}

		if _, err := os.Stat(zipFilePath); os.IsNotExist(err) {
			t.Fatalf("Downloaded file doesn't exist at the expected path of %s", zipFilePath)
		}
	}
}

func TestDownloadZipFileWithBadRepoValues(t *testing.T) {
	cases := []struct {
		repoOwner   string
		repoName    string
		gitTag      string
		githubToken string
	}{
		{"https://github.com/gruntwork-io/fetch-test-public/archive/does-not-exist.zip", "MyNameIsWhat", "x.y.z", ""},
	}

	for _, tc := range cases {
		_, _, err := downloadGithubZipFile(tc.repoOwner, tc.repoName, tc.gitTag, tc.githubToken)
		if err == nil {
			t.Fatalf("Expected error for bad repo values: %s/%s:%s", tc.repoOwner, tc.repoName, tc.gitTag)
		}
	}
}
