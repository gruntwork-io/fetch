package main

import (
	"os"
	"testing"
	"path/filepath"
	"io/ioutil"
	"fmt"
	"strings"
	"github.com/stretchr/testify/assert"
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
		gitHubCommit := GitHubCommit{
			Repo: GitHubRepo{
				Owner: tc.repoOwner,
				Name: tc.repoName,
			},
			GitTag: tc.gitTag,
		}

		zipFilePath, err := downloadGithubZipFile(gitHubCommit, tc.githubToken)
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

	cases := []struct {
		repoOwner   string
		repoName    string
		branchName  string
		githubToken string
	}{
		{"gruntwork-io", "fetch-test-public", "sample-branch", ""},
		{"gruntwork-io", "fetch-test-private", "sample-branch", os.Getenv("GITHUB_OAUTH_TOKEN")},
	}

	for _, tc := range cases {
		gitHubCommit := GitHubCommit{
			Repo: GitHubRepo{
				Owner: tc.repoOwner,
				Name: tc.repoName,
			},
			BranchName: tc.branchName,
		}

		zipFilePath, err := downloadGithubZipFile(gitHubCommit, tc.githubToken)
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

	cases := []struct {
		repoOwner   string
		repoName    string
		branchName  string
		githubToken string
	}{
		{"gruntwork-io", "fetch-test-public", "branch-that-doesnt-exist", ""},
	}

	for _, tc := range cases {
		gitHubCommit := GitHubCommit{
			Repo: GitHubRepo{
				Owner: tc.repoOwner,
				Name: tc.repoName,
			},
			BranchName: tc.branchName,
		}

		zipFilePath, err := downloadGithubZipFile(gitHubCommit, tc.githubToken)
		defer os.RemoveAll(zipFilePath)
		if err == nil {
			t.Fatalf("Expected that attempt to download repo %s/%s for branch \"%s\" would fail, but received no error.", tc.repoOwner, tc.repoName, tc.branchName)
		}
	}
}

func TestDownloadGitCommitFile(t *testing.T) {
	t.Parallel()

	cases := []struct {
		repoOwner   string
		repoName    string
		commitSha   string
		githubToken string
	}{
		{"gruntwork-io", "fetch-test-public", "d2de34edb4c6564e0674b3f390b3b1fb0468183a", ""},
		{"gruntwork-io", "fetch-test-public", "57752e7f1df0acbd3c1e61545d5c4d0e87699d84", ""},
		{"gruntwork-io", "fetch-test-public", "f32a08313e30f116a1f5617b8b68c11f1c1dbb61", ""},
		{"gruntwork-io", "fetch-test-private", "676cfb92b54d33538c756c7a9479bfc3f6b44de2", os.Getenv("GITHUB_OAUTH_TOKEN")},
	}

	for _, tc := range cases {
		gitHubCommit := GitHubCommit{
			Repo: GitHubRepo{
				Owner: tc.repoOwner,
				Name: tc.repoName,
			},
			CommitSha: tc.commitSha,
		}

		zipFilePath, err := downloadGithubZipFile(gitHubCommit, tc.githubToken)
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

	cases := []struct {
		repoOwner   string
		repoName    string
		commitSha   string
		githubToken string
	}{
		{"gruntwork-io", "fetch-test-public", "hello-world", ""},
		{"gruntwork-io", "fetch-test-public", "i-am-a-non-existent-commit", ""},
		// remove a single letter from the beginning of an otherwise legit commit sha
		// interestingly, through testing I found that GitHub will attempt to find the right commit sha if you
		// truncate the end of it.
		{"gruntwork-io", "fetch-test-public", "7752e7f1df0acbd3c1e61545d5c4d0e87699d84", ""},
	}

	for _, tc := range cases {
		gitHubCommit := GitHubCommit{
			Repo: GitHubRepo{
				Owner: tc.repoOwner,
				Name: tc.repoName,
			},
			CommitSha: tc.commitSha,
		}

		zipFilePath, err := downloadGithubZipFile(gitHubCommit, tc.githubToken)
		defer os.RemoveAll(zipFilePath)
		if err == nil {
			t.Fatalf("Expected that attempt to download repo %s/%s at commmit sha \"%s\" would fail, but received no error.", tc.repoOwner, tc.repoName, tc.commitSha)
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
		gitHubCommit := GitHubCommit{
			Repo: GitHubRepo{
				Owner: tc.repoOwner,
				Name: tc.repoName,
			},
			GitTag: tc.gitTag,
		}

		_, err := downloadGithubZipFile(gitHubCommit, tc.githubToken)
		if err == nil && err.errorCode != 500 {
			t.Fatalf("Expected error for bad repo values: %s/%s:%s", tc.repoOwner, tc.repoName, tc.gitTag)
		}
	}
}

func TestDownloadSymlink(t *testing.T) {
	repoOwner := "gruntwork-io"
	repoName := "fetch-test-public"
	commitSha := "7549f0d4fb54782697beed421647dfa9f7c90d7f"

	tmpDir := mkTmpDir(t)
	fmt.Printf("tmpDir = %s\n", tmpDir)
	gitHubCommit := newGitHubCommit(repoOwner, repoName, commitSha)

	zipFilePath := downloadZipFile(t, gitHubCommit, "")
	defer os.RemoveAll(zipFilePath)

	extractZipFile(t, zipFilePath, "/", tmpDir)

	expectedFile1Path := filepath.Join(tmpDir, "file3.txt")
	assert.True(t, fileExists(expectedFile1Path), "Expected file to exist at %s", expectedFile1Path)

	expectedFile1Contents := readFileContents(t, expectedFile1Path)
	assert.Equal(t, "hello, world!\n", expectedFile1Contents, "Expected file contents to match.")

	expectedFile2Path := filepath.Join(tmpDir, "symlinked-folder", "file1.txt")
	expectedFile3Path := filepath.Join(tmpDir, "symlinked-folder", "file2.txt")
	expectedFile4Path := filepath.Join(tmpDir, "symlinked-folder", "file3.txt")

	assert.True(t, fileExists(expectedFile2Path), "Expected file to exist at %s", expectedFile2Path)
	assert.True(t, fileExists(expectedFile3Path), "Expected file to exist at %s", expectedFile3Path)
	assert.True(t, fileExists(expectedFile4Path), "Expected file to exist at %s", expectedFile4Path)

	expectedFile2Contents := readFileContents(t, expectedFile2Path)
	expectedFile3Contents := readFileContents(t, expectedFile3Path)
	expectedFile4Contents := readFileContents(t, expectedFile4Path)

	assert.Equal(t, "", expectedFile2Contents, "Expected file contents to match.")
	assert.Equal(t, "", expectedFile3Contents, "Expected file contents to match.")
	assert.Equal(t, "hello, world!\n", expectedFile4Contents, "Expected file contents to match.")
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
		{"test-fixtures/fetch-test-public-0.0.3.zip", "/", 4, []string{"/README.md"} },
		{"test-fixtures/fetch-test-public-0.0.3.zip", "/folder", 2, nil},
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

func mkTmpDir(t *testing.T) string {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	return tmpDir
}

func newGitHubCommit(repoOwner, repoName, commitSha string) *GitHubCommit {
	gitHubCommit := &GitHubCommit{
		Repo: GitHubRepo{
			Owner: repoOwner,
			Name: repoName,
		},
		CommitSha: commitSha,
	}

	return gitHubCommit
}

// Reminder: Make sure to call "defer os.RemoveAll(zipFilePath)" after calling this function
func downloadZipFile(t *testing.T, gitHubCommit *GitHubCommit, gitHubToken string) string {
	zipFilePath, err := downloadGithubZipFile(*gitHubCommit, "")
	if err != nil {
		t.Fatalf("Failed to download repo %s/%s at commmit sha \"%s\": %s", gitHubCommit.Repo.Owner, gitHubCommit.Repo.Name, gitHubCommit.CommitSha, err)
	}

	return zipFilePath
}

func extractZipFile(t *testing.T, zipFilePath string, filesToExtractFromZipPath string, localPath string) {
	err := extractFiles(zipFilePath, filesToExtractFromZipPath, localPath)
	if err != nil {
		t.Fatalf("Failed to extract files: %s", err)
	}
}

func readFileContents(t *testing.T, path string) string {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file contents")
	}

	return string(bytes)
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
