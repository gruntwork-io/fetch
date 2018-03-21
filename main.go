package main

import (
	"os"
	"github.com/codegangsta/cli"
	"fmt"
	"errors"
	"path"
)

// This variable is set at build time using -ldflags parameters. For more info, see:
// http://stackoverflow.com/a/11355611/483528
var VERSION string

type FetchOptions struct {
	RepoUrl                  string
	CommitSha                string
	BranchName               string
	TagConstraint            string
	GithubToken              string
	SourcePaths              []string
	ReleaseAsset             string
	ReleaseAssetChecksum     string
	ReleaseAssetChecksumAlgo string
	ResolveSymlinks          bool
	LocalDownloadPath        string
}

const OPTION_REPO = "repo"
const OPTION_COMMIT = "commit"
const OPTION_BRANCH = "branch"
const OPTION_TAG = "tag"
const OPTION_GITHUB_TOKEN = "github-oauth-token"
const OPTION_SOURCE_PATH = "source-path"
const OPTION_RELEASE_ASSET = "release-asset"
const OPTION_RELEASE_ASSET_CHECKSUM = "release-asset-checksum"
const OPTION_RELEASE_ASSET_CHECKSUM_ALGO = "release-asset-checksum-algo"
const OPTION_RESOLVE_SYMLINKS = "resolve-symlinks"

const ENV_VAR_GITHUB_TOKEN = "GITHUB_OAUTH_TOKEN"

func main() {
	app := cli.NewApp()
	app.Name = "fetch"
	app.Usage = "fetch makes it easy to download files, folders, and release assets from a specific git commit, branch, or tag of public and private GitHub repos."
	app.UsageText = "fetch [global options] <local-download-path>\n   (See https://github.com/gruntwork-io/fetch for examples, argument definitions, and additional docs.)"
	app.Version = VERSION

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  OPTION_REPO,
			Usage: "Required. Fully qualified URL of the GitHub repo.",
		},
		cli.StringFlag{
			Name:  OPTION_COMMIT,
			Usage: "The specific git commit SHA to download. If specified, will override --branch and --tag.",
		},
		cli.StringFlag{
			Name:  OPTION_BRANCH,
			Usage: "The git branch from which to download the commit; the latest commit in the branch\n\twill be used.\n\tIf specified, will override --tag.",
		},
		cli.StringFlag{
			Name:  OPTION_TAG,
			Usage: "The specific git tag to download, expressed with Version Constraint Operators.\n\tIf left blank, fetch will download the latest git tag.\n\tSee https://github.com/gruntwork-io/fetch#version-constraint-operators for examples.",
		},
		cli.StringFlag{
			Name:   OPTION_GITHUB_TOKEN,
			Usage:  "A GitHub Personal Access Token, which is required for downloading from private\n\trepos. Populate by setting env var",
			EnvVar: ENV_VAR_GITHUB_TOKEN,
		},
		cli.StringSliceFlag{
			Name:  OPTION_SOURCE_PATH,
			Usage: "The source path to download from the repo. If this or --release-asset aren't specified,\n\tall files are downloaded. Can be specified more than once.",
		},
		cli.StringFlag{
			Name:  OPTION_RELEASE_ASSET,
			Usage: "The name of a release asset--that is, a binary uploaded to a GitHub Release--to download.\n\tOnly works with --tag.",
		},
		cli.StringFlag{
			Name:  OPTION_RELEASE_ASSET_CHECKSUM,
			Usage: "The checksum that a release asset should have. Fetch will fail if this value is non-empty\n\tand does not match the checksum computed by Fetch.",
		},
		cli.StringFlag{
			Name:  OPTION_RELEASE_ASSET_CHECKSUM_ALGO,
			Usage: "The algorithm Fetch will use to compute a checksum of the release asset. Acceptable values\n\tare \"sha256\" and \"sha512\".",
		},
		cli.BoolFlag{
			Name:  OPTION_RESOLVE_SYMLINKS,
			Usage: "If set to true, Fetch will attempt to resolve any symlinks in the repo by replacing\nthe symlinks with the contents of the targets they point to.",
		},
	}

	app.Action = runFetchWrapper

	// Run the definition of App.Action
	app.Run(os.Args)
}

