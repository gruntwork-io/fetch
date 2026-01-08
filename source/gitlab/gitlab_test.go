package gitlab

import (
	"os"
	"testing"

	"github.com/gruntwork-io/fetch/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUrl(t *testing.T) {
	t.Parallel()

	config := source.Config{
		ApiVersion: "v4",
		Logger:     nil,
	}

	src := NewGitLabSource(config)

	cases := []struct {
		repoUrl string
		owner   string
		name    string
		token   string
	}{
		{"https://gitlab.com/owner/project", "owner", "project", ""},
		{"http://gitlab.com/owner/project", "owner", "project", ""},
		{"https://www.gitlab.com/owner/project", "owner", "project", ""},
		{"https://gitlab.com/owner/project/", "owner", "project", ""},
		{"https://gitlab.com/owner/project.git", "owner", "project", ""},
		{"https://gitlab.com/owner/project?foo=bar", "owner", "project", "token"},
	}

	for _, tc := range cases {
		t.Run(tc.repoUrl, func(t *testing.T) {
			repo, err := src.ParseUrl(tc.repoUrl, tc.token)
			require.NoError(t, err)

			assert.Equal(t, tc.owner, repo.Owner)
			assert.Equal(t, tc.name, repo.Name)
			assert.Equal(t, tc.repoUrl, repo.Url)
			assert.Equal(t, tc.token, repo.Token)
		})
	}
}

func TestParseUrlNestedGroups(t *testing.T) {
	t.Parallel()

	config := source.Config{
		ApiVersion: "v4",
		Logger:     nil,
	}

	src := NewGitLabSource(config)

	// GitLab supports nested groups (subgroups)
	cases := []struct {
		repoUrl string
		owner   string
		name    string
	}{
		{"https://gitlab.com/group/subgroup/project", "group/subgroup", "project"},
		{"https://gitlab.com/a/b/c/project", "a/b/c", "project"},
		{"https://gitlab.com/org/team/service/api", "org/team/service", "api"},
	}

	for _, tc := range cases {
		t.Run(tc.repoUrl, func(t *testing.T) {
			repo, err := src.ParseUrl(tc.repoUrl, "")
			require.NoError(t, err)

			assert.Equal(t, tc.owner, repo.Owner, "owner/namespace mismatch")
			assert.Equal(t, tc.name, repo.Name, "project name mismatch")
		})
	}
}

func TestParseUrlMalformed(t *testing.T) {
	t.Parallel()

	config := source.Config{
		ApiVersion: "v4",
		Logger:     nil,
	}

	src := NewGitLabSource(config)

	cases := []struct {
		repoUrl     string
		description string
	}{
		{"https://gitlab.com/project", "Missing owner (only one path segment)"},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			_, err := src.ParseUrl(tc.repoUrl, "")
			assert.Error(t, err)
		})
	}
}

func TestParseUrlWwwPrefix(t *testing.T) {
	t.Parallel()

	config := source.Config{
		ApiVersion: "v4",
		Logger:     nil,
	}

	src := NewGitLabSource(config)

	// www.gitlab.com URLs should have www. stripped from apiUrl
	cases := []struct {
		repoUrl string
		apiUrl  string
	}{
		{"https://www.gitlab.com/owner/project", "gitlab.com"},
		{"https://WWW.gitlab.com/owner/project", "gitlab.com"},
		{"https://gitlab.com/owner/project", "gitlab.com"},
	}

	for _, tc := range cases {
		t.Run(tc.repoUrl, func(t *testing.T) {
			repo, err := src.ParseUrl(tc.repoUrl, "")
			require.NoError(t, err)

			assert.Equal(t, tc.apiUrl, repo.ApiUrl, "apiUrl should have www. stripped")
		})
	}
}

func TestParseUrlSelfHosted(t *testing.T) {
	t.Parallel()

	config := source.Config{
		ApiVersion: "v4",
		Logger:     nil,
	}

	src := NewGitLabSource(config)

	// Self-hosted GitLab instances
	cases := []struct {
		repoUrl string
		owner   string
		name    string
		apiUrl  string
	}{
		{"https://git.company.com/team/project", "team", "project", "git.company.com"},
		{"https://gitlab.internal.io/org/repo", "org", "repo", "gitlab.internal.io"},
	}

	for _, tc := range cases {
		t.Run(tc.repoUrl, func(t *testing.T) {
			repo, err := src.ParseUrl(tc.repoUrl, "")
			require.NoError(t, err)

			assert.Equal(t, tc.owner, repo.Owner)
			assert.Equal(t, tc.name, repo.Name)
			assert.Equal(t, tc.apiUrl, repo.ApiUrl)
		})
	}
}

