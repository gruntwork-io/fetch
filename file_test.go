package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/gruntwork-io/fetch/source"
	_ "github.com/gruntwork-io/fetch/source/github"
	"github.com/stretchr/testify/require"
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

	config := source.Config{
		ApiVersion: "v3",
		Logger:     GetProjectLogger(),
	}
	src, err := source.NewSource(source.TypeGitHub, config)
	require.NoError(t, err)

	cases := []struct {
		repoUrl     string
		gitTag      string
		githubToken string
	}{
		{"https://github.com/gruntwork-io/fetch-test-public", "v0.0.1", ""},
		{"https://github.com/gruntwork-io/fetch-test-private", "v0.0.2", os.Getenv("GITHUB_OAUTH_TOKEN")},
	}

	for _, tc := range cases {
		repo, err := src.ParseUrl(tc.repoUrl, tc.githubToken)
		require.NoError(t, err)

		zipFilePath, err := src.DownloadSourceZip(repo, tc.gitTag)
		defer os.RemoveAll(zipFilePath)

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

	config := source.Config{
		ApiVersion: "v3",
		Logger:     GetProjectLogger(),
	}
	src, err := source.NewSource(source.TypeGitHub, config)
	require.NoError(t, err)

	cases := []struct {
		repoUrl     string
		branchName  string
		githubToken string
	}{
		{"https://github.com/gruntwork-io/fetch-test-public", "sample-branch", ""},
		{"https://github.com/gruntwork-io/fetch-test-private", "sample-branch", os.Getenv("GITHUB_OAUTH_TOKEN")},
	}

	for _, tc := range cases {
		repo, err := src.ParseUrl(tc.repoUrl, tc.githubToken)
		require.NoError(t, err)

		zipFilePath, err := src.DownloadSourceZip(repo, tc.branchName)
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

	config := source.Config{
		ApiVersion: "v3",
		Logger:     GetProjectLogger(),
	}
	src, err := source.NewSource(source.TypeGitHub, config)
	require.NoError(t, err)

	cases := []struct {
		repoUrl     string
		branchName  string
		githubToken string
	}{
		{"https://github.com/gruntwork-io/fetch-test-public", "branch-that-doesnt-exist", ""},
	}

	for _, tc := range cases {
		repo, err := src.ParseUrl(tc.repoUrl, tc.githubToken)
		require.NoError(t, err)

		zipFilePath, err := src.DownloadSourceZip(repo, tc.branchName)
		defer os.RemoveAll(zipFilePath)

		if err == nil {
			t.Fatalf("Expected that attempt to download repo %s for branch \"%s\" would fail, but received no error.", tc.repoUrl, tc.branchName)
		}
	}
}

func TestDownloadGitCommitFile(t *testing.T) {
	t.Parallel()

	config := source.Config{
		ApiVersion: "v3",
		Logger:     GetProjectLogger(),
	}
	src, err := source.NewSource(source.TypeGitHub, config)
	require.NoError(t, err)

	cases := []struct {
		repoUrl     string
		commitSha   string
		githubToken string
	}{
		{"https://github.com/gruntwork-io/fetch-test-public", "d2de34edb4c6564e0674b3f390b3b1fb0468183a", ""},
		{"https://github.com/gruntwork-io/fetch-test-public", "57752e7f1df0acbd3c1e61545d5c4d0e87699d84", ""},
		{"https://github.com/gruntwork-io/fetch-test-public", "f32a08313e30f116a1f5617b8b68c11f1c1dbb61", ""},
		{"https://github.com/gruntwork-io/fetch-test-private", "676cfb92b54d33538c756c7a9479bfc3f6b44de2", os.Getenv("GITHUB_OAUTH_TOKEN")},
	}

	for _, tc := range cases {
		repo, err := src.ParseUrl(tc.repoUrl, tc.githubToken)
		require.NoError(t, err)

		zipFilePath, err := src.DownloadSourceZip(repo, tc.commitSha)
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

	config := source.Config{
		ApiVersion: "v3",
		Logger:     GetProjectLogger(),
	}
	src, err := source.NewSource(source.TypeGitHub, config)
	require.NoError(t, err)

	cases := []struct {
		repoUrl     string
		commitSha   string
		githubToken string
	}{
		{"https://github.com/gruntwork-io/fetch-test-public", "hello-world", ""},
		{"https://github.com/gruntwork-io/fetch-test-public", "i-am-a-non-existent-commit", ""},
		// remove a single letter from the beginning of an otherwise legit commit sha
		// interestingly, through testing I found that GitHub will attempt to find the right commit sha if you
		// truncate the end of it.
		{"https://github.com/gruntwork-io/fetch-test-public", "7752e7f1df0acbd3c1e61545d5c4d0e87699d84", ""},
	}

	for _, tc := range cases {
		repo, err := src.ParseUrl(tc.repoUrl, tc.githubToken)
		require.NoError(t, err)

		zipFilePath, err := src.DownloadSourceZip(repo, tc.commitSha)
		defer os.RemoveAll(zipFilePath)

		if err == nil {
			t.Fatalf("Expected that attempt to download repo %s at commit sha \"%s\" would fail, but received no error.", tc.repoUrl, tc.commitSha)
		}
	}
}

func TestDownloadZipFileWithBadRepoValues(t *testing.T) {
	t.Parallel()

	config := source.Config{
		ApiVersion: "v3",
		Logger:     GetProjectLogger(),
	}
	src, err := source.NewSource(source.TypeGitHub, config)
	require.NoError(t, err)

	cases := []struct {
		repoUrl     string
		gitTag      string
		githubToken string
	}{
		{"https://github.com/gruntwork-io/non-existent-repo", "x.y.z", ""},
	}

	for _, tc := range cases {
		repo, err := src.ParseUrl(tc.repoUrl, tc.githubToken)
		require.NoError(t, err)

		zipFilePath, err := src.DownloadSourceZip(repo, tc.gitTag)
		defer os.RemoveAll(zipFilePath)

		if err == nil {
			t.Fatalf("Expected error for bad repo values: %s:%s", tc.repoUrl, tc.gitTag)
		}
	}
}

func TestExtractFiles(t *testing.T) {
	t.Parallel()

	cases := []struct {
		localFilePath     string
		filePathToExtract string
		expectedNumFiles  int
		nonemptyFiles     []string
	}{
		{"test-fixtures/fetch-test-public-0.0.1.zip", "/", 1, nil},
		{"test-fixtures/fetch-test-public-0.0.2.zip", "/", 2, nil},
		{"test-fixtures/fetch-test-public-0.0.3.zip", "/", 4, []string{"/README.md"}},
		{"test-fixtures/fetch-test-public-0.0.3.zip", "/folder", 2, nil},
		{"test-fixtures/fetch-test-public-0.0.4.zip", "/aaa", 2, []string{"/hello.txt", "/subaaa/subhello.txt"}},
	}

	for _, tc := range cases {
		// Create a temp directory
		tempDir, err := os.MkdirTemp("", "")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %s", err)
		}
		defer os.RemoveAll(tempDir)

		fileCount, err := extractFiles(tc.localFilePath, tc.filePathToExtract, tempDir)
		if err != nil {
			t.Fatalf("Failed to extract files: %s", err)
		}

		if fileCount != tc.expectedNumFiles {
			t.Fatalf("Expected to extract %d files, extracted %d instead", tc.expectedNumFiles, fileCount)
		}

		// Count the number of files in the directory
		var numFiles int
		filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				numFiles++
			}
			return nil
		})

		if numFiles != tc.expectedNumFiles {
			t.Fatalf("While extracting %s, expected to find %d file(s), but found %d. Local path = %s", tc.localFilePath, tc.expectedNumFiles, numFiles, tempDir)
		}

		// Ensure that files declared to be non-empty are in fact non-empty
		filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
			relativeFilename := strings.TrimPrefix(path, tempDir)

			if !info.IsDir() && slices.Contains(tc.nonemptyFiles, relativeFilename) {
				if info.Size() == 0 {
					t.Fatalf("Expected %s in %s to have non-zero file size, but found file size = %d.\n", relativeFilename, tc.localFilePath, info.Size())
				}
			}
			return nil
		})

	}
}

func TestExtractFilesExtractFile(t *testing.T) {
	// Create a temp directory
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %s", err)
	}
	defer os.RemoveAll(tempDir)

	zipFilePath := "test-fixtures/fetch-test-public-0.0.4.zip"
	filePathToExtract := "zzz.txt"
	localFileName := "/localzzz.txt"
	expectedFileCount := 1
	localPathName := filepath.Join(tempDir, localFileName)
	fileCount, err := extractFiles(zipFilePath, filePathToExtract, localPathName)

	if err != nil {
		t.Fatalf("Failed to extract files: %s", err)
	}

	if fileCount != expectedFileCount {
		t.Fatalf("Expected to extract %d files, extracted %d instead", expectedFileCount, fileCount)
	}

	filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		relativeFilename := strings.TrimPrefix(path, tempDir)

		if !info.IsDir() {
			if relativeFilename != localFileName {
				t.Fatalf("Expected local file %s to be created, but not found.\n", localFileName)
			}
		}
		return nil
	})
}