// We just want to call runFetch(), but app.Action won't permit us to return an error, so call a wrapper function instead.
func runFetchWrapper(c *cli.Context) {
	err := runFetch(c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}

// Run the fetch program
func runFetch(c *cli.Context) error {
	options := parseOptions(c)
	if err := validateOptions(options); err != nil {
		return err
	}

	// Get the tags for the given repo
	tags, fetchErr := FetchTags(options.RepoUrl, options.GithubToken)
	if fetchErr != nil {
		if fetchErr.errorCode == INVALID_GITHUB_TOKEN_OR_ACCESS_DENIED {
			return errors.New(getErrorMessage(INVALID_GITHUB_TOKEN_OR_ACCESS_DENIED, fetchErr.details))
		} else if fetchErr.errorCode == REPO_DOES_NOT_EXIST_OR_ACCESS_DENIED {
			return errors.New(getErrorMessage(REPO_DOES_NOT_EXIST_OR_ACCESS_DENIED, fetchErr.details))
		} else {
			return fmt.Errorf("Error occurred while getting tags from GitHub repo: %s", fetchErr)
		}
	}

	specific, desiredTag := isTagConstraintSpecificTag(options.TagConstraint)
	if !specific {
		// Find the specific release that matches the latest version constraint
		latestTag, err := getLatestAcceptableTag(options.TagConstraint, tags)
		if err != nil {
			if err.errorCode == INVALID_TAG_CONSTRAINT_EXPRESSION {
				return errors.New(getErrorMessage(INVALID_TAG_CONSTRAINT_EXPRESSION, err.details))
			} else {
				return fmt.Errorf("Error occurred while computing latest tag that satisfies version contraint expression: %s", err)
			}
		}
		desiredTag = latestTag
	}

	// Prepare the vars we'll need to download
	repo, fetchErr := ParseUrlIntoGitHubRepo(options.RepoUrl, options.GithubToken)
	if fetchErr != nil {
		return fmt.Errorf("Error occurred while parsing GitHub URL: %s", fetchErr)
	}

	// If no release asset and no source paths are specified, then by default, download all the source files from the repo
	if len(options.SourcePaths) == 0 && options.ReleaseAsset == "" {
		options.SourcePaths = []string{"/"}
	}

	// Download any requested source files
	if err := downloadSourcePaths(repo, desiredTag, options); err != nil {
		return err
	}

	// Download the requested release asset
	assetPath, err := downloadReleaseAsset(options.ReleaseAsset, options.LocalDownloadPath, repo, desiredTag)
	if err != nil {
		return err
	}

	// If applicable, verify the release asset
	if options.ReleaseAssetChecksum != "" {
		fetchErr = verifyChecksumOfReleaseAsset(assetPath, options.ReleaseAssetChecksum, options.ReleaseAssetChecksumAlgo)
		if fetchErr != nil {
			return fetchErr
		}
	}

	return nil
}

func parseOptions(c *cli.Context) *FetchOptions {
	localDownloadPath := c.Args().First()
	sourcePaths := c.StringSlice(OPTION_SOURCE_PATH)

	// Maintain backwards compatibility with older versions of fetch that passed source paths as an optional first
	// command-line arg
	if c.NArg() == 2 {
		fmt.Printf("DEPRECATION WARNING: passing source paths via command-line args is deprecated. Please use the --%s option instead!\n", OPTION_SOURCE_PATH)
		sourcePaths = []string{c.Args().First()}
		localDownloadPath = c.Args().Get(1)
	}

	return &FetchOptions{
		RepoUrl:                  c.String(OPTION_REPO),
		CommitSha:                c.String(OPTION_COMMIT),
		BranchName:               c.String(OPTION_BRANCH),
		TagConstraint:            c.String(OPTION_TAG),
		GithubToken:              c.String(OPTION_GITHUB_TOKEN),
		SourcePaths:              sourcePaths,
		ReleaseAsset:             c.String(OPTION_RELEASE_ASSET),
		ReleaseAssetChecksum:     c.String(OPTION_RELEASE_ASSET_CHECKSUM),
		ReleaseAssetChecksumAlgo: c.String(OPTION_RELEASE_ASSET_CHECKSUM_ALGO),
		ResolveSymlinks:          c.Bool(OPTION_RESOLVE_SYMLINKS),
		LocalDownloadPath:        localDownloadPath,
	}
}

func validateOptions(options *FetchOptions) error {
	if options.RepoUrl == "" {
		return fmt.Errorf("The --%s flag is required. Run \"fetch --help\" for full usage info.", OPTION_REPO)
	}

	if options.LocalDownloadPath == "" {
		return fmt.Errorf("Missing required arguments specifying the local download path. Run \"fetch --help\" for full usage info.")
	}

	if options.TagConstraint == "" && options.CommitSha == "" && options.BranchName == "" {
		return fmt.Errorf("You must specify exactly one of --%s, --%s, or --%s. Run \"fetch --help\" for full usage info.", OPTION_TAG, OPTION_COMMIT, OPTION_BRANCH)
	}

	if options.ReleaseAsset != "" && options.TagConstraint == "" {
		return fmt.Errorf("The --%s flag can only be used with --%s. Run \"fetch --help\" for full usage info.", OPTION_RELEASE_ASSET, OPTION_TAG)
	}

	if options.ReleaseAssetChecksum != "" && options.ReleaseAssetChecksumAlgo == "" {
		return fmt.Errorf("If the %s flag is set, you must also enter a value for the %s flag.", OPTION_RELEASE_ASSET_CHECKSUM, OPTION_RELEASE_ASSET_CHECKSUM_ALGO)
	}

	return nil
}

// Download the specified source files from the given repo
func downloadSourcePaths(githubRepo GitHubRepo, latestTag string, options *FetchOptions) error {
	if len(options.SourcePaths) == 0 {
		return nil
	}

	// We want to respect the GitHubCommit Hierarchy of "CommitSha > GitTag > BranchName"
	// Note that CommitSha or BranchName may be blank here if the user did not specify values for these.
	// If the user specified no value for GitTag, our call to getLatestAcceptableTag() above still gave us some value
	// So we can guarantee (at least logically) that this struct instance is in a valid state right now.
	gitHubCommit := GitHubCommit{
		Repo:       githubRepo,
		GitTag:     latestTag,
		BranchName: options.BranchName,
		CommitSha:  options.CommitSha,
	}

	// Download that release as a .zip file
	if gitHubCommit.CommitSha != "" {
		fmt.Printf("Downloading git commit \"%s\" of %s ...\n", gitHubCommit.CommitSha, githubRepo.Url)
	} else if gitHubCommit.BranchName != "" {
		fmt.Printf("Downloading latest commit from branch \"%s\" of %s ...\n", gitHubCommit.BranchName, githubRepo.Url)
	} else if gitHubCommit.GitTag != "" {
		fmt.Printf("Downloading tag \"%s\" of %s ...\n", latestTag, githubRepo.Url)
	} else {
		return fmt.Errorf("The commit sha, tag, and branch name are all empty.")
	}

	localZipFilePath, err := downloadGithubZipFile(gitHubCommit, githubRepo.Token)
	if err != nil {
		return fmt.Errorf("Error occurred while downloading zip file from GitHub repo: %s", err)
	}
	defer cleanupZipFile(localZipFilePath)

	// Unzip and move the files we need to our destination
	for _, sourcePath := range options.SourcePaths {
		fmt.Printf("Extracting files from <repo>%s to %s ...\n", sourcePath, options.LocalDownloadPath)
		if err := extractFiles(localZipFilePath, sourcePath, options.LocalDownloadPath, options); err != nil {
			return fmt.Errorf("Error occurred while extracting files from GitHub zip file: %s", err.Error())
		}
	}

	fmt.Println("Download and file extraction complete.")
	return nil
}

// Download the specified binary file that was uploaded as a release asset to the specified GitHub release.
// Returns the path where the release asset was downloaded.
func downloadReleaseAsset(assetName string, destPath string, githubRepo GitHubRepo, tag string) (string, error) {
	var assetPath string

	if assetName == "" {
		return assetPath, nil
	}

	release, err := GetGitHubReleaseInfo(githubRepo, tag)
	if err != nil {
		return assetPath, err
	}

	asset := findAssetInRelease(assetName, release)
	if asset == nil {
		return assetPath, fmt.Errorf("Could not find asset %s in release %s", assetName, tag)
	}

	assetPath = path.Join(destPath, asset.Name)
	fmt.Printf("Downloading release asset %s to %s\n", asset.Name, assetPath)
	if err := DownloadReleaseAsset(githubRepo, asset.Id, assetPath); err != nil {
		return assetPath, err
	}

	fmt.Println("Download of release assets complete.")
	return assetPath, nil
}

func findAssetInRelease(assetName string, release GitHubReleaseApiResponse) *GitHubReleaseAsset {
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			return &asset
		}
	}

	return nil
}

