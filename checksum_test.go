package main

import (
	"os"
	"testing"

	"github.com/gruntwork-io/fetch/source"
	_ "github.com/gruntwork-io/fetch/source/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const SAMPLE_RELEASE_ASSET_GITHUB_REPO_URL = "https://github.com/gruntwork-io/fetch-test-public"
const SAMPLE_RELEASE_ASSET_VERSION = "v0.0.5"
const SAMPLE_RELEASE_ASSET_NAME = "sample_linux_amd64"

// Checksums can be computed by running "shasum -a [256|512] /path/to/file" on any UNIX system
const SAMPLE_RELEASE_ASSET_CHECKSUM_SHA256 = "d54d20eadbec9c9cc5ac6e0e371abc96d64a0d724d1cf45297fab6098026a69d"
const SAMPLE_RELEASE_ASSET_CHECKSUM_SHA512 = "1690ac6a33e8184c37abeb250efe218e93ad3c2f2c899b97b6767192075e7a7c26314279fe346a2581a6c10cf25b8e425eeb2f5593e773de56a222576904d72a"

var SAMPLE_RELEASE_ASSET_CHECKSUMS_SHA256 = map[string]bool{
	"d54d20eadbec9c9cc5ac6e0e371abc96d64a0d724d1cf45297fab6098026a69d": true,
	"3ad00987ec952f521f89cd901e4cf4fcf25100a3145613131a893a52bc78ee24": true,
}

var SAMPLE_RELEASE_ASSET_CHECKSUMS_SHA256_NO_MATCH = map[string]bool{
	"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX": true,
	"YYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYY": true,
}

func TestVerifyReleaseAsset(t *testing.T) {
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

	assetPaths, fetchErr := downloadReleaseAssetsWithSource(logger, src, SAMPLE_RELEASE_ASSET_NAME, tmpDir, repo, SAMPLE_RELEASE_ASSET_VERSION, false)
	if fetchErr != nil {
		t.Fatalf("Failed to download release asset: %s", fetchErr)
	}

	if len(assetPaths) != 1 {
		t.Fatalf("Incorrect number of release assets: %d", len(assetPaths))
	}

	checksumSha256, fetchErr := computeChecksum(assetPaths[0], "sha256")
	if fetchErr != nil {
		t.Fatalf("Failed to compute file checksum: %s", fetchErr)
	}

	checksumSha512, fetchErr := computeChecksum(assetPaths[0], "sha512")
	if fetchErr != nil {
		t.Fatalf("Failed to compute file checksum: %s", fetchErr)
	}

	assert.Equal(t, SAMPLE_RELEASE_ASSET_CHECKSUM_SHA256, checksumSha256, "SHA256 checksum of sample asset failed to match.")
	assert.Equal(t, SAMPLE_RELEASE_ASSET_CHECKSUM_SHA512, checksumSha512, "SHA512 checksum of sample asset failed to match.")
}

func TestVerifyChecksumOfReleaseAsset(t *testing.T) {
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
		t.Fatalf("Incorrect number of release assets: %d", len(assetPaths))
	}

	for _, assetPath := range assetPaths {
		checksumErr := verifyChecksumOfReleaseAsset(logger, assetPath, SAMPLE_RELEASE_ASSET_CHECKSUMS_SHA256, "sha256")
		if checksumErr != nil {
			t.Fatalf("Expected downloaded asset to match one of %d checksums: %s", len(SAMPLE_RELEASE_ASSET_CHECKSUMS_SHA256), checksumErr)
		}
	}

	for _, assetPath := range assetPaths {
		checksumErr := verifyChecksumOfReleaseAsset(logger, assetPath, SAMPLE_RELEASE_ASSET_CHECKSUMS_SHA256_NO_MATCH, "sha256")
		if checksumErr == nil {
			t.Fatalf("Expected downloaded asset to not match any checksums")
		}
	}
}

func mkTempDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %s", err)
	}

	return tmpDir
}
