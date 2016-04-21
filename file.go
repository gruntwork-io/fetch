package main

import (
	"io/ioutil"
	"os"
	"fmt"
	"net/http"
	"io"
	"path/filepath"
	"bytes"
)

// Download the zip file at the given URL to a temporary local directory.
// Returns the directory where the file is contained, and the path to the file itself.
// IMPORTANT: You must call defer os.RemoveAll(dir) when done with this directory!
func downloadGithubZipFile(repoOwner, repoName, gitTag, githubToken string) (string, string, *fetchError) {
	// Create a temp directory
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", "", newErr(err)
	}

	// Create an empty file to write to
	file, err := os.Create(filepath.Join(tempDir, "repo.zip"))
	if err != nil {
		return "", "", newErr(err)
	}
	defer file.Close()

	// Define the url
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/zipball/%s", repoOwner, repoName, gitTag)
	fmt.Printf("url = %s\n", url)

	// Download the file, possibly using the GitHub oAuth Token
	httpClient := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", newErr(err)
	}

	if githubToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", githubToken))
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", newErr(err)
	}
	if resp.StatusCode != 200 {
		// Convert the resp.Body to a string
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		respBody := buf.String()

		return "", "", newError(500, fmt.Sprintf("Failed to download file at the url %s. Received HTTP Response %d. Body: %s", url, resp.StatusCode, respBody))
	}
	if resp.Header.Get("Content-Type") != "application/zip" {
		return "", "", newError(500, fmt.Sprintf("Failed to download file at the url %s. Expected HTTP Response's \"Content-Type\" header to be \"application/zip\", but was \"%s\"", url, resp.Header.Get("Content-Type")))
	}

	// Copy the contents of the downloaded file to our empty file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", "", newErr(err)
	}

	return tempDir, filepath.Join(tempDir, "repo.zip"), newEmptyError()
}