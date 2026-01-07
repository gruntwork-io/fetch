package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/gruntwork-io/fetch/source"
	_ "github.com/gruntwork-io/fetch/source/github" // Register GitHub source
	_ "github.com/gruntwork-io/fetch/source/gitlab" // Register GitLab source
	"github.com/gruntwork-io/go-commons/logging"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// This variable is set at build time using -ldflags parameters. For more info, see:
// http://stackoverflow.com/a/11355611/483528
var VERSION string

type FetchOptions struct {
	RepoUrl                  string
	GitRef                   string
	CommitSha                string
	BranchName               string
	TagConstraint            string
	GithubToken              string
	GitlabToken              string
	SourceType               string // "github", "gitlab", or "auto"
	SourcePaths              []string
	ReleaseAsset             string
	ReleaseAssetChecksums    map[string]bool
	ReleaseAssetChecksumAlgo string
	Stdout                   bool
	LocalDownloadPath        string
	GithubApiVersion         string
	WithProgress             bool

	// Project logger
	Logger *logrus.Entry
}

type AssetDownloadResult struct {
	assetPath string
	err       error
}

const optionRepo = "repo"
const optionRef = "ref"
const optionCommit = "commit"
const optionBranch = "branch"
const optionTag = "tag"
const optionGithubToken = "github-oauth-token"
const optionSourcePath = "source-path"
const optionReleaseAsset = "release-asset"
const optionReleaseAssetChecksum = "release-asset-checksum"
const optionReleaseAssetChecksumAlgo = "release-asset-checksum-algo"
const optionStdout = "stdout"
const optionGithubAPIVersion = "github-api-version"
const optionWithProgress = "progress"
const optionLogLevel = "log-level"
const optionSource = "source"
const optionGitlabToken = "gitlab-token"

const envVarGithubToken = "GITHUB_OAUTH_TOKEN"
const envVarGitlabToken = "GITLAB_TOKEN"

