package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/gruntwork-io/fetch/source"
	_ "github.com/gruntwork-io/fetch/source/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Expect to download 2 assets:
// - health-checker_linux_386
// - health-checker_linux_amd64
const SAMPLE_RELEASE_ASSET_REGEX = "health-checker_linux_[a-z0-9]+"

func TestDownloadReleaseAssets(t *testing.T) {
	tmpDir := mkTempDir(t)
	defer os.RemoveAll(tmpDir)
	logger := GetProjectLogger()

	config := source.Config{
		ApiVersion: "v3",
		Logger:     logger,
	}
	src, err := source.NewSource(source.TypeGitHub, config)
	require.NoError(t, err)

	repo, err := src.ParseUrl(SAMPLE_RELEASE_ASSET_GITHUB_REPO_URL, "")
	require.NoError(t, err)

	assetPaths, fetchErr := downloadReleaseAssetsWithSource(logger, src, SAMPLE_RELEASE_ASSET_REGEX, tmpDir, repo, SAMPLE_RELEASE_ASSET_VERSION, false)
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
	defer os.RemoveAll(tmpDir)
	logger := GetProjectLogger()

	config := source.Config{
		ApiVersion: "v3",
		Logger:     logger,
	}
	src, err := source.NewSource(source.TypeGitHub, config)
	require.NoError(t, err)

	const githubRepoUrl = "https://github.com/gruntwork-io/fetch-test-public"
	const releaseAsset = "hello+world.txt"
	const assetVersion = "v0.0.4"

	repo, err := src.ParseUrl(githubRepoUrl, "")
	require.NoError(t, err)

	assetPaths, fetchErr := downloadReleaseAssetsWithSource(logger, src, releaseAsset, tmpDir, repo, assetVersion, false)
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
	defer os.RemoveAll(tmpDir)
	logger := GetProjectLogger()

	config := source.Config{
		ApiVersion: "v3",
		Logger:     logger,
	}
	src, err := source.NewSource(source.TypeGitHub, config)
	require.NoError(t, err)

	repo, err := src.ParseUrl(SAMPLE_RELEASE_ASSET_GITHUB_REPO_URL, "")
	require.NoError(t, err)

	_, fetchErr := downloadReleaseAssetsWithSource(logger, src, "*", tmpDir, repo, SAMPLE_RELEASE_ASSET_VERSION, false)
	if fetchErr == nil {
		t.Fatalf("Expected error for invalid regex")
	}
}

func TestInvalidReleaseAssetTag(t *testing.T) {
	tmpDir := mkTempDir(t)
	defer os.RemoveAll(tmpDir)
	logger := GetProjectLogger()

	config := source.Config{
		ApiVersion: "v3",
		Logger:     logger,
	}
	src, err := source.NewSource(source.TypeGitHub, config)
	require.NoError(t, err)

	repo, err := src.ParseUrl(SAMPLE_RELEASE_ASSET_GITHUB_REPO_URL, "")
	require.NoError(t, err)

	_, fetchErr := downloadReleaseAssetsWithSource(logger, src, SAMPLE_RELEASE_ASSET_REGEX, tmpDir, repo, "6.6.6", false)
	assert.Error(t, fetchErr)
}
