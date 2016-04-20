package main

import (
	"testing"
)

func TestGetListOfReleasesFromGitHubRepo(t *testing.T) {
	cases := []struct {
		repoUrl string
		lastReleaseTag string
		firstReleaseTag string
	}{
		// These tests are obviously brittle. Suggestions welcome on a more durable way to test.
		//{"https://github.com/gruntwork-io/script-modules", "v0.0.19", "v0.0.1"}, // private repo, requires GitHub oAuth Token
		{"https://github.com/brikis98/ping-play", "v0.0.13", "v0.0.2"},
	}

	for _, tc := range cases {
		releases, err := FetchReleases(tc.repoUrl)
		if err != nil {
			t.Fatalf("error fetching releases: %s", err)
		}

		if len(releases) != 0 && tc.firstReleaseTag == "" {
			t.Fatalf("expected empty list of releases for repo %s, but got first release = %s", tc.repoUrl, releases[0])
		}

		if len(releases) == 0 && tc.firstReleaseTag != "" {
			t.Fatalf("expected non-empty list of releases for repo %s, but no releases were found", tc.repoUrl)
		}

		if releases[len(releases)-1] != tc.firstReleaseTag {
			t.Fatalf("error parsing github releases for repo %s. expected first release = %s. actual = %s", tc.repoUrl, tc.firstReleaseTag, releases[0])
		}

		if releases[0] != tc.lastReleaseTag {
			t.Fatalf("error parsing github releases for repo %s. expected first release = %s. actual = %s", tc.repoUrl, tc.lastReleaseTag, releases[len(releases)-1])
		}
	}
}

func TestExtractUrlIntoGitHubRepo(t *testing.T) {
	cases := []struct {
		repoUrl string
		owner string
		name string
	}{
		{"https://github.com/brikis98/ping-play", "brikis98", "ping-play"},
		{"http://github.com/brikis98/ping-play", "brikis98", "ping-play"},
		{"https://github.com/gruntwork-io/script-modules", "gruntwork-io", "script-modules"},
		{"http://github.com/gruntwork-io/script-modules", "gruntwork-io", "script-modules"},
	}

	for _, tc := range cases {
		repo, err := ExtractUrlIntoGitHubRepo(tc.repoUrl)
		if err != nil {
			t.Fatalf("error extracting url %s into a GitHubRepo struct: %s", tc.repoUrl, err)
		}

		if repo.Owner != tc.owner {
			t.Fatalf("while extracting %s, expected owner %s, received %s", tc.repoUrl, tc.owner, repo.Owner)
		}

		if repo.Name != tc.name {
			t.Fatalf("while extracting %s, expected owner %s, received %s", tc.repoUrl, tc.name, repo.Name)
		}
	}
}

func TextExtractUrlThrowsErrorOnMalformedUrl(t *testing.T) {
	cases := []struct {
		repoUrl string
	}{
		{"https://githubb.com/brikis98/ping-play"},
		{"github.com/brikis98/ping-play"},
		{"curl://github.com/brikis98/ping-play"},
	}

	for _, tc := range cases {
		_, err := ExtractUrlIntoGitHubRepo(tc.repoUrl)
		if err == nil {
			t.Fatalf("Expected error on malformed url %s, but no error was received.", tc.repoUrl)
		}
	}
}