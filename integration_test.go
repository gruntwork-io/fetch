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
	t.Parallel()

	tmpDownloadPath := createTempDir(t, "fetch-branch-test")

	cases := []struct {
		name       string
		repoUrl    string
		branchName string
		sourcePath string
	}{
		// Test on a public repo whose sole purpose is to be a test fixture for this tool
		{"branch option with public repo", "https://github.com/gruntwork-io/fetch-test-public", "sample-branch", "/"},

		// Private repo equivalent
		{"branch option with private repo", "https://github.com/gruntwork-io/fetch-test-private", "sample-branch", "/"},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			cmd := fmt.Sprintf("fetch --repo %s --branch %s --source-path %s %s", tc.repoUrl, tc.branchName, tc.sourcePath, tmpDownloadPath)
			//t.Fatal(cmd)
			output, _, err := runFetchCommandWithOutput(t, cmd)
			require.NoError(t, err)

			// When --branch is specified, ensure the latest commit is fetched
			assert.Contains(t, output, "Downloading latest commit from branch")
		})
	}
}

func runFetchCommandWithOutput(t *testing.T, command string) (string, string, error) {
	// Note: As most of fetch writes directly to stdout and stderr using the fmt package, we need to temporarily override
	// the OS pipes. This is based loosely on https://stackoverflow.com/questions/10473800/in-go-how-do-i-capture-stdout-of-a-function-into-a-string/10476304#10476304
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}

	stdoutReader, stdoutWriter, err1 := os.Pipe()
	if err1 != nil {
		return "", "", err1
	}

	stderrReader, stderrWriter, err2 := os.Pipe()
	if err2 != nil {
		return "", "", err2
	}

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// override the pipes to capture output
	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter

	// execute the fetch command which produces output
	err := runFetchCommand(t, command)
	if err != nil {
		return "", "", err
	}

	// copy the output to the buffers in seperate goroutines so printing can't block indefinitely
	stdoutC := make(chan bytes.Buffer)
	stderrC := make(chan bytes.Buffer)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, stdoutReader)
		stdoutC <- buf
	}()

	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, stderrReader)
		stderrC <- buf
	}()

	// reset the pipes back to normal
	stdoutWriter.Close()
	stderrWriter.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	stdout = <-stdoutC
	stderr = <-stderrC

	// log the buffers for easier debugging. this is inspired by the integration tests in Terragrunt.
	// For more information, see: https://github.com/gruntwork-io/terragrunt/blob/master/test/integration_test.go.
	logBufferContentsLineByLine(t, stdout, "stdout")
	logBufferContentsLineByLine(t, stderr, "stderr")
	return stdout.String(), stderr.String(), nil
}

func runFetchCommand(t *testing.T, command string) error {
	args := strings.Split(command, " ")

	app := CreateFetchCli(VERSION)
	app.Action = runFetchWrapper
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

// We want to call runFetch() using the app.Action wrapper like the main CLI handler, but we don't want to write to strerr
// and suddenly exit using os.Exit(1), so we use a separate wrapper method in the integration tests.
func runFetchTestWrapper(c *cli.Context) error {
	return runFetch(c)
}
