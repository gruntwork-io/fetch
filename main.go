package main

import (
	"os"
	"github.com/codegangsta/cli"
	"fmt"
	"errors"
)

func main() {
	app := cli.NewApp()
	app.Name = "fetch"
	app.Usage = "download a file or folder from a specific release of a public or private GitHub repo subject to the Semantic Versioning constraints you impose"
	app.UsageText = "fetch [global options] [<repo-download-filter>] <local-download-path>\n   (See https://github.com/gruntwork-io/fetch for examples, argument definitions, and additional docs.)"
	app.Version = getVersion(Version, VersionPrerelease)

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name: "repo",
			Usage: "Required. Fully qualified URL of the GitHub repo.",
		},
		cli.StringFlag{
			Name: "commit",
			Usage: "The specific git commit SHA to download. If specified, will override --branch and --tag.",
		},
		cli.StringFlag{
			Name: "branch",
			Usage: "The git branch from which to download the commit; the latest commit in th branch will be used. If specified, will override --tag.",
		},
		cli.StringFlag{
			Name: "tag",
			Usage: "The specific git tag to download, expressed with Version Constraint Operators.\n\tIf left blank, fetch will download the latest git tag.\n\tSee https://github.com/gruntwork-io/fetch#version-constraint-operators for examples.",
		},
		cli.StringFlag{
			Name: "github-oauth-token",
			Usage: "Required for private repos. A GitHub Personal Access Token (https://help.github.com/articles/creating-an-access-token-for-command-line-use/).",
		},
	}

	app.Action = runFetchWrapper

	// Run the definition of App.Action
	app.Run(os.Args)
}

// We just want to call runFetch(), but app.Action won't permit us to return an error, so call a wrapper function instead.
func runFetchWrapper (c *cli.Context) {
	err := runFetch(c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}

// Run the fetch program
func runFetch (c *cli.Context) error {

	// Validate required flags
	if c.String("repo") == "" {
		return fmt.Errorf("The --repo flag is required. Run \"fetch --help\" for full usage info.")
	}

	repoUrl := c.String("repo")
	commitSha := c.String("commit")
	branchName := c.String("branch")
	tagConstraint := c.String("tag")
	githubToken := c.String("github-oauth-token")

	// Validate args
	if len(c.Args()) == 0 || len(c.Args()) > 2 {
		return fmt.Errorf("Missing required arguments. Run \"fetch --help\" for full usage info.")
	}

	var repoDownloadFilter string
	var localFileDst string

	// Assume the <repo-download-filter> arg is missing, so set a default
	if len(c.Args()) == 1 {
		repoDownloadFilter = "/"
		localFileDst = c.Args()[0]
	}

	// We have two args so load both
	if len(c.Args()) == 2 {
		repoDownloadFilter = c.Args()[0]
		localFileDst = c.Args()[1]
	}

	// Get the tags for the given repo
	tags, err := FetchTags(repoUrl, githubToken)
	if err != nil {
		if err.errorCode == INVALID_GITHUB_TOKEN_OR_ACCESS_DENIED {
			return errors.New(getErrorMessage(INVALID_GITHUB_TOKEN_OR_ACCESS_DENIED, err.details))
		} else if err.errorCode == REPO_DOES_NOT_EXIST_OR_ACCESS_DENIED {
			return errors.New(getErrorMessage(REPO_DOES_NOT_EXIST_OR_ACCESS_DENIED, err.details))
		} else {
			return fmt.Errorf("Error occurred while getting tags from GitHub repo: %s", err)
		}
	}

	// Find the specific release that matches the latest version constraint
	latestTag, err := getLatestAcceptableTag(tagConstraint, tags)
	if err != nil {
		if err.errorCode == INVALID_TAG_CONSTRAINT_EXPRESSION {
			return errors.New(getErrorMessage(INVALID_TAG_CONSTRAINT_EXPRESSION, err.details))
		} else {
			return fmt.Errorf("Error occurred while computing latest tag that satisfies version contraint expression: %s", err)
		}
	}

	// Prepare the vars we'll need to download
	repo, goErr := ParseUrlIntoGitHubRepo(repoUrl)
	if goErr != nil {
		return fmt.Errorf("Error occurred while parsing GitHub URL: %s", err)
	}

	// We want to respect the GitHubCommit Hierarchy of "CommitSha > GitTag > BranchName"
	// Note that CommitSha or BranchName may be blank here if the user did not specify values for these.
	// If the user specified no value for GitTag, our call to getLatestAcceptableTag() above still gave us some value
	// So we can guarantee (at least logically) that this struct instance is in a valid state right now.
	gitHubCommit := GitHubCommit{
		Repo: repo,
		GitTag: latestTag,
		BranchName: branchName,
		CommitSha: commitSha,
	}

	// Download that release as a .zip file
	if gitHubCommit.CommitSha != "" {
		fmt.Printf("Downloading git commit \"%s\" of %s ...\n", gitHubCommit.CommitSha, repoUrl)
	} else if gitHubCommit.BranchName != "" {
		fmt.Printf("Downloading latest commit from branch \"%s\" of %s ...\n", gitHubCommit.BranchName, repoUrl)
	} else if gitHubCommit.GitTag != "" {
		fmt.Printf("Downloading tag \"%s\" of %s ...\n", latestTag, repoUrl)
	} else {
		return fmt.Errorf("The commit sha, tag, and branch name are all empty.")
	}

	localZipFilePath, err := downloadGithubZipFile(gitHubCommit, githubToken)
	if err != nil {
		return fmt.Errorf("Error occurred while downloading zip file from GitHub repo: %s", err)
	}
	defer cleanupZipFile(localZipFilePath)

	// Unzip and move the files we need to our destination
	fmt.Printf("Extracting files from <repo>%s to %s ...\n", repoDownloadFilter, localFileDst)
	if goErr = extractFiles(localZipFilePath, repoDownloadFilter, localFileDst); goErr != nil {
		return fmt.Errorf("Error occurred while extracting files from GitHub zip file: %s", goErr)
	}

	fmt.Println("Download and file extraction complete.")
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

// getVersion returns a properly formatted version string
func getVersion(version string, versionPreRelease string) string {
	if versionPreRelease != "" {
		return version
	} else {
		return fmt.Sprintf("%s-%s", version, versionPreRelease)
	}
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