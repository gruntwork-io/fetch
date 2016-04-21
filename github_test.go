package main

import (
	"testing"
	"os"
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
	}{
		{"https://github.com/brikis98/ping-play", "brikis98", "ping-play"},
		{"http://github.com/brikis98/ping-play", "brikis98", "ping-play"},
		{"https://github.com/gruntwork-io/script-modules", "gruntwork-io", "script-modules"},
		{"http://github.com/gruntwork-io/script-modules", "gruntwork-io", "script-modules"},
		{"http://www.github.com/gruntwork-io/script-modules", "gruntwork-io", "script-modules"},
		{"http://www.github.com/gruntwork-io/script-modules/", "gruntwork-io", "script-modules"},
		{"http://www.github.com/gruntwork-io/script-modules?foo=bar", "gruntwork-io", "script-modules"},
		{"http://www.github.com/gruntwork-io/script-modules?foo=bar&foo=baz", "gruntwork-io", "script-modules"},
	}

	for _, tc := range cases {
		repo, err := ParseUrlIntoGitHubRepo(tc.repoUrl)
		if err != nil {
			t.Fatalf("error extracting url %s into a GitHubRepo struct: %s", tc.repoUrl, err)
		}

		if repo.Owner != tc.owner {
			t.Fatalf("while extracting %s, expected owner %s, received %s", tc.repoUrl, tc.owner, repo.Owner)
		}

		if repo.Name != tc.name {
			t.Fatalf("while extracting %s, expected name %s, received %s", tc.repoUrl, tc.name, repo.Name)
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
		_, err := ParseUrlIntoGitHubRepo(tc.repoUrl)
		if err == nil {
			t.Fatalf("Expected error on malformed url %s, but no error was received.", tc.repoUrl)
		}
	}
}