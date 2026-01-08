package gitlab

// GitLabTagResponse models GitLab API /projects/:id/repository/tags response
type GitLabTagResponse struct {
	Name   string `json:"name"`
	Commit struct {
		Id string `json:"id"`
	} `json:"commit"`
}

// GitLabReleaseResponse models GitLab API /projects/:id/releases/:tag response
type GitLabReleaseResponse struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Assets      struct {
		Count   int                  `json:"count"`
		Sources []GitLabAssetSource  `json:"sources"`
		Links   []GitLabAssetLink    `json:"links"`
	} `json:"assets"`
}

// GitLabAssetSource models archive sources in release response
type GitLabAssetSource struct {
	Format string `json:"format"`
	Url    string `json:"url"`
}

// GitLabAssetLink models uploaded asset links in release response
type GitLabAssetLink struct {
	Id              int    `json:"id"`
	Name            string `json:"name"`
	Url             string `json:"url"`
	LinkType        string `json:"link_type"`
	DirectAssetPath string `json:"direct_asset_path"`
}
