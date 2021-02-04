package main

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetListOfReleasesFromGitHubRepo(t *testing.T) {
	t.Parallel()
	testInst := GitHubInstance{
		BaseUrl: "github.com",
		ApiUrl:  "api.github.com",
	}

	cases := []struct {
		repoUrl          string
		firstReleaseTag  string
		lastReleaseTag   string
		expectedNumTags  int
		gitHubOAuthToken string
		testInst         GitHubInstance
	}{
		// Test on a public repo whose sole purpose is to be a test fixture for this tool
		{"https://github.com/gruntwork-io/fetch-test-public", "v0.0.1", "v0.0.3", 3, "", testInst},

		// Private repo equivalent
		{"https://github.com/gruntwork-io/fetch-test-private", "v0.0.2", "v0.0.2", 1, os.Getenv("GITHUB_OAUTH_TOKEN"), testInst},
	}

	for _, tc := range cases {
		releases, err := FetchTags(tc.repoUrl, tc.gitHubOAuthToken, testInst)
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

func TestGetNextPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		links       string
		expectedUrl string
	}{
		{"next-and-last-urls", `<https://api.github.com/repos/123456789/example-repo/tags?per_page=100&page=2>; rel="next", <https://api.github.com/repos/123456789/example-repo/tags?per_page=100&page=3>; rel="last"`, "https://api.github.com/repos/123456789/example-repo/tags?per_page=100&page=2"},
		{"next-and-last-urls-no-whitespace", `<https://api.github.com/repos/123456789/example-repo/tags?per_page=100&page=2>;rel="next",<https://api.github.com/repos/123456789/example-repo/tags?per_page=100&page=3>;rel="last"`, "https://api.github.com/repos/123456789/example-repo/tags?per_page=100&page=2"},
		{"next-and-last-urls-extra-whitespace", `    <https://api.github.com/repos/123456789/example-repo/tags?per_page=100&page=2>;      rel="next",     <https://api.github.com/repos/123456789/example-repo/tags?per_page=100&page=3>   ;    rel="last"`, "https://api.github.com/repos/123456789/example-repo/tags?per_page=100&page=2"},
		{"first-and-next-urls", `<https://api.github.com/repos/123456789/example-repo/tags?page=1>; rel="first", <https://api.github.com/repos/123456789/example-repo/tags?per_page=100&page=2>; rel="next"`, "https://api.github.com/repos/123456789/example-repo/tags?per_page=100&page=2"},
		{"next-only", `<https://api.github.com/repos/123456789/example-repo/tags?per_page=100&page=2>; rel="next"`, "https://api.github.com/repos/123456789/example-repo/tags?per_page=100&page=2"},
		{"first-and-last-urls", `<https://api.github.com/repos/123456789/example-repo/tags?page=1>; rel="first", <https://api.github.com/repos/123456789/example-repo/tags?per_page=100&page=2>; rel="last"`, ""},
		{"empty", ``, ""},
		{"garbage", `junk not related to links header at all`, ""},
	}

	for _, tc := range cases {
		// The following is necessary to make sure tc's values don't
		// get updated due to concurrency within the scope of t.Run(..) below
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			nextUrl := getNextUrl(tc.links)
			require.Equal(t, tc.expectedUrl, nextUrl)
		})
	}

}

