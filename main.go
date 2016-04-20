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

	app.Flags = []cli.Flag {
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
		// Validate required args
		if c.String("repo") == "" {
			fmt.Fprintf(os.Stderr, "ERROR: The --repo argument is required. Run \"%s --help\" for full usage info.", app.Name)
			os.Exit(1)
		}
		//name := "someone"
		//if c.NArg() > 0 {
		//	name = c.Args()[0]
		//}
		//if c.String("repo") == "josh" {
		//	println("Hola")
		//} else {
		//	println("Hello")
		//}
		releases, err := FetchReleases(c.String("repo"), c.String("github-oauth-token"))
		if err != nil {
			if err.errorCode == 401 {
				fmt.Fprintf(os.Stderr, getErrorMessage(401, err.details))
			} else if err.errorCode == 404 {
				fmt.Fprintf(os.Stderr, getErrorMessage(404, err.details))
			} else {
				panic(err)
			}
		}

		fmt.Printf("%v", releases)
	}

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
	case 401:
		return fmt.Sprintf(`
ERROR: Received an HTTP 401 Response when attempting to fetch your files.

This means that either your GitHub oAuth Token is invalid, or that the token is valid but is being used to request access
to either a public repo or a private repo to which you don't have access.

Underlying error message:
%s
`, errorDetails)
	case 404:
		return fmt.Sprintf(`
ERROR: Received an HTTP 404 Response when attempting to fetch your files.

This means that either no GitHub repo exists at the URL provided, or that you don't have permission to access it.
If the URL is correct, you may need to pass in a --github-oauth-token.

Underlying error message:
%s
`, errorDetails)
	}

	return ""
}