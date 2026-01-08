package github

// GitHubInstance represents a GitHub instance (public or enterprise)
type GitHubInstance struct {
	BaseUrl string
	ApiUrl  string
}

// GitHubTagsApiResponse models GitHub API /repos/:owner/:repo/tags response
type GitHubTagsApiResponse struct {
	Name       string `json:"name"`
	ZipBallUrl string `json:"zipball_url"`
	TarballUrl string `json:"tarball_url"`
	Commit     GitHubTagsCommitApiResponse
}

// GitHubTagsCommitApiResponse models commit info in tags response
type GitHubTagsCommitApiResponse struct {
	Sha string `json:"sha"`
	Url string `json:"url"`
}

// GitHubReleaseApiResponse models GitHub API /repos/:owner/:repo/releases/tags/:tag response
type GitHubReleaseApiResponse struct {
	Id     int                   `json:"id"`
	Url    string                `json:"url"`
	Name   string                `json:"name"`
	Assets []GitHubReleaseAsset  `json:"assets"`
}

// GitHubReleaseAsset models asset info in release response
type GitHubReleaseAsset struct {
	Id   int    `json:"id"`
	Url  string `json:"url"`
	Name string `json:"name"`
}
