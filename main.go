package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"

	cli "gopkg.in/urfave/cli.v1"
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
	ReleaseAssetChecksums    map[string]bool
	ReleaseAssetChecksumAlgo string
	LocalDownloadPath        string
	GithubApiVersion         string
	WithProgress             bool
}

type AssetDownloadResult struct {
	assetPath string
	err       error
}

const optionRepo = "repo"
const optionCommit = "commit"
const optionBranch = "branch"
const optionTag = "tag"
const optionGithubToken = "github-oauth-token"
const optionSourcePath = "source-path"
const optionReleaseAsset = "release-asset"
const optionReleaseAssetChecksum = "release-asset-checksum"
const optionReleaseAssetChecksumAlgo = "release-asset-checksum-algo"
const optionGithubAPIVersion = "github-api-version"
const optionWithProgress = "progress"

const envVarGithubToken = "GITHUB_OAUTH_TOKEN"

func main() {
	app := cli.NewApp()
	app.Name = "fetch"
	app.Usage = "fetch makes it easy to download files, folders, and release assets from a specific git commit, branch, or tag of public and private GitHub repos."
	app.UsageText = "fetch [global options] <local-download-path>\n   (See https://github.com/gruntwork-io/fetch for examples, argument definitions, and additional docs.)"
	app.Version = VERSION

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  optionRepo,
			Usage: "Required. Fully qualified URL of the GitHub repo.",
		},
		cli.StringFlag{
			Name:  optionCommit,
			Usage: "The specific git commit SHA to download. If specified, will override --branch and --tag.",
		},
		cli.StringFlag{
			Name:  optionBranch,
			Usage: "The git branch from which to download the commit; the latest commit in the branch\n\twill be used.\n\tIf specified, will override --tag.",
		},
		cli.StringFlag{
			Name:  optionTag,
			Usage: "The specific git tag to download, expressed with Version Constraint Operators.\n\tIf left blank, fetch will download the latest git tag.\n\tSee https://github.com/gruntwork-io/fetch#version-constraint-operators for examples.",
		},
		cli.StringFlag{
			Name:   optionGithubToken,
			Usage:  "A GitHub Personal Access Token, which is required for downloading from private\n\trepos. Populate by setting env var",
			EnvVar: envVarGithubToken,
		},
		cli.StringSliceFlag{
			Name:  optionSourcePath,
			Usage: "The source path to download from the repo. If this or --release-asset aren't specified,\n\tall files are downloaded. Can be specified more than once.",
		},
		cli.StringFlag{
			Name:  optionReleaseAsset,
			Usage: "The name of a release asset--that is, a binary uploaded to a GitHub Release--to download.\n\tOnly works with --tag.",
		},
		cli.StringSliceFlag{
			Name:  optionReleaseAssetChecksum,
			Usage: "The checksum that a release asset should have. Fetch will fail if this value is non-empty\n\tand does not match any of the checksums computed by Fetch.\n\tCan be specified more than once. If more than one\n\trelease asset is downloaded and one or more checksums are provided,\n\tthe asset's checksum must match one.",
		},
		cli.StringFlag{
			Name:  optionReleaseAssetChecksumAlgo,
			Usage: "The algorithm Fetch will use to compute a checksum of the release asset. Acceptable values\n\tare \"sha256\" and \"sha512\".",
		},
		cli.StringFlag{
			Name:  optionGithubAPIVersion,
			Value: "v3",
			Usage: "The api version of the GitHub instance. If left blank, v3 will be used.\n\tThis will only be used if the repo url is not a github.com url.",
		},
		cli.BoolFlag{
			Name:  optionWithProgress,
			Usage: "Display progress on file downloads, especially useful for large files",
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

	instance, fetchErr := ParseUrlIntoGithubInstance(options.RepoUrl, options.GithubApiVersion)
	if fetchErr != nil {
		return fetchErr
	}

	// Get the tags for the given repo
	tags, fetchErr := FetchTags(options.RepoUrl, options.GithubToken, instance)
	if fetchErr != nil {
		if fetchErr.errorCode == invalidGithubTokenOrAccessDenied {
			return errors.New(getErrorMessage(invalidGithubTokenOrAccessDenied, fetchErr.details))
		} else if fetchErr.errorCode == repoDoesNotExistOrAccessDenied {
			return errors.New(getErrorMessage(repoDoesNotExistOrAccessDenied, fetchErr.details))
		} else {
			return fmt.Errorf("Error occurred while getting tags from GitHub repo: %s", fetchErr)
		}
	}

	specific, desiredTag := isTagConstraintSpecificTag(options.TagConstraint)
	if !specific {
		// Find the specific release that matches the latest version constraint
		latestTag, err := getLatestAcceptableTag(options.TagConstraint, tags)
		if err != nil {
			if err.errorCode == invalidTagConstraintExpression {
				return errors.New(getErrorMessage(invalidTagConstraintExpression, err.details))
			} else {
				return fmt.Errorf("Error occurred while computing latest tag that satisfies version contraint expression: %s", err)
			}
		}
		desiredTag = latestTag
	}

	// Prepare the vars we'll need to download
	repo, fetchErr := ParseUrlIntoGitHubRepo(options.RepoUrl, options.GithubToken, instance)
	if fetchErr != nil {
		return fmt.Errorf("Error occurred while parsing GitHub URL: %s", fetchErr)
	}

	// If no release asset and no source paths are specified, then by default, download all the source files from the repo
	if len(options.SourcePaths) == 0 && options.ReleaseAsset == "" {
		options.SourcePaths = []string{"/"}
	}

	// Download any requested source files
	if err := downloadSourcePaths(options.SourcePaths, options.LocalDownloadPath, repo, desiredTag, options.BranchName, options.CommitSha, instance); err != nil {
		return err
	}

	// Download the requested release assets
	assetPaths, err := downloadReleaseAssets(options.ReleaseAsset, options.LocalDownloadPath, repo, desiredTag, options.WithProgress)
	if err != nil {
		return err
	}

	// If applicable, verify the release asset
	if len(options.ReleaseAssetChecksums) > 0 {
		for _, assetPath := range assetPaths {
			fetchErr = verifyChecksumOfReleaseAsset(assetPath, options.ReleaseAssetChecksums, options.ReleaseAssetChecksumAlgo)
			if fetchErr != nil {
				return fetchErr
			}
		}
	}

	return nil
}