// Create the Fetch CLI App
func CreateFetchCli(version string, writer io.Writer, errwriter io.Writer) *cli.App {
	app := &cli.App{
		Name:      "fetch",
		Usage:     "fetch makes it easy to download files, folders, and release assets from a specific git commit, branch, or tag of public and private GitHub repos.",
		UsageText: "fetch [global options] <local-download-path>\n   (See https://github.com/gruntwork-io/fetch for examples, argument definitions, and additional docs.)",
		Authors:   []*cli.Author{{Name: "Gruntwork", Email: "www.gruntwork.io"}},
		Version:   version,
		Writer:    writer,
		ErrWriter: errwriter,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  optionRepo,
				Usage: "Required. Fully qualified URL of the GitHub repo.",
			},
			&cli.StringFlag{
				Name:  optionRef,
				Usage: "The git reference to download. If specified, will take lower precendence than --commit, --branch, and --tag.",
			},
			&cli.StringFlag{
				Name:  optionCommit,
				Usage: "The specific git commit SHA to download. If specified, will override --branch and --tag.",
			},
			&cli.StringFlag{
				Name:  optionBranch,
				Usage: "The git branch from which to download the commit; the latest commit in the branch\n\twill be used.\n\tIf specified, will override --tag.",
			},
			&cli.StringFlag{
				Name:  optionTag,
				Usage: "The specific git tag to download, expressed with Version Constraint Operators.\n\tIf left blank, fetch will download the latest git tag.\n\tSee https://github.com/gruntwork-io/fetch#version-constraint-operators for examples.",
			},
			&cli.StringFlag{
				Name:    optionGithubToken,
				Usage:   "A GitHub Personal Access Token, which is required for downloading from private\n\trepos. Populate by setting env var",
				EnvVars: []string{envVarGithubToken},
			},
			&cli.StringSliceFlag{
				Name:  optionSourcePath,
				Usage: "The source path to download from the repo. If this or --release-asset aren't specified,\n\tall files are downloaded. Can be specified more than once.",
			},
			&cli.StringFlag{
				Name:  optionReleaseAsset,
				Usage: "The name of a release asset--that is, a binary uploaded to a GitHub Release--to download.\n\tOnly works with --tag.",
			},
			&cli.StringSliceFlag{
				Name:  optionReleaseAssetChecksum,
				Usage: "The checksum that a release asset should have. Fetch will fail if this value is non-empty\n\tand does not match any of the checksums computed by Fetch.\n\tCan be specified more than once. If more than one\n\trelease asset is downloaded and one or more checksums are provided,\n\tthe asset's checksum must match one.",
			},
			&cli.StringFlag{
				Name:  optionReleaseAssetChecksumAlgo,
				Usage: "The algorithm Fetch will use to compute a checksum of the release asset. Acceptable values\n\tare \"sha256\" and \"sha512\".",
			},
			&cli.StringFlag{
				Name:  optionStdout,
				Usage: "If \"true\", the contents of the release asset is sent to standard output so it can be piped to another command.",
			},
			&cli.StringFlag{
				Name:  optionGithubAPIVersion,
				Value: "v3",
				Usage: "The api version of the GitHub instance. If left blank, v3 will be used.\n\tThis will only be used if the repo url is not a github.com url.",
			},
			&cli.BoolFlag{
				Name:  optionWithProgress,
				Usage: "Display progress on file downloads, especially useful for large files",
			},
			&cli.StringFlag{
				Name:  optionLogLevel,
				Value: logrus.InfoLevel.String(),
				Usage: "The logging level of the command. Acceptable values\n\tare \"trace\", \"debug\", \"info\", \"warn\", \"error\", \"fatal\" and \"panic\".",
			},
			&cli.StringFlag{
				Name:    optionSource,
				Aliases: []string{"s"},
				Value:   "auto",
				Usage:   "The source type to use: \"github\", \"gitlab\", or \"auto\" (auto-detect from URL).",
			},
			&cli.StringFlag{
				Name:    optionGitlabToken,
				Usage:   "A GitLab Personal Access Token for downloading from private GitLab repos.",
				EnvVars: []string{envVarGitlabToken},
			},
		},
		Before: initLogger,
		Action: runFetchWrapper,
	}

	return app
}

func main() {
	app := CreateFetchCli(VERSION, os.Stdout, os.Stderr)

	// Run the definition of App.Action
	app.Run(os.Args)
}

// initLogger initializes the Logger before any command is actually executed. This function will handle all the setup
// code, such as setting up the logger with the appropriate log level.
func initLogger(cliContext *cli.Context) error {
	// Set logging level
	logLevel := cliContext.String(optionLogLevel)
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return fmt.Errorf("Error: %s", err)
	}
	logging.SetGlobalLogLevel(level)
	return nil
}

// We just want to call runFetch(), but app.Action won't permit us to return an error, so call a wrapper function instead.
func runFetchWrapper(c *cli.Context) error {
	// initialize the logger
	logger := GetProjectLogger()
	err := runFetch(c, logger)
	if err != nil {
		logger.Errorf("%s\n", err)
		os.Exit(1)
	}
	return nil
}

// Run the fetch program
func runFetch(c *cli.Context, logger *logrus.Entry) error {
	options := parseOptions(c, logger)
	if err := validateOptions(options); err != nil {
		return err
	}

	// Determine which source to use
	sourceType, err := source.ParseSourceType(options.SourceType)
	if err != nil {
		return err
	}

	// Auto-detect source type from URL if needed
	if sourceType == source.TypeAuto {
		sourceType, err = source.DetectSourceType(options.RepoUrl)
		if err != nil {
			return err
		}
	}

	// Use source package for all sources (GitHub and GitLab)
	return runFetchWithSource(c, logger, options, sourceType)
}