func TestType(t *testing.T) {
	config := source.Config{
		ApiVersion: "v4",
		Logger:     nil,
	}

	src := NewGitLabSource(config)
	assert.Equal(t, source.TypeGitLab, src.Type())
}

func TestEncodeProjectPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		owner    string
		name     string
		expected string
	}{
		{"owner", "project", "owner%2Fproject"},
		{"group/subgroup", "project", "group%2Fsubgroup%2Fproject"},
		{"a/b/c", "repo", "a%2Fb%2Fc%2Frepo"},
	}

	for _, tc := range cases {
		t.Run(tc.owner+"/"+tc.name, func(t *testing.T) {
			encoded := encodeProjectPath(tc.owner, tc.name)
			assert.Equal(t, tc.expected, encoded)
		})
	}
}

// Integration tests below - these make real HTTP calls to GitLab

func TestFetchTagsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	t.Parallel()

	config := source.Config{
		ApiVersion: "v4",
		Logger:     nil,
	}

	src := NewGitLabSource(config)

	// Test on a public GitLab repo (gitlab-runner is well-known and stable)
	repoUrl := "https://gitlab.com/gitlab-org/gitlab-runner"

	tags, err := src.FetchTags(repoUrl, "")
	require.NoError(t, err, "FetchTags should succeed for public repo")

	// gitlab-runner has many releases, should have at least 10 tags
	assert.GreaterOrEqual(t, len(tags), 10, "Should have multiple tags")

	// Check that we get semver-like tags (v17.x.x, v16.x.x, etc.)
	foundValidTag := false
	for _, tag := range tags {
		if len(tag) > 0 && (tag[0] == 'v' || tag[0] >= '0' && tag[0] <= '9') {
			foundValidTag = true
			break
		}
	}
	assert.True(t, foundValidTag, "Should find at least one semver tag")
}

func TestGetReleaseInfoIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	t.Parallel()

	config := source.Config{
		ApiVersion: "v4",
		Logger:     nil,
	}

	src := NewGitLabSource(config)

	// Parse the gitlab-runner URL
	repo, err := src.ParseUrl("https://gitlab.com/gitlab-org/gitlab-runner", "")
	require.NoError(t, err)

	// Get release info for a known tag
	release, err := src.GetReleaseInfo(repo, "v17.0.0")
	require.NoError(t, err, "GetReleaseInfo should succeed for public release")

	// Check release has assets
	assert.NotEmpty(t, release.Assets, "Release should have assets")

	// Check that we can find the Linux amd64 binary
	foundLinuxBinary := false
	for _, asset := range release.Assets {
		if asset.Name == "gitlab-runner-linux-amd64" {
			foundLinuxBinary = true
			assert.NotEmpty(t, asset.Url, "Asset should have URL")
			break
		}
	}
	assert.True(t, foundLinuxBinary, "Should find gitlab-runner-linux-amd64 asset")
}

func TestDownloadReleaseAssetIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	// Note: Not parallel since it downloads a large file

	config := source.Config{
		ApiVersion: "v4",
		Logger:     nil,
	}

	src := NewGitLabSource(config)

	// Parse the gitlab-runner URL
	repo, err := src.ParseUrl("https://gitlab.com/gitlab-org/gitlab-runner", "")
	require.NoError(t, err)

	// Get release info
	release, err := src.GetReleaseInfo(repo, "v17.0.0")
	require.NoError(t, err)

	// Find a small asset to download (checksums file is small)
	var checksumAsset source.ReleaseAsset
	for _, asset := range release.Assets {
		if asset.Name == "release.sha256" {
			checksumAsset = asset
			break
		}
	}
	require.NotEmpty(t, checksumAsset.Name, "Should find release.sha256 asset")

	// Create temp file
	tmpFile, err := os.CreateTemp("", "gitlab-test-download-*")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Download the asset
	err = src.DownloadReleaseAsset(repo, checksumAsset, tmpFile.Name(), false)
	require.NoError(t, err, "DownloadReleaseAsset should succeed")

	// Verify file exists and has content
	info, err := os.Stat(tmpFile.Name())
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0), "Downloaded file should have content")
}