func parseOptions(c *cli.Context) FetchOptions {
	localDownloadPath := c.Args().First()
	sourcePaths := c.StringSlice(optionSourcePath)
	assetChecksums := c.StringSlice(optionReleaseAssetChecksum)
	assetChecksumMap := make(map[string]bool, len(assetChecksums))

	// Maintain backwards compatibility with older versions of fetch that passed source paths as an optional first
	// command-line arg
	if c.NArg() == 2 {
		fmt.Printf("DEPRECATION WARNING: passing source paths via command-line args is deprecated. Please use the --%s option instead!\n", optionSourcePath)
		sourcePaths = []string{c.Args().First()}
		localDownloadPath = c.Args().Get(1)
	}

	for _, assetChecksum := range assetChecksums {
		assetChecksumMap[assetChecksum] = true
	}

	return FetchOptions{
		RepoUrl:                  c.String(optionRepo),
		CommitSha:                c.String(optionCommit),
		BranchName:               c.String(optionBranch),
		TagConstraint:            c.String(optionTag),
		GithubToken:              c.String(optionGithubToken),
		SourcePaths:              sourcePaths,
		ReleaseAsset:             c.String(optionReleaseAsset),
		ReleaseAssetChecksums:    assetChecksumMap,
		ReleaseAssetChecksumAlgo: c.String(optionReleaseAssetChecksumAlgo),
		LocalDownloadPath:        localDownloadPath,
		GithubApiVersion:         c.String(optionGithubAPIVersion),
		WithProgress:             c.IsSet(optionWithProgress),
	}
}

func validateOptions(options FetchOptions) error {
	if options.RepoUrl == "" {
		return fmt.Errorf("The --%s flag is required. Run \"fetch --help\" for full usage info.", optionRepo)
	}

	if options.LocalDownloadPath == "" {
		return fmt.Errorf("Missing required arguments specifying the local download path. Run \"fetch --help\" for full usage info.")
	}

	if options.TagConstraint == "" && options.CommitSha == "" && options.BranchName == "" {
		return fmt.Errorf("You must specify exactly one of --%s, --%s, or --%s. Run \"fetch --help\" for full usage info.", optionTag, optionCommit, optionBranch)
	}

	if options.ReleaseAsset != "" && options.TagConstraint == "" {
		return fmt.Errorf("The --%s flag can only be used with --%s. Run \"fetch --help\" for full usage info.", optionReleaseAsset, optionTag)
	}

	if len(options.ReleaseAssetChecksums) > 0 && options.ReleaseAssetChecksumAlgo == "" {
		return fmt.Errorf("If the %s flag is set, you must also enter a value for the %s flag.", optionReleaseAssetChecksum, optionReleaseAssetChecksumAlgo)
	}

	return nil
}

