package github

import (
	"os"
	"testing"

	"github.com/gruntwork-io/fetch/source"
	"github.com/stretchr/testify/require"
)

func TestParseUrl(t *testing.T) {
	t.Parallel()

	config := source.Config{
		ApiVersion: "v3",
		Logger:     nil,
	}

	src := NewGitHubSource(config)

	cases := []struct {
		repoUrl string
		owner   string
		name    string
		token   string
	}{
		{"https://github.com/brikis98/ping-play", "brikis98", "ping-play", ""},
		{"http://github.com/brikis98/ping-play", "brikis98", "ping-play", ""},
		{"https://github.com/gruntwork-io/script-modules", "gruntwork-io", "script-modules", ""},
		{"http://github.com/gruntwork-io/script-modules", "gruntwork-io", "script-modules", ""},
		{"http://www.github.com/gruntwork-io/script-modules", "gruntwork-io", "script-modules", ""},
		{"http://www.github.com/gruntwork-io/script-modules/", "gruntwork-io", "script-modules", ""},
		{"http://www.github.com/gruntwork-io/script-modules?foo=bar", "gruntwork-io", "script-modules", "token"},
		{"http://www.github.com/gruntwork-io/script-modules?foo=bar&foo=baz", "gruntwork-io", "script-modules", "token"},
	}

	for _, tc := range cases {
		repo, err := src.ParseUrl(tc.repoUrl, tc.token)
		if err != nil {
			t.Fatalf("error extracting url %s into a Repo struct: %s", tc.repoUrl, err)
		}

		if repo.Owner != tc.owner {
			t.Fatalf("while extracting %s, expected owner %s, received %s", tc.repoUrl, tc.owner, repo.Owner)
		}

		if repo.Name != tc.name {
			t.Fatalf("while extracting %s, expected name %s, received %s", tc.repoUrl, tc.name, repo.Name)
		}

		if repo.Url != tc.repoUrl {
			t.Fatalf("while extracting %s, expected url %s, received %s", tc.repoUrl, tc.repoUrl, repo.Url)
		}

		if repo.Token != tc.token {
			t.Fatalf("while extracting %s, expected token %s, received %s", tc.repoUrl, tc.token, repo.Token)
		}
	}
}

func TestParseUrlMalformed(t *testing.T) {
	t.Parallel()

	config := source.Config{
		ApiVersion: "v3",
		Logger:     nil,
	}

	src := NewGitHubSource(config)

	cases := []struct {
		repoUrl     string
		description string
	}{
		// Note: https://githubb.com/... is now valid because custom domains are
		// treated as GitHub Enterprise instances (user can use --source flag)
		{"github.com/brikis98/ping-play", "Missing protocol"},
		{"curl://github.com/brikis98/ping-play", "Invalid protocol"},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			_, err := src.ParseUrl(tc.repoUrl, "")
			if err == nil {
				t.Fatalf("Expected error on malformed url %s, but no error was received.", tc.repoUrl)
			}
		})
	}
}

func TestParseUrlGitHubEnterprise(t *testing.T) {
	t.Parallel()

	config := source.Config{
		ApiVersion: "v3",
		Logger:     nil,
	}

	src := NewGitHubSource(config)

	// GitHub Enterprise URLs should work
	cases := []struct {
		repoUrl string
		owner   string
		name    string
		apiUrl  string
	}{
		{"https://ghe.mycompany.com/team/project", "team", "project", "ghe.mycompany.com/api/v3"},
		{"https://github.internal.com/org/repo", "org", "repo", "github.internal.com/api/v3"},
	}

	for _, tc := range cases {
		repo, err := src.ParseUrl(tc.repoUrl, "")
		require.NoError(t, err)

		if repo.Owner != tc.owner {
			t.Fatalf("expected owner %s, got %s", tc.owner, repo.Owner)
		}
		if repo.Name != tc.name {
			t.Fatalf("expected name %s, got %s", tc.name, repo.Name)
		}
		if repo.ApiUrl != tc.apiUrl {
			t.Fatalf("expected apiUrl %s, got %s", tc.apiUrl, repo.ApiUrl)
		}
	}
}

func TestType(t *testing.T) {
	config := source.Config{
		ApiVersion: "v3",
		Logger:     nil,
	}

	src := NewGitHubSource(config)
	if src.Type() != source.TypeGitHub {
		t.Fatalf("expected type %s, got %s", source.TypeGitHub, src.Type())
	}
}

// Integration tests below require GITHUB_OAUTH_TOKEN