func TestParseUrlIntoGithubInstance(t *testing.T) {
	t.Parallel()

	ghTestInst := GitHubInstance{
		BaseUrl: "github.com",
		ApiUrl:  "api.github.com",
	}
	wwwGhTestInst := GitHubInstance{
		BaseUrl: "www.github.com",
		ApiUrl:  "api.github.com",
	}
	gheTestInst := GitHubInstance{
		BaseUrl: "ghe.mycompany.com",
		ApiUrl:  "ghe.mycompany.com/api/v3",
	}
	wwwGheTestInst := GitHubInstance{
		BaseUrl: "www.ghe.mycompany.com",
		ApiUrl:  "www.ghe.mycompany.com/api/v3",
	}
	myCoTestInst := GitHubInstance{
		BaseUrl: "mycogithub.com",
		ApiUrl:  "mycogithub.com/api/v3",
	}
	wwwMyCoTestInst := GitHubInstance{
		BaseUrl: "www.mycogithub.com",
		ApiUrl:  "www.mycogithub.com/api/v3",
	}
	localTestInst := GitHubInstance{
		BaseUrl: "mycogithub.local",
		ApiUrl:  "mycogithub.local/api/v3",
	}
	netTestInst := GitHubInstance{
		BaseUrl: "mycogithub.net",
		ApiUrl:  "mycogithub.net/api/v3",
	}

	cases := []struct {
		repoUrl      string
		apiv         string
		expectedInst GitHubInstance
	}{
		{"http://www.github.com/gruntwork-io/script-modules/", "", wwwGhTestInst},
		{"https://www.github.com/gruntwork-io/script-modules/", "", wwwGhTestInst},
		{"http://github.com/gruntwork-io/script-modules/", "", ghTestInst},
		{"http://www.ghe.mycompany.com/gruntwork-io/script-modules", "v3", wwwGheTestInst},
		{"https://www.ghe.mycompany.com/gruntwork-io/script-modules", "v3", wwwGheTestInst},
		{"http://ghe.mycompany.com/gruntwork-io/script-modules", "v3", gheTestInst},
		{"http://www.mycogithub.com/gruntwork-io/script-modules", "v3", wwwMyCoTestInst},
		{"https://www.mycogithub.com/gruntwork-io/script-modules", "v3", wwwMyCoTestInst},
		{"http://mycogithub.com/gruntwork-io/script-modules", "v3", myCoTestInst},
		{"http://mycogithub.local/gruntwork-io/script-modules", "v3", localTestInst},
		{"http://mycogithub.net/gruntwork-io/script-modules", "v3", netTestInst},
	}

	for _, tc := range cases {
		logger := GetProjectLogger()
		inst, err := ParseUrlIntoGithubInstance(logger, tc.repoUrl, tc.apiv)
		if err != nil {
			t.Fatalf("error extracting url %s into a GitHubRepo struct: %s", tc.repoUrl, err)
		}

		if inst.BaseUrl != tc.expectedInst.BaseUrl {
			t.Fatalf("while parsing %s, expected base url %s, received %s", tc.repoUrl, tc.expectedInst.BaseUrl, inst.BaseUrl)
		}

		if inst.ApiUrl != tc.expectedInst.ApiUrl {
			t.Fatalf("while parsing %s, expected api url %s, received %s", tc.repoUrl, tc.expectedInst.ApiUrl, inst.ApiUrl)
		}
	}
}

