package main

import (
	"fmt"
	"os"
	"testing"

	cli "gopkg.in/urfave/cli.v1"
)

// Expect to download 2 assets:
// - health-checker_linux_386
// - health-checker_linux_amd64
const SAMPLE_RELEASE_ASSET_REGEX = "health-checker_linux_[a-z0-9]+"

func TestDownloadReleaseAssets(t *testing.T) {
	tmpDir := mkTempDir(t)
	testInst := GitHubInstance{
		BaseUrl: "github.com",
		ApiUrl:  "api.github.com",
	}

	githubRepo, err := ParseUrlIntoGitHubRepo(SAMPLE_RELEASE_ASSET_GITHUB_REPO_URL, "", testInst)
	if err != nil {
		t.Fatalf("Failed to parse sample release asset GitHub URL into Fetch GitHubRepo struct: %s", err)
	}

	assetPaths, fetchErr := downloadReleaseAssets(SAMPLE_RELEASE_ASSET_REGEX, tmpDir, githubRepo, SAMPLE_RELEASE_ASSET_VERSION)
	if fetchErr != nil {
		t.Fatalf("Failed to download release asset: %s", fetchErr)
	}

	if len(assetPaths) != 2 {
		t.Fatalf("Expected to download 2 assets, not %d", len(assetPaths))
	}

	for _, assetPath := range assetPaths {
		if _, err := os.Stat(assetPath); os.IsNotExist(err) {
			t.Fatalf("Downloaded file should exist at %s", assetPath)
		} else {
			fmt.Printf("Verified the downloaded asset exists at %s\n", assetPath)
		}
	}
}

func TestInvalidReleaseAssetsRegex(t *testing.T) {
	tmpDir := mkTempDir(t)
	testInst := GitHubInstance{
		BaseUrl: "github.com",
		ApiUrl:  "api.github.com",
	}

	githubRepo, err := ParseUrlIntoGitHubRepo(SAMPLE_RELEASE_ASSET_GITHUB_REPO_URL, "", testInst)
	if err != nil {
		t.Fatalf("Failed to parse sample release asset GitHub URL into Fetch GitHubRepo struct: %s", err)
	}

	_, fetchErr := downloadReleaseAssets("*", tmpDir, githubRepo, SAMPLE_RELEASE_ASSET_VERSION)
	if fetchErr == nil {
		t.Fatalf("Expected error for invalid regex")
	}
}

func TestEmptyOptionValues(t *testing.T) {
	app := cli.NewApp()
	app.Name = "main_test"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  OPTION_REPO,
		},
		cli.StringFlag{
			Name:  OPTION_COMMIT,
		},
		cli.StringFlag{
			Name:  OPTION_BRANCH,
		},
		cli.StringFlag{
			Name:  OPTION_TAG,
		},
		cli.StringFlag{
			Name:   OPTION_GITHUB_TOKEN,
			EnvVar: ENV_VAR_GITHUB_TOKEN,
		},
		cli.StringSliceFlag{
			Name:  OPTION_SOURCE_PATH,
		},
		cli.StringFlag{
			Name:  OPTION_RELEASE_ASSET,
		},
		cli.StringSliceFlag{
			Name:  OPTION_RELEASE_ASSET_CHECKSUM,
		},
		cli.StringFlag{
			Name:  OPTION_RELEASE_ASSET_CHECKSUM_ALGO,
		},
		cli.StringFlag{
			Name:  OPTION_GITHUB_API_VERSION,
			Value: "v3",
		},
	}

    app.Action = func(c *cli.Context) error {
        _, err := parseOptions(c)
        return err
    }

    var args []string
    var expected string
    var err error

    optionsList := []string{
        OPTION_REPO,
        OPTION_COMMIT,
        OPTION_BRANCH,
        OPTION_TAG,
        OPTION_GITHUB_TOKEN,
        OPTION_SOURCE_PATH,
        OPTION_RELEASE_ASSET,
        OPTION_RELEASE_ASSET_CHECKSUM,
        OPTION_RELEASE_ASSET_CHECKSUM_ALGO,
        OPTION_GITHUB_API_VERSION,
    }

    for _, option := range optionsList {
        args = os.Args[0:1]
        dashedOption := "--" + option
        emptyOption := dashedOption + "="
        args = append(args, emptyOption)

        err = app.Run(args)
        expected = fmt.Sprintf("You specified the %s flag but did not provide any value.", dashedOption)
        if (err != nil) && (err.Error() != expected) { 
            t.Fatalf("Expected '%s' but received '%s'", expected, err.Error())
        }

        if err == nil {
            t.Fatalf("Expected '%s' but received nothing", expected)
        }
    }
}