func TestFetchTags(t *testing.T) {
	if os.Getenv("GITHUB_OAUTH_TOKEN") == "" {
		t.Skip("Skipping integration test - GITHUB_OAUTH_TOKEN not set")
	}
	t.Parallel()

	config := source.Config{
		ApiVersion: "v3",
		Logger:     nil,
	}

	src := NewGitHubSource(config)

	cases := []struct {
		repoUrl          string
		firstReleaseTag  string
		lastReleaseTag   string
		expectedNumTags  int
		gitHubOAuthToken string
	}{
		// Test on a public repo whose sole purpose is to be a test fixture for this tool
		{"https://github.com/gruntwork-io/fetch-test-public", "v0.0.1", "v0.0.4", 4, ""},

		// Private repo equivalent
		{"https://github.com/gruntwork-io/fetch-test-private", "v0.0.2", "v0.0.2", 1, os.Getenv("GITHUB_OAUTH_TOKEN")},
	}

	for _, tc := range cases {
		releases, err := src.FetchTags(tc.repoUrl, tc.gitHubOAuthToken)
		if err != nil {
			t.Fatalf("error fetching releases: %s", err)
		}

		if len(releases) != tc.expectedNumTags {
			t.Fatalf("expected %d releases, but got %d", tc.expectedNumTags, len(releases))
		}

		if releases[len(releases)-1] != tc.firstReleaseTag {
			t.Fatalf("expected first release = %s, actual = %s", tc.firstReleaseTag, releases[len(releases)-1])
		}

		if releases[0] != tc.lastReleaseTag {
			t.Fatalf("expected last release = %s, actual = %s", tc.lastReleaseTag, releases[0])
		}
	}
}

func TestGetReleaseInfo(t *testing.T) {
	if os.Getenv("GITHUB_OAUTH_TOKEN") == "" {
		t.Skip("Skipping integration test - GITHUB_OAUTH_TOKEN not set")
	}
	t.Parallel()

	token := os.Getenv("GITHUB_OAUTH_TOKEN")

	config := source.Config{
		ApiVersion: "v3",
		Logger:     nil,
	}

	src := NewGitHubSource(config)

	cases := []struct {
		repoUrl     string
		repoToken   string
		tag         string
		expectedId  int
		expectedUrl string
		assetCount  int
	}{
		{"https://github.com/gruntwork-io/fetch-test-private", token, "v0.0.2", 3064041, "https://api.github.com/repos/gruntwork-io/fetch-test-private/releases/3064041", 1},
		{"https://github.com/gruntwork-io/fetch-test-public", "", "v0.0.3", 3065803, "https://api.github.com/repos/gruntwork-io/fetch-test-public/releases/3065803", 0},
	}

	for _, tc := range cases {
		repo, err := src.ParseUrl(tc.repoUrl, tc.repoToken)
		if err != nil {
			t.Fatalf("Failed to parse %s: %s", tc.repoUrl, err.Error())
		}

		resp, err := src.GetReleaseInfo(repo, tc.tag)
		if err != nil {
			t.Fatalf("Failed to fetch release info for %s: %s", tc.repoUrl, err.Error())
		}

		if resp.Id != tc.expectedId {
			t.Fatalf("Expected release ID %d but got %d", tc.expectedId, resp.Id)
		}

		if resp.Url != tc.expectedUrl {
			t.Fatalf("Expected release URL %s but got %s", tc.expectedUrl, resp.Url)
		}

		if len(resp.Assets) != tc.assetCount {
			t.Fatalf("Expected %d assets but got %d", tc.assetCount, len(resp.Assets))
		}
	}
}

func TestDownloadReleaseAsset(t *testing.T) {
	if os.Getenv("GITHUB_OAUTH_TOKEN") == "" {
		t.Skip("Skipping integration test - GITHUB_OAUTH_TOKEN not set")
	}
	t.Parallel()

	token := os.Getenv("GITHUB_OAUTH_TOKEN")

	config := source.Config{
		ApiVersion: "v3",
		Logger:     nil,
	}

	src := NewGitHubSource(config)

	cases := []struct {
		repoUrl   string
		repoToken string
		tag       string
		assetId   int
		assetName string
		progress  bool
	}{
		{"https://github.com/gruntwork-io/fetch-test-private", token, "v0.0.2", 1872521, "test-asset.png", false},
		{"https://github.com/gruntwork-io/fetch-test-public", "", "v0.0.2", 1872641, "test-asset.png", false},
	}

	for _, tc := range cases {
		repo, err := src.ParseUrl(tc.repoUrl, tc.repoToken)
		if err != nil {
			t.Fatalf("Failed to parse %s: %s", tc.repoUrl, err.Error())
		}

		tmpFile, tmpErr := os.CreateTemp("", "test-download-release-asset")
		if tmpErr != nil {
			t.Fatalf("Failed to create temp file: %s", tmpErr.Error())
		}
		defer os.Remove(tmpFile.Name())

		asset := source.ReleaseAsset{
			Id:   tc.assetId,
			Name: tc.assetName,
		}

		if err := src.DownloadReleaseAsset(repo, asset, tmpFile.Name(), tc.progress); err != nil {
			t.Fatalf("Failed to download asset %d: %s", tc.assetId, err.Error())
		}

		if _, err := os.Stat(tmpFile.Name()); os.IsNotExist(err) {
			t.Fatalf("Downloaded file does not exist at %s", tmpFile.Name())
		}
	}
}