// runFetchWithSource runs fetch using the new source package (for GitLab and future sources)
func runFetchWithSource(c *cli.Context, logger *logrus.Entry, options FetchOptions, sourceType source.SourceType) error {
	// Get the appropriate token
	token := options.GithubToken
	if sourceType == source.TypeGitLab && options.GitlabToken != "" {
		token = options.GitlabToken
	}

	// Create the source
	config := source.Config{
		ApiVersion: options.GithubApiVersion,
		Logger:     logger,
	}

	src, err := source.NewSource(sourceType, config)
	if err != nil {
		return fmt.Errorf("Failed to create source: %s", err)
	}

	logger.Infof("Using %s source for %s\n", src.Type(), options.RepoUrl)

	// Parse the repo URL
	repo, err := src.ParseUrl(options.RepoUrl, token)
	if err != nil {
		return fmt.Errorf("Error parsing repo URL: %s", err)
	}

	// Get tags from repo
	tags, err := src.FetchTags(options.RepoUrl, token)
	if err != nil {
		return fmt.Errorf("Error fetching tags: %s", err)
	}

	// Resolve tag constraint
	var specific bool
	var desiredTag string
	var tagConstraint string

	if options.GitRef != "" {
		specific, desiredTag = isTagConstraintSpecificTag(options.GitRef)
		tagConstraint = options.GitRef
	} else {
		specific, desiredTag = isTagConstraintSpecificTag(options.TagConstraint)
		tagConstraint = options.TagConstraint
	}

	if !specific {
		latestTag, err := getLatestAcceptableTag(tagConstraint, tags)
		if err != nil {
			if err.errorCode == invalidTagConstraintExpression {
				return errors.New(getErrorMessage(invalidTagConstraintExpression, err.details))
			}
			return fmt.Errorf("Error computing latest tag: %s", err)
		}
		desiredTag = latestTag
	}

	// If no release asset and no source paths, download all files
	if len(options.SourcePaths) == 0 && options.ReleaseAsset == "" {
		options.SourcePaths = []string{"/"}
	}

	// Download source paths
	if len(options.SourcePaths) > 0 {
		if err := downloadSourcePathsWithSource(logger, src, options.SourcePaths, options.LocalDownloadPath, repo, desiredTag, options.BranchName, options.CommitSha, token); err != nil {
			return err
		}
	}

	// Download release assets
	var assetPaths []string
	if options.ReleaseAsset != "" {
		assetPaths, err = downloadReleaseAssetsWithSource(logger, src, options.ReleaseAsset, options.LocalDownloadPath, repo, desiredTag, options.WithProgress)
		if err != nil {
			return err
		}
	}

	// Verify checksums
	if len(options.ReleaseAssetChecksums) > 0 {
		for _, assetPath := range assetPaths {
			fetchErr := verifyChecksumOfReleaseAsset(logger, assetPath, options.ReleaseAssetChecksums, options.ReleaseAssetChecksumAlgo)
			if fetchErr != nil {
				return fetchErr
			}
		}
	}

	// Output to stdout if requested
	if options.Stdout {
		if len(assetPaths) == 1 {
			dat, err := os.ReadFile(assetPaths[0])
			if err != nil {
				return err
			}
			c.App.Writer.Write(dat)
		} else if len(assetPaths) > 1 {
			logger.Warn("Multiple assets were downloaded. Ignoring --stdout")
		} else {
			logger.Warn("No assets downloaded. Ignoring --stdout")
		}
	}

	return nil
}

