package main

import (
	"testing"
	"os"
	"reflect"
	"io/ioutil"
)

func TestGetListOfReleasesFromGitHubRepo(t *testing.T) {
	t.Parallel()

	cases := []struct {
		repoUrl          string
		firstReleaseTag  string
		lastReleaseTag   string
		gitHubOAuthToken string
	}{
		// Test on a public repo whose sole purpose is to be a test fixture for this tool
		{"https://github.com/gruntwork-io/fetch-test-public", "v0.0.1", "v0.0.3", ""},

		// Private repo equivalent
		{"https://github.com/gruntwork-io/fetch-test-private", "v0.0.2", "v0.0.2", os.Getenv("GITHUB_OAUTH_TOKEN")},
	}

	for _, tc := range cases {
		releases, err := FetchTags(tc.repoUrl, tc.gitHubOAuthToken)
		if err != nil {
			t.Fatalf("error fetching releases: %s", err)
		}

		if len(releases) != 0 && tc.firstReleaseTag == "" {
			t.Fatalf("expected empty list of releases for repo %s, but got first release = %s", tc.repoUrl, releases[0])
		}

		if len(releases) == 0 && tc.firstReleaseTag != "" {
			t.Fatalf("expected non-empty list of releases for repo %s, but no releases were found", tc.repoUrl)
		}

		if releases[len(releases) - 1] != tc.firstReleaseTag {
			t.Fatalf("error parsing github releases for repo %s. expected first release = %s, actual = %s", tc.repoUrl, tc.firstReleaseTag, releases[len(releases) - 1])
		}

		if releases[0] != tc.lastReleaseTag {
			t.Fatalf("error parsing github releases for repo %s. expected first release = %s, actual = %s", tc.repoUrl, tc.lastReleaseTag, releases[0])
		}
	}
}

func TestParseUrlIntoGitHubRepo(t *testing.T) {
	t.Parallel()

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
		repo, err := ParseUrlIntoGitHubRepo(tc.repoUrl, tc.token)
		if err != nil {
			t.Fatalf("error extracting url %s into a GitHubRepo struct: %s", tc.repoUrl, err)
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

	cases := []struct {
		repoUrl string
	}{
		{"https://githubb.com/brikis98/ping-play"},
		{"github.com/brikis98/ping-play"},
		{"curl://github.com/brikis98/ping-play"},
	}

	for _, tc := range cases {
		_, err := ParseUrlIntoGitHubRepo(tc.repoUrl, "")
		if err == nil {
			t.Fatalf("Expected error on malformed url %s, but no error was received.", tc.repoUrl)
		}
	}
}

func TestGetGitHubReleaseInfo(t *testing.T) {
	t.Parallel()

	token := os.Getenv("GITHUB_OAUTH_TOKEN")

	expectedFetchTestPrivateRelease := GitHubReleaseApiResponse{
		Id: 3064041,
		Url: "https://api.github.com/repos/gruntwork-io/fetch-test-private/releases/3064041",
		Name: "v0.0.2",
		Assets: []GitHubReleaseAsset{
			{
				Id: 1872521,
				Url: "https://api.github.com/repos/gruntwork-io/fetch-test-private/releases/assets/1872521",
				Name: "test-asset.png",
			},
		},
	}

	expectedFetchTestPublicRelease := GitHubReleaseApiResponse{
		Id: 3065803,
		Url: "https://api.github.com/repos/gruntwork-io/fetch-test-public/releases/3065803",
		Name: "v0.0.3",
		Assets: []GitHubReleaseAsset{},
	}

	cases := []struct {
		repoUrl   string
		repoToken string
		tag       string
		expected  GitHubReleaseApiResponse
	}{
		{"https://github.com/gruntwork-io/fetch-test-private", token, "v0.0.2", expectedFetchTestPrivateRelease},
		{"https://github.com/gruntwork-io/fetch-test-public", "", "v0.0.3", expectedFetchTestPublicRelease},
	}

	for _, tc := range cases {
		repo, err := ParseUrlIntoGitHubRepo(tc.repoUrl, tc.repoToken)
		if err != nil {
			t.Fatalf("Failed to parse %s into GitHub URL due to error: %s", tc.repoUrl, err.Error())
		}

		resp, err := GetGitHubReleaseInfo(repo, tc.tag)
		if err != nil {
			t.Fatalf("Failed to fetch GitHub release info for repo %s due to error: %s", tc.repoToken, err.Error())
		}

		if !reflect.DeepEqual(tc.expected, resp) {
			t.Fatalf("Expected GitHub release %+v but got GitHub release %+v", tc.expected, resp)
		}
	}
}

func TestDownloadReleaseAsset(t *testing.T) {
	t.Parallel()

	token := os.Getenv("GITHUB_OAUTH_TOKEN")

	cases := []struct {
		repoUrl   string
		repoToken string
		tag       string
		assetId   int
	}{
		{"https://github.com/gruntwork-io/fetch-test-private", token, "v0.0.2", 1872521},
		{"https://github.com/gruntwork-io/fetch-test-public", "", "v0.0.2", 1872641},
	}

	for _, tc := range cases {
		repo, err := ParseUrlIntoGitHubRepo(tc.repoUrl, tc.repoToken)
		if err != nil {
			t.Fatalf("Failed to parse %s into GitHub URL due to error: %s", tc.repoUrl, err.Error())
		}

		tmpFile, tmpErr := ioutil.TempFile("", "test-download-release-asset")
		if tmpErr != nil {
			t.Fatalf("Failed to create temp file due to error: %s", tmpErr.Error())
		}

		if err := DownloadReleaseAsset(repo, tc.assetId, tmpFile.Name()); err != nil {
			t.Fatalf("Failed to download asset %d to %s from GitHub URL %s due to error: %s", tc.assetId, tmpFile.Name(), tc.repoUrl, err.Error())
		}

		defer os.Remove(tmpFile.Name())

		if !fileExists(tmpFile.Name()) {
			t.Fatalf("Got no errors downloading asset %d to %s from GitHub URL %s, but %s does not exist!", tc.assetId, tmpFile.Name(), tc.repoUrl, tmpFile.Name())
		}
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}