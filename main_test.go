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

	repo, err := src.ParseUrl("https://github.com/gruntwork-io/fetch-test-public", "")
	require.NoError(t, err)

	_, fetchErr := downloadReleaseAssetsWithSource(logger, src, "*", tmpDir, repo, "v0.0.4", false)
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

	repo, err := src.ParseUrl("https://github.com/gruntwork-io/fetch-test-public", "")
	require.NoError(t, err)

	_, fetchErr := downloadReleaseAssetsWithSource(logger, src, "hello+world.txt", tmpDir, repo, "6.6.6", false)
	assert.Error(t, fetchErr)
}

func mkTempDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %s", err)
	}

	return tmpDir
}