// downloadSourcePathsWithSource downloads source files using the source package
func downloadSourcePathsWithSource(logger *logrus.Entry, src source.Source, sourcePaths []string, destPath string, repo source.Repo, latestTag, branchName, commitSha, token string) error {
	if len(sourcePaths) == 0 {
		return nil
	}

	commit := source.Commit{
		Repo:       repo,
		GitRef:     latestTag,
		GitTag:     latestTag,
		BranchName: branchName,
		CommitSha:  commitSha,
	}

	// Log what we're downloading
	if commit.CommitSha != "" {
		logger.Infof("Downloading commit \"%s\" of %s ...\n", commit.CommitSha, repo.Url)
	} else if commit.BranchName != "" {
		logger.Infof("Downloading latest commit from branch \"%s\" of %s ...\n", commit.BranchName, repo.Url)
	} else if commit.GitTag != "" {
		logger.Infof("Downloading tag \"%s\" of %s ...\n", latestTag, repo.Url)
	} else if commit.GitRef != "" {
		logger.Infof("Downloading ref \"%s\" of %s ...\n", commit.GitRef, repo.Url)
	} else {
		return fmt.Errorf("No commit, tag, branch, or ref specified")
	}

	// Download zip file
	localZipFilePath, err := downloadZipFileWithSource(logger, src, commit, token)
	if err != nil {
		return fmt.Errorf("Error downloading zip: %s", err)
	}
	defer cleanupZipFile(localZipFilePath)

	// Extract files
	for _, sourcePath := range sourcePaths {
		logger.Infof("Extracting files from <repo>%s to %s ...\n", sourcePath, destPath)
		fileCount, err := extractFiles(localZipFilePath, sourcePath, destPath)
		plural := ""
		if fileCount != 1 {
			plural = "s"
		}
		logger.Infof("%d file%s extracted\n", fileCount, plural)
		if err != nil {
			return fmt.Errorf("Error extracting files: %s", err)
		}
	}

	logger.Infof("Download and extraction complete.\n")
	return nil
}

// downloadZipFileWithSource downloads a zip file using the source package
func downloadZipFileWithSource(logger *logrus.Entry, src source.Source, commit source.Commit, token string) (string, error) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", err
	}

	// Create HTTP request
	req, err := src.MakeArchiveRequest(commit, token)
	if err != nil {
		return "", err
	}

	logger.Debugf("Downloading ZIP archive: %s", req.URL)

	// Execute request
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Failed to download: HTTP %d", resp.StatusCode)
	}

	// Write to temp file
	zipPath := path.Join(tempDir, "repo.zip")
	out, err := os.Create(zipPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return zipPath, nil
}

// downloadReleaseAssetsWithSource downloads release assets using the source package
func downloadReleaseAssetsWithSource(logger *logrus.Entry, src source.Source, assetRegex, destPath string, repo source.Repo, tag string, withProgress bool) ([]string, error) {
	var assetPaths []string

	if assetRegex == "" {
		return assetPaths, nil
	}

	// Get release info
	release, err := src.GetReleaseInfo(repo, tag)
	if err != nil {
		return nil, fmt.Errorf("Error getting release info: %s", err)
	}

	// Find matching assets
	pattern, err := regexp.Compile(assetRegex)
	if err != nil {
		return nil, fmt.Errorf("Invalid asset regex: %s", err)
	}

	var matchingAssets []source.ReleaseAsset
	for _, asset := range release.Assets {
		if pattern.MatchString(asset.Name) || asset.Name == assetRegex {
			matchingAssets = append(matchingAssets, asset)
		}
	}

	if len(matchingAssets) == 0 {
		return nil, fmt.Errorf("No assets matching %s in release %s", assetRegex, tag)
	}

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return nil, fmt.Errorf("Failed to create destination directory %s: %s", destPath, err)
	}

	// Download assets concurrently
	var wg sync.WaitGroup
	results := make(chan AssetDownloadResult, len(matchingAssets))

	for _, asset := range matchingAssets {
		wg.Add(1)
		go func(asset source.ReleaseAsset) {
			defer wg.Done()
			assetPath := path.Join(destPath, asset.Name)
			logger.Infof("Downloading asset %s to %s\n", asset.Name, assetPath)
			if err := src.DownloadReleaseAsset(repo, asset, assetPath, withProgress); err == nil {
				logger.Infof("Downloaded %s\n", assetPath)
				results <- AssetDownloadResult{assetPath, nil}
			} else {
				logger.Errorf("Download failed for %s: %s\n", asset.Name, err)
				results <- AssetDownloadResult{assetPath, err}
			}
		}(asset)
	}

	wg.Wait()
	close(results)
	logger.Infof("Download of release assets complete\n")

	var errorStrs []string
	for result := range results {
		if result.err != nil {
			errorStrs = append(errorStrs, fmt.Sprintf("%s: %s", result.assetPath, result.err))
		} else {
			assetPaths = append(assetPaths, result.assetPath)
		}
	}

	if len(errorStrs) > 0 {
		logger.Errorf("%d errors downloading assets:\n\t%s", len(errorStrs), strings.Join(errorStrs, "\n\t"))
	}

	return assetPaths, nil
}

