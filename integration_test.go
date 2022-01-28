package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cli "gopkg.in/urfave/cli.v1"
)

func TestFetchWithBranchOption(t *testing.T) {
	tmpDownloadPath := createTempDir(t, "fetch-branch-test")

	cases := []struct {
		name         string
		repoUrl      string
		branchName   string
		sourcePath   string
		expectedFile string
	}{
		// Test on a public repo whose sole purpose is to be a test fixture for this tool
		{"branch option with public repo", "https://github.com/gruntwork-io/fetch-test-public", "sample-branch", "/", "foo.txt"},

		// Private repo equivalent
		{"branch option with private repo", "https://github.com/gruntwork-io/fetch-test-private", "sample-branch", "/", "bar.txt"},
	}

	for _, tc := range cases {
		// The following is necessary to make sure tc's values don't
		// get updated due to concurrency within the scope of t.Run(..) below
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cmd := fmt.Sprintf("fetch --repo %s --branch %s --source-path %s %s", tc.repoUrl, tc.branchName, tc.sourcePath, tmpDownloadPath)
			_, erroutput, err := runFetchCommandWithOutput(t, cmd)
			require.NoError(t, err)

			// When --branch is specified, ensure the latest commit is fetched
			assert.Contains(t, erroutput, "Downloading latest commit from branch")

			// Ensure the expected file was downloaded
			assert.FileExists(t, JoinPath(tmpDownloadPath, tc.expectedFile))
		})
	}
}

func TestFetchWithStdoutOption(t *testing.T) {
	tmpDownloadPath, err := ioutil.TempDir("", "fetch-stdout-test")
	require.NoError(t, err)

	repoUrl := "https://github.com/gruntwork-io/fetch-test-public"
	releaseTag := "v0.0.4"
	releaseAsset := "hello+world.txt"

	cmd := fmt.Sprintf("fetch --repo %s --tag %s --release-asset %s --stdout true %s", repoUrl, releaseTag, releaseAsset, tmpDownloadPath)
	t.Logf("Testing command: %s", cmd)
	stdoutput, _, err := runFetchCommandWithOutput(t, cmd)
	require.NoError(t, err)

	// Ensure the expected file was downloaded
	assert.FileExists(t, JoinPath(tmpDownloadPath, releaseAsset))

	// When --stdout is specified, ensure the file contents are piped to the standard output stream
	assert.Contains(t, stdoutput, "hello world")
}

func TestFetchWithStdoutOptionMultipleAssets(t *testing.T) {
	tmpDownloadPath, err := ioutil.TempDir("", "fetch-stdout-test")
	require.NoError(t, err)

	repoUrl := SAMPLE_RELEASE_ASSET_GITHUB_REPO_URL
	releaseTag := SAMPLE_RELEASE_ASSET_VERSION
	releaseAsset := SAMPLE_RELEASE_ASSET_REGEX

	cmd := fmt.Sprintf("fetch --repo %s --tag %s --release-asset %s --stdout true %s", repoUrl, releaseTag, releaseAsset, tmpDownloadPath)
	t.Logf("Testing command: %s", cmd)
	_, stderr, err := runFetchCommandWithOutput(t, cmd)
	require.NoError(t, err)

	// When --stdout is specified, ensure the file contents are piped to the standard output stream
	assert.Contains(t, stderr, "Multiple assets were downloaded. Ignoring --stdout")
}

func runFetchCommandWithOutput(t *testing.T, command string) (string, string, error) {
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}

	err := runFetchCommand(t, command, &stdout, &stderr)
	if err != nil {
		return "", "", err
	}

	// log the buffers for easier debugging. this is inspired by the integration tests in Terragrunt.
	// For more information, see: https://github.com/gruntwork-io/terragrunt/blob/master/test/integration_test.go.
	logBufferContentsLineByLine(t, stdout, "stdout")
	logBufferContentsLineByLine(t, stderr, "stderr")
	return stdout.String(), stderr.String(), nil
}

func runFetchCommand(t *testing.T, command string, writer io.Writer, errwriter io.Writer) error {
	args := strings.Split(command, " ")

	app := CreateFetchCli(VERSION, writer, errwriter)
	app.Action = runFetchTestWrapper
	return app.Run(args)
}

func logBufferContentsLineByLine(t *testing.T, out bytes.Buffer, label string) {
	t.Logf("[%s] Full contents of %s:", t.Name(), label)
	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		t.Logf("[%s] %s", t.Name(), line)
	}
}

func createTempDir(t *testing.T, prefix string) string {
	dir, err := ioutil.TempDir(os.TempDir(), prefix)
	if err != nil {
		t.Fatalf("Could not create temporary directory due to error: %v", err)
	}
	defer os.RemoveAll(dir)
	return dir
}

// We want to call runFetch() using the app.Action wrapper like the main CLI handler, but we don't want to write to stderr
// and suddenly exit using os.Exit(1), so we use a separate wrapper method in the integration tests.
func runFetchTestWrapper(c *cli.Context) error {
	// initialize the logger
	logger := GetProjectLoggerWithWriter(c.App.ErrWriter)
	return runFetch(c, logger)
}