// Download the specified source files from the given repo
func downloadSourcePaths(sourcePaths []string, destPath string, githubRepo GitHubRepo, latestTag string, branchName string, commitSha string, instance GitHubInstance) error {
	if len(sourcePaths) == 0 {
		return nil
	}

	// We want to respect the GitHubCommit Hierarchy of "CommitSha > GitTag > BranchName"
	// Note that CommitSha or BranchName may be blank here if the user did not specify values for these.
	// If the user specified no value for GitTag, our call to getLatestAcceptableTag() above still gave us some value
	// So we can guarantee (at least logically) that this struct instance is in a valid state right now.
	gitHubCommit := GitHubCommit{
		Repo:       githubRepo,
		GitTag:     latestTag,
		BranchName: branchName,
		CommitSha:  commitSha,
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

	localZipFilePath, err := downloadGithubZipFile(gitHubCommit, githubRepo.Token, instance)
	if err != nil {
		return fmt.Errorf("Error occurred while downloading zip file from GitHub repo: %s", err)
	}
	defer cleanupZipFile(localZipFilePath)

	// Unzip and move the files we need to our destination
	for _, sourcePath := range sourcePaths {
		fmt.Printf("Extracting files from <repo>%s to %s ...\n", sourcePath, destPath)
		if err := extractFiles(localZipFilePath, sourcePath, destPath); err != nil {
			return fmt.Errorf("Error occurred while extracting files from GitHub zip file: %s", err.Error())
		}
	}

	fmt.Println("Download and file extraction complete.")
	return nil
}

// Download any matching files that were uploaded as release assets to the specified GitHub release.
// Each file that matches the assetRegex will be downloaded in a separate go routine. If any of the
// downloads fail, an error will be returned. It is possible that only some of the matching assets
// were downloaded. For those that succeeded, the path they were downloaded to will be passed back
// along with the error.
// Returns the paths where the release assets were downloaded.
func downloadReleaseAssets(assetRegex string, destPath string, githubRepo GitHubRepo, tag string, withProgress bool) ([]string, error) {
	var err error
	var assetPaths []string

	if assetRegex == "" {
		return assetPaths, nil
	}

	release, releaseInfoErr := GetGitHubReleaseInfo(githubRepo, tag)
	if releaseInfoErr != nil {
		return nil, err
	}

	assets, err := findAssetsInRelease(assetRegex, release)
	if err != nil {
		return nil, err
	}
	if assets == nil {
		return nil, fmt.Errorf("Could not find assets matching %s in release %s", assetRegex, tag)
	}

	var wg sync.WaitGroup
	results := make(chan AssetDownloadResult, len(assets))

	for _, asset := range assets {
		wg.Add(1)
		go func(asset *GitHubReleaseAsset, results chan<- AssetDownloadResult) {
			// Signal the WaitGroup once this go routine has finished
			defer wg.Done()

			assetPath := path.Join(destPath, asset.Name)
			fmt.Printf("Downloading release asset %s to %s\n", asset.Name, assetPath)
			if downloadErr := DownloadReleaseAsset(githubRepo, asset.Id, assetPath, withProgress); downloadErr == nil {
				fmt.Printf("Downloaded %s\n", assetPath)
				results <- AssetDownloadResult{assetPath, nil}
			} else {
				fmt.Printf("Download failed for %s: %s\n", asset.Name, downloadErr)
				results <- AssetDownloadResult{assetPath, downloadErr}
			}
		}(asset, results)
	}

	wg.Wait()
	close(results)
	fmt.Println("Download of release assets complete")

	var errorStrs []string
	for result := range results {
		if result.err != nil {
			errorStrs = append(errorStrs, fmt.Sprintf("%s: %s", result.assetPath, result.err))
		} else {
			assetPaths = append(assetPaths, result.assetPath)
		}
	}

	if numErrors := len(errorStrs); numErrors > 0 {
		err = fmt.Errorf("%d errors while downloading assets:\n\t%s", numErrors, strings.Join(errorStrs, "\n\t"))
	}

	return assetPaths, err
}

func findAssetsInRelease(assetRegex string, release GitHubReleaseApiResponse) ([](*GitHubReleaseAsset), error) {
	var matches [](*GitHubReleaseAsset)

	pattern, err := regexp.Compile(assetRegex)
	if err != nil {
		return nil, fmt.Errorf("Could not parse provided release asset regex: %s", err.Error())
	}

	for _, asset := range release.Assets {
		matched := pattern.MatchString(asset.Name)
		if matched {
			assetRef := asset
			matches = append(matches, &assetRef)
		}
	}

	return matches, nil
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
	case invalidTagConstraintExpression:
		return fmt.Sprintf(`
The --tag value you entered is not a valid constraint expression.
See https://github.com/gruntwork-io/fetch#version-constraint-operators for examples.

Underlying error message:
%s
`, errorDetails)
	case invalidGithubTokenOrAccessDenied:
		return fmt.Sprintf(`
Received an HTTP 401 Response when attempting to query the repo for its tags.

This means that either your GitHub oAuth Token is invalid, or that the token is valid but is being used to request access
to either a public repo or a private repo to which you don't have access.

Underlying error message:
%s
`, errorDetails)
	case repoDoesNotExistOrAccessDenied:
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