func parseOptions(c *cli.Context, logger *logrus.Entry) FetchOptions {
	localDownloadPath := c.Args().First()
	sourcePaths := c.StringSlice(optionSourcePath)
	assetChecksums := c.StringSlice(optionReleaseAssetChecksum)
	assetChecksumMap := make(map[string]bool, len(assetChecksums))

	// Maintain backwards compatibility with older versions of fetch that passed source paths as an optional first
	// command-line arg
	if c.NArg() == 2 {
		logger.Warnf("DEPRECATION WARNING: passing source paths via command-line args is deprecated. Please use the --%s option instead!\n", optionSourcePath)
		sourcePaths = []string{c.Args().First()}
		localDownloadPath = c.Args().Get(1)
	}

	for _, assetChecksum := range assetChecksums {
		assetChecksumMap[assetChecksum] = true
	}

	return FetchOptions{
		RepoUrl:                  c.String(optionRepo),
		GitRef:                   c.String(optionRef),
		CommitSha:                c.String(optionCommit),
		BranchName:               c.String(optionBranch),
		TagConstraint:            c.String(optionTag),
		GithubToken:              c.String(optionGithubToken),
		GitlabToken:              c.String(optionGitlabToken),
		SourceType:               c.String(optionSource),
		SourcePaths:              sourcePaths,
		ReleaseAsset:             c.String(optionReleaseAsset),
		ReleaseAssetChecksums:    assetChecksumMap,
		ReleaseAssetChecksumAlgo: c.String(optionReleaseAssetChecksumAlgo),
		Stdout:                   c.String(optionStdout) == "true",
		LocalDownloadPath:        localDownloadPath,
		GithubApiVersion:         c.String(optionGithubAPIVersion),
		WithProgress:             c.IsSet(optionWithProgress),
		Logger:                   logger,
	}
}

func validateOptions(options FetchOptions) error {
	if options.RepoUrl == "" {
		return fmt.Errorf("The --%s flag is required. Run \"fetch --help\" for full usage info.", optionRepo)
	}

	if options.LocalDownloadPath == "" {
		return fmt.Errorf("Missing required arguments specifying the local download path. Run \"fetch --help\" for full usage info.")
	}

	if options.GitRef == "" && options.TagConstraint == "" && options.CommitSha == "" && options.BranchName == "" {
		return fmt.Errorf("You must specify exactly one of --%s, --%s, --%s, or --%s. Run \"fetch --help\" for full usage info.", optionRef, optionTag, optionCommit, optionBranch)
	}

	if options.ReleaseAsset != "" && options.TagConstraint == "" {
		return fmt.Errorf("The --%s flag can only be used with --%s. Run \"fetch --help\" for full usage info.", optionReleaseAsset, optionTag)
	}

	if len(options.ReleaseAssetChecksums) > 0 && options.ReleaseAssetChecksumAlgo == "" {
		return fmt.Errorf("If the %s flag is set, you must also enter a value for the %s flag.", optionReleaseAssetChecksum, optionReleaseAssetChecksumAlgo)
	}

	// Validate source type
	validSourceTypes := map[string]bool{"auto": true, "github": true, "gitlab": true}
	if !validSourceTypes[options.SourceType] {
		return fmt.Errorf("Invalid --%s value: %s. Valid values are: auto, github, gitlab", optionSource, options.SourceType)
	}

	return nil
}

// Delete the temp directory containing the zip file.
func cleanupZipFile(localZipFilePath string) error {
	// Remove the entire temp directory containing the zip
	tempDir := filepath.Dir(localZipFilePath)
	err := os.RemoveAll(tempDir)
	if err != nil {
		return fmt.Errorf("Failed to delete temp directory at %s", tempDir)
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
