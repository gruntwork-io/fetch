package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/gruntwork-io/fetch/source"
	_ "github.com/gruntwork-io/fetch/source/github"
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

func mkTempDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %s", err)
	}

	return tmpDir
}
