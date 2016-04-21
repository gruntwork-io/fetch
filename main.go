package main

import (
	"os"
	"github.com/codegangsta/cli"
	"fmt"
)

func main() {
	app := cli.NewApp()
	app.Name = "fetch"
	app.Usage = "download a file or folder from a specific release of a GitHub repo subject to the Semantic Versioning constraints you impose"
	app.Version = getVersion(Version, VersionPrerelease)

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name: "repo",
			Usage: "Required. Fully qualified URL of the GitHub repo.",
		},
		cli.StringFlag{
			Name: "tag",
			Usage: "The specific git tag to download, expressed with Version Constraint Operators.\n\tIf left blank, fetch will download the latest git tag.\n\tSee https://github.com/gruntwork-io/fetch#version-constraint-operators for examples.",
		},
		cli.StringFlag{
			Name: "github-oauth-token",
			Usage: "A GitHub Personal Access Token (https://help.github.com/articles/creating-an-access-token-for-command-line-use/)\n\tThis is used to authenticate fetch to private GitHub repos.",
		},
	}

	app.Action = func(c *cli.Context) {

		repoUrl := c.String("repo")
		tagConstraint := c.String("tag")
		githubToken := c.String("github-oauth-token")

		// TODO: process repoFilePath and localFileDst args from command line
		repoFilePath := "/"
		localFileDst := "/Users/josh/temp"

		// Validate required args
		if repoUrl == "" {
			fmt.Fprintf(os.Stderr, "ERROR: The --repo argument is required. Run \"%s --help\" for full usage info.", app.Name)
			os.Exit(1)
		}

		// Get the tags for the given repo
		tags, err := FetchTags(repoUrl, githubToken)
		if err != nil {
			if err.errorCode == 401 {
				fmt.Fprintf(os.Stderr, getErrorMessage(401, err.details))
				os.Exit(1)
			} else if err.errorCode == 404 {
				fmt.Fprintf(os.Stderr, getErrorMessage(404, err.details))
				os.Exit(1)
			} else {
				panic(err)
			}
		}

		// Find the specific release that matches the latest version constraint
		latestTag, err := getLatestAcceptableTag(tagConstraint, tags)
		if err != nil {
			if err.errorCode == 100 {
				fmt.Fprintf(os.Stderr, getErrorMessage(100, err.details))
				os.Exit(1)
			} else {
				panic(err)
			}
		}

		// Download that release as a .zip file
		fmt.Printf("Downloading tag \"%s\" of GitHub repo %s\n", latestTag, repoUrl)

		repo, goErr := ExtractUrlIntoGitHubRepo(repoUrl)
		if goErr != nil {
			panic(err)
		}

		gitHubCommit := gitHubCommit{
			repo: repo,
			gitTag: latestTag,
		}

		localZipFilePath, err := downloadGithubZipFile(gitHubCommit, githubToken)
		if err != nil {
			panic(err)
		}
		defer os.Remove(localZipFilePath)

		// Unzip and move the files we need to our destination
		fmt.Printf("Unzipping...\n")
		if goErr = extractFiles(localZipFilePath, repoFilePath, localFileDst); err != nil {
			panic(err)
		}

		fmt.Printf("Download and file extraction complete.")
	}

	// Run the definition of App.Action
	app.Run(os.Args)
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
	case 100:
		return fmt.Sprintf(`
ERROR: The --tag value you entered is not a valid constraint expression.
See https://github.com/gruntwork-io/fetch#version-constraint-operators for examples.

Underlying error message:
%s
`, errorDetails)
	case 401:
		return fmt.Sprintf(`
ERROR: Received an HTTP 401 Response when attempting to query the repo for its tags.

This means that either your GitHub oAuth Token is invalid, or that the token is valid but is being used to request access
to either a public repo or a private repo to which you don't have access.

Underlying error message:
%s
`, errorDetails)
	case 404:
		return fmt.Sprintf(`
ERROR: Received an HTTP 404 Response when attempting to query the repo for its tags.

This means that either no GitHub repo exists at the URL provided, or that you don't have permission to access it.
If the URL is correct, you may need to pass in a --github-oauth-token.

Underlying error message:
%s
`, errorDetails)
	}

	return ""
}