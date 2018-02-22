package main

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
)

const SAMPLE_RELEASE_ASSET_GITHUB_REPO_URL ="https://github.com/gruntwork-io/health-checker"
const SAMPLE_RELEASE_ASSET_VERSION="v0.0.2"
const SAMPLE_RELEASE_ASSET_NAME="health-checker_linux_amd64"

// Checksums can be computed by running "shasum -a [256|512] /path/to/file" on any UNIX system
const SAMPLE_RELEASE_ASSET_CHECKSUM_SHA256="4314590d802760c29a532e2ef22689d4656d184b3daa63f96bc8b8f76f5d22f0"
const SAMPLE_RELEASE_ASSET_CHECKSUM_SHA512="28d9e487c1001e3c28d915c9edd3ed37632f10b923bd94d4d9ac6d28c0af659abbe2456da167763d51def2182fef01c3f73c67edf527d4ed1389a28ba10db332"

func TestVerifyReleaseAsset(t *testing.T) {
	tmpDir := mkTempDir(t)
	releaseAssets := []string{SAMPLE_RELEASE_ASSET_NAME}

	githubRepo, err := ParseUrlIntoGitHubRepo(SAMPLE_RELEASE_ASSET_GITHUB_REPO_URL, "")
	if err != nil {
		t.Fatalf("Failed to parse sample release asset GitHub URL into Fetch GitHubRepo struct: %s", err)
	}

	assetPaths, fetchErr := downloadReleaseAssets(releaseAssets, tmpDir, githubRepo, SAMPLE_RELEASE_ASSET_VERSION)
	if fetchErr != nil {
		t.Fatalf("Failed to download release asset: %s", fetchErr)
	}

	if len(assetPaths) > 1 {
		t.Fatalf("Downloaded more release assets than expected: %s", err)
	}

	assetPath := assetPaths[0]

	checksumSha256, fetchErr := computeChecksum(assetPath, "sha256")
	if fetchErr != nil {
		t.Fatalf("Failed to compute file checksum: %s", fetchErr)
	}

	checksumSha512, fetchErr := computeChecksum(assetPath, "sha512")
	if fetchErr != nil {
		t.Fatalf("Failed to compute file checksum: %s", fetchErr)
	}

	assert.Equal(t, SAMPLE_RELEASE_ASSET_CHECKSUM_SHA256, checksumSha256, "SHA256 checksum of sample asset failed to match.")
	assert.Equal(t, SAMPLE_RELEASE_ASSET_CHECKSUM_SHA512, checksumSha512, "SHA512 checksum of sample asset failed to match.")
}

func TestAllEqual(t *testing.T) {
	assert.True(t, isAllEqual())
	assert.True(t, isAllEqual(1))
	assert.True(t, isAllEqual(1, 1))
	assert.True(t, isAllEqual(1, 1, 1))
	assert.False(t, isAllEqual(0, 1))
	assert.False(t, isAllEqual(0, 1, 1))
	assert.False(t, isAllEqual(1, 0, 1))
	assert.False(t, isAllEqual(1, 0, 0))
	assert.False(t, isAllEqual(0, 9, 0, 0, 0))
	assert.False(t, isAllEqual(0, 9, 2, 0, 0, 0))
}

func mkTempDir(t *testing.T) string {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %s", err)
	}

	return tmpDir
}