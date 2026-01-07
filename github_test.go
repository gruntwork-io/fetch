package main

import (
	"os"
	"testing"

	"github.com/gruntwork-io/fetch/source"
	_ "github.com/gruntwork-io/fetch/source/github"
	"github.com/stretchr/testify/require"
)

func TestGetListOfReleasesFromGitHubRepo(t *testing.T) {
	t.Parallel()

	config := source.Config{
		ApiVersion: "v3",
		Logger:     nil,
	}

	src, err := source.NewSource(source.SourceTypeGitHub, config)
	require.NoError(t, err)

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

		if len(releases) != 0 && tc.firstReleaseTag == "" {
			t.Fatalf("expected empty list of releases for repo %s, but got first release = %s", tc.repoUrl, releases[0])
		}

		if len(releases) == 0 && tc.firstReleaseTag != "" {
			t.Fatalf("expected non-empty list of releases for repo %s, but no releases were found", tc.repoUrl)
		}

		if len(releases) != tc.expectedNumTags {
			t.Fatalf("expected %d releases, but got %d", tc.expectedNumTags, len(releases))
		}

		if releases[len(releases)-1] != tc.firstReleaseTag {
			t.Fatalf("error parsing github releases for repo %s. expected first release = %s, actual = %s", tc.repoUrl, tc.firstReleaseTag, releases[len(releases)-1])
		}

		if releases[0] != tc.lastReleaseTag {
			t.Fatalf("error parsing github releases for repo %s. expected first release = %s, actual = %s", tc.repoUrl, tc.lastReleaseTag, releases[0])
		}
	}
}

func TestParseUrlIntoGitHubRepo(t *testing.T) {
	t.Parallel()

	config := source.Config{
		ApiVersion: "v3",
		Logger:     nil,
	}

	src, err := source.NewSource(source.SourceTypeGitHub, config)
	require.NoError(t, err)

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

func TestParseUrlThrowsErrorOnMalformedUrl(t *testing.T) {
	t.Parallel()

	config := source.Config{
		ApiVersion: "v3",
		Logger:     nil,
	}

	src, err := source.NewSource(source.SourceTypeGitHub, config)
	require.NoError(t, err)

	cases := []struct {
		repoUrl string
	}{
		// Note: https://githubb.com/... is now valid because custom domains are
		// treated as GitHub Enterprise instances (user can use --source flag)
		{"github.com/brikis98/ping-play"},        // Missing protocol
		{"curl://github.com/brikis98/ping-play"}, // Invalid protocol
	}

	for _, tc := range cases {
		_, err := src.ParseUrl(tc.repoUrl, "")
		if err == nil {
			t.Fatalf("Expected error on malformed url %s, but no error was received.", tc.repoUrl)
		}
	}
}

func TestGetGitHubReleaseInfo(t *testing.T) {
	t.Parallel()

	token := os.Getenv("GITHUB_OAUTH_TOKEN")

	config := source.Config{
		ApiVersion: "v3",
		Logger:     nil,
	}

	src, err := source.NewSource(source.SourceTypeGitHub, config)
	require.NoError(t, err)

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
			t.Fatalf("Failed to parse %s into Repo due to error: %s", tc.repoUrl, err.Error())
		}

		resp, err := src.GetReleaseInfo(repo, tc.tag)
		if err != nil {
			t.Fatalf("Failed to fetch release info for repo %s due to error: %s", tc.repoUrl, err.Error())
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

func TestDownloadGitHubReleaseAsset(t *testing.T) {
	t.Parallel()

	token := os.Getenv("GITHUB_OAUTH_TOKEN")

	config := source.Config{
		ApiVersion: "v3",
		Logger:     nil,
	}

	src, err := source.NewSource(source.SourceTypeGitHub, config)
	require.NoError(t, err)

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
		{"https://github.com/gruntwork-io/fetch-test-public", "", "v0.0.2", 1872641, "test-asset.png", true},
	}

	for _, tc := range cases {
		repo, err := src.ParseUrl(tc.repoUrl, tc.repoToken)
		if err != nil {
			t.Fatalf("Failed to parse %s into Repo due to error: %s", tc.repoUrl, err.Error())
		}

		tmpFile, tmpErr := os.CreateTemp("", "test-download-release-asset")
		if tmpErr != nil {
			t.Fatalf("Failed to create temp file due to error: %s", tmpErr.Error())
		}

		asset := source.ReleaseAsset{
			Id:   tc.assetId,
			Name: tc.assetName,
		}

		if err := src.DownloadReleaseAsset(repo, asset, tmpFile.Name(), tc.progress); err != nil {
			t.Fatalf("Failed to download asset %d to %s from URL %s due to error: %s", tc.assetId, tmpFile.Name(), tc.repoUrl, err.Error())
		}

		defer os.Remove(tmpFile.Name())

		if !fileExists(tmpFile.Name()) {
			t.Fatalf("Got no errors downloading asset %d to %s from URL %s, but %s does not exist!", tc.assetId, tmpFile.Name(), tc.repoUrl, tmpFile.Name())
		}
	}
}

func TestDetectSourceType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		repoUrl      string
		expectedType source.SourceType
	}{
		{"https://github.com/owner/repo", source.SourceTypeGitHub},
		{"https://www.github.com/owner/repo", source.SourceTypeGitHub},
		{"https://gitlab.com/owner/repo", source.SourceTypeGitLab},
		{"https://www.gitlab.com/owner/repo", source.SourceTypeGitLab},
		// Custom domains default to GitHub (user must use --source flag)
		{"https://git.company.com/owner/repo", source.SourceTypeGitHub},
		{"https://ghe.mycompany.com/owner/repo", source.SourceTypeGitHub},
	}

	for _, tc := range cases {
		detected, err := source.DetectSourceType(tc.repoUrl)
		require.NoError(t, err)
		if detected != tc.expectedType {
			t.Fatalf("Expected %s for URL %s but got %s", tc.expectedType, tc.repoUrl, detected)
		}
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
