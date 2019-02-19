package main

import (
	"testing"
)

// Expect to download 2 assets:
// - health-checker_linux_386
// - health-checker_linux_amd64
const SAMPLE_RELEASE_ASSET_REGEX = "health-checker_linux_[a-z0-9]+"

func TestDownloadReleaseAssets(t *testing.T) {
	tmpDir := mkTempDir(t)
	testInst := GitHubInstance{
		BaseUrl: "github.com",
		ApiUrl:  "api.github.com",
	}

	githubRepo, err := ParseUrlIntoGitHubRepo(SAMPLE_RELEASE_ASSET_GITHUB_REPO_URL, "", testInst)
	if err != nil {
		t.Fatalf("Failed to parse sample release asset GitHub URL into Fetch GitHubRepo struct: %s", err)
	}

	assetPaths, fetchErr := downloadReleaseAssets(SAMPLE_RELEASE_ASSET_REGEX, tmpDir, githubRepo, SAMPLE_RELEASE_ASSET_VERSION)
	if fetchErr != nil {
		t.Fatalf("Failed to download release asset: %s", fetchErr)
	}

	if len(assetPaths) != 2 {
		t.Fatalf("Expected to download 2 assets, not %d", len(assetPaths))
	}
}

func TestInvalidReleaseAssetsRegex(t *testing.T) {
	tmpDir := mkTempDir(t)
	testInst := GitHubInstance{
		BaseUrl: "github.com",
		ApiUrl:  "api.github.com",
	}

	githubRepo, err := ParseUrlIntoGitHubRepo(SAMPLE_RELEASE_ASSET_GITHUB_REPO_URL, "", testInst)
	if err != nil {
		t.Fatalf("Failed to parse sample release asset GitHub URL into Fetch GitHubRepo struct: %s", err)
	}

	_, fetchErr := downloadReleaseAssets("*", tmpDir, githubRepo, SAMPLE_RELEASE_ASSET_VERSION)
	if fetchErr == nil {
		t.Fatalf("Expected error for invalid regex")
	}
}
