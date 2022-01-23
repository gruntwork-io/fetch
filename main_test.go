package main

import (
	"fmt"
	"os"
	"testing"
)

// Expect to download 2 assets:
// - health-checker_linux_386
// - health-checker_linux_amd64
const SAMPLE_RELEASE_ASSET_REGEX = "health-checker_linux_[a-z0-9]+"

func TestDownloadReleaseAssets(t *testing.T) {
	tmpDir := mkTempDir(t)
	logger := GetProjectLogger()
	testInst := GitHubInstance{
		BaseUrl: "github.com",
		ApiUrl:  "api.github.com",
	}

	githubRepo, err := ParseUrlIntoGitHubRepo(SAMPLE_RELEASE_ASSET_GITHUB_REPO_URL, "", testInst)
	if err != nil {
		t.Fatalf("Failed to parse sample release asset GitHub URL into Fetch GitHubRepo struct: %s", err)
	}

	assetPaths, fetchErr := downloadReleaseAssets(logger, SAMPLE_RELEASE_ASSET_REGEX, tmpDir, githubRepo, SAMPLE_RELEASE_ASSET_VERSION, false)
	if fetchErr != nil {
		t.Fatalf("Failed to download release asset: %s", fetchErr)
	}

	if len(assetPaths) != 2 {
		t.Fatalf("Expected to download 2 assets, not %d", len(assetPaths))
	}

	for _, assetPath := range assetPaths {
		if _, err := os.Stat(assetPath); os.IsNotExist(err) {
			t.Fatalf("Downloaded file should exist at %s", assetPath)
		} else {
			fmt.Printf("Verified the downloaded asset exists at %s\n", assetPath)
		}
	}
}

func TestDownloadReleaseAssetsWithRegexCharacters(t *testing.T) {
	tmpDir := mkTempDir(t)
	logger := GetProjectLogger()
	testInst := GitHubInstance{
		BaseUrl: "github.com",
		ApiUrl:  "api.github.com",
	}

	const githubRepoUrl = "https://github.com/gruntwork-io/fetch-test-public"
	const releaseAsset = "hello+world.txt"
	const assetVersion = "v0.0.4"

	githubRepo, err := ParseUrlIntoGitHubRepo(githubRepoUrl, "", testInst)
	if err != nil {
		t.Fatalf("Failed to parse sample release asset GitHub URL into Fetch GitHubRepo struct: %s", err)
	}

	assetPaths, fetchErr := downloadReleaseAssets(logger, releaseAsset, tmpDir, githubRepo, assetVersion, false)
	if fetchErr != nil {
		t.Fatalf("Failed to download release asset: %s", fetchErr)
	}

	if len(assetPaths) != 1 {
		t.Fatalf("Expected to download 1 assets, not %d", len(assetPaths))
	}

	assetPath := assetPaths[0]

	if _, err := os.Stat(assetPath); os.IsNotExist(err) {
		t.Fatalf("Downloaded file should exist at %s", assetPath)
	} else {
		fmt.Printf("Verified the downloaded asset exists at %s\n", assetPath)
	}
}

func TestInvalidReleaseAssetsRegex(t *testing.T) {
	tmpDir := mkTempDir(t)
	logger := GetProjectLogger()
	testInst := GitHubInstance{
		BaseUrl: "github.com",
		ApiUrl:  "api.github.com",
	}

	githubRepo, err := ParseUrlIntoGitHubRepo(SAMPLE_RELEASE_ASSET_GITHUB_REPO_URL, "", testInst)
	if err != nil {
		t.Fatalf("Failed to parse sample release asset GitHub URL into Fetch GitHubRepo struct: %s", err)
	}

	_, fetchErr := downloadReleaseAssets(logger, "*", tmpDir, githubRepo, SAMPLE_RELEASE_ASSET_VERSION, false)
	if fetchErr == nil {
		t.Fatalf("Expected error for invalid regex")
	}
}