// Delete the given zip file.
func cleanupZipFile(localZipFilePath string) error {
	err := os.Remove(localZipFilePath)
	if err != nil {
		return fmt.Errorf("Failed to delete local zip file at %s", localZipFilePath)
	}

	return nil
}

func getErrorMessage(errorCode int, errorDetails string) string {
	switch errorCode {
	case INVALID_TAG_CONSTRAINT_EXPRESSION:
		return fmt.Sprintf(`
The --tag value you entered is not a valid constraint expression.
See https://github.com/gruntwork-io/fetch#version-constraint-operators for examples.

Underlying error message:
%s
`, errorDetails)
	case INVALID_GITHUB_TOKEN_OR_ACCESS_DENIED:
		return fmt.Sprintf(`
Received an HTTP 401 Response when attempting to query the repo for its tags.

This means that either your GitHub oAuth Token is invalid, or that the token is valid but is being used to request access
to either a public repo or a private repo to which you don't have access.

Underlying error message:
%s
`, errorDetails)
	case REPO_DOES_NOT_EXIST_OR_ACCESS_DENIED:
		return fmt.Sprintf(`
Received an HTTP 404 Response when attempting to query the repo for its tags.

This means that either no GitHub repo exists at the URL provided, or that you don't have permission to access it.
If the URL is correct, you may need to pass in a --github-oauth-token.

Underlying error message:
%s
`, errorDetails)
	}

	return ""
}