func TestParseUrlIntoGitHubRepo(t *testing.T) {
	t.Parallel()
	ghTestInst := GitHubInstance{
		BaseUrl: "github.com",
		ApiUrl:  "api.github.com",
	}
	gheTestInst := GitHubInstance{
		BaseUrl: "ghe.mycompany.com",
		ApiUrl:  "ghe.mycompany.com/api/v3",
	}
	myCoTestInst := GitHubInstance{
		BaseUrl: "mycogithub.com",
		ApiUrl:  "mycogithub.com/api/v3",
	}

	cases := []struct {
		repoUrl  string
		owner    string
		name     string
		token    string
		testInst GitHubInstance
	}{
		{"https://github.com/brikis98/ping-play", "brikis98", "ping-play", "", ghTestInst},
		{"http://github.com/brikis98/ping-play", "brikis98", "ping-play", "", ghTestInst},
		{"https://github.com/gruntwork-io/script-modules", "gruntwork-io", "script-modules", "", ghTestInst},
		{"http://github.com/gruntwork-io/script-modules", "gruntwork-io", "script-modules", "", ghTestInst},
		{"http://www.github.com/gruntwork-io/script-modules", "gruntwork-io", "script-modules", "", ghTestInst},
		{"http://www.github.com/gruntwork-io/script-modules/", "gruntwork-io", "script-modules", "", ghTestInst},
		{"http://www.github.com/gruntwork-io/script-modules?foo=bar", "gruntwork-io", "script-modules", "token", ghTestInst},
		{"http://www.github.com/gruntwork-io/script-modules?foo=bar&foo=baz", "gruntwork-io", "script-modules", "token", ghTestInst},
		{"http://www.ghe.mycompany.com/gruntwork-io/script-modules?foo=bar&foo=baz", "gruntwork-io", "script-modules", "token", gheTestInst},
		{"https://www.ghe.mycompany.com/gruntwork-io/script-modules?foo=bar&foo=baz", "gruntwork-io", "script-modules", "token", gheTestInst},
		{"http://ghe.mycompany.com/gruntwork-io/script-modules?foo=bar&foo=baz", "gruntwork-io", "script-modules", "token", gheTestInst},
		{"http://mycogithub.com/gruntwork-io/script-modules?foo=bar&foo=baz", "gruntwork-io", "script-modules", "token", myCoTestInst},
	}

	for _, tc := range cases {
		repo, err := ParseUrlIntoGitHubRepo(tc.repoUrl, tc.token, tc.testInst)
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
	testInst := GitHubInstance{}

	cases := []struct {
		repoUrl string
	}{
		{"https://githubb.com/brikis98/ping-play"},
		{"github.com/brikis98/ping-play"},
		{"curl://github.com/brikis98/ping-play"},
	}

	for _, tc := range cases {
		_, err := ParseUrlIntoGitHubRepo(tc.repoUrl, "", testInst)
		if err == nil {
			t.Fatalf("Expected error on malformed url %s, but no error was received.", tc.repoUrl)
		}
	}
}

func TestGetGitHubReleaseInfo(t *testing.T) {
	t.Parallel()

	token := os.Getenv("GITHUB_OAUTH_TOKEN")

	expectedFetchTestPrivateRelease := GitHubReleaseApiResponse{
		Id:   3064041,
		Url:  "https://api.github.com/repos/gruntwork-io/fetch-test-private/releases/3064041",
		Name: "v0.0.2",
		Assets: []GitHubReleaseAsset{
			{
				Id:   1872521,
				Url:  "https://api.github.com/repos/gruntwork-io/fetch-test-private/releases/assets/1872521",
				Name: "test-asset.png",
			},
		},
	}

	expectedFetchTestPublicRelease := GitHubReleaseApiResponse{
		Id:     3065803,
		Url:    "https://api.github.com/repos/gruntwork-io/fetch-test-public/releases/3065803",
		Name:   "v0.0.3",
		Assets: []GitHubReleaseAsset{},
	}

	testInst := GitHubInstance{
		BaseUrl: "github.com",
		ApiUrl:  "api.github.com",
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
		repo, err := ParseUrlIntoGitHubRepo(tc.repoUrl, tc.repoToken, testInst)
		if err != nil {
			t.Fatalf("Failed to parse %s into GitHub URL due to error: %s", tc.repoUrl, err.Error())
		}

		resp, err := GetGitHubReleaseInfo(repo, tc.tag)
		if err != nil {
			t.Fatalf("Failed to fetch GitHub release info for repo %s due to error: %s", tc.repoToken, err.Error())
		}

		if !reflect.DeepEqual(tc.expected, resp) {
			t.Fatalf("Expected GitHub release %v but got GitHub release %v", tc.expected, resp)
		}
	}
}

func TestDownloadGitHubPulicReleaseAsset(t *testing.T) {
	t.Parallel()

	token := os.Getenv("GITHUB_OAUTH_TOKEN")

	testInst := GitHubInstance{
		BaseUrl: "github.com",
		ApiUrl:  "api.github.com",
	}

	cases := []struct {
		repoUrl   string
		repoToken string
		tag       string
		assetId   int
		progress  bool
	}{
		{"https://github.com/gruntwork-io/fetch-test-private", token, "v0.0.2", 1872521, false},
		{"https://github.com/gruntwork-io/fetch-test-public", "", "v0.0.2", 1872641, false},
		{"https://github.com/gruntwork-io/fetch-test-public", "", "v0.0.2", 1872641, true},
	}

	for _, tc := range cases {
		repo, err := ParseUrlIntoGitHubRepo(tc.repoUrl, tc.repoToken, testInst)
		if err != nil {
			t.Fatalf("Failed to parse %s into GitHub URL due to error: %s", tc.repoUrl, err.Error())
		}

		tmpFile, tmpErr := ioutil.TempFile("", "test-download-release-asset")
		if tmpErr != nil {
			t.Fatalf("Failed to create temp file due to error: %s", tmpErr.Error())
		}

		if err := DownloadReleaseAsset(repo, tc.assetId, tmpFile.Name(), tc.progress); err != nil {
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
