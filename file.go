package main

import (
	"io/ioutil"
	"os"
	"fmt"
	"net/http"
	"path/filepath"
	"bytes"
	"archive/zip"
	"strings"
)

// Download the zip file at the given URL to a temporary local directory.
// Returns the asbolute path to the downloaded zip file.
// IMPORTANT: You must call "defer os.RemoveAll(dir)" in the calling function when done with the downloaded zip file!
func downloadGithubZipFile(githubRelease gitHubCommit, githubToken string) (string, *fetchError) {

	// Create a temp directory
	// Note that ioutil.TempDir has a peculiar interface. We need not specify any meaningful values to achieve our
	// goal of getting a temporary directory.
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", wrapError(err)
	}

	// Define the url
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/zipball/%s", githubRelease.repo.Owner, githubRelease.repo.Name, githubRelease.gitTag)

	// Download the file, possibly using the GitHub oAuth Token
	httpClient := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", wrapError(err)
	}

	if githubToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", githubToken))
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", wrapError(err)
	}

	// Load the resp.Body into a buffer so we can convert it to a string or []bytes as necessary
	respBodyBuffer := new(bytes.Buffer)
	respBodyBuffer.ReadFrom(resp.Body)

	if resp.StatusCode != 200 {
		return "", newError(500, fmt.Sprintf("Failed to download file at the url %s. Received HTTP Response %d. Body: %s", url, resp.StatusCode, respBodyBuffer.String()))
	}
	if resp.Header.Get("Content-Type") != "application/zip" {
		return "", newError(500, fmt.Sprintf("Failed to download file at the url %s. Expected HTTP Response's \"Content-Type\" header to be \"application/zip\", but was \"%s\"", url, resp.Header.Get("Content-Type")))
	}

	// Copy the contents of the downloaded file to our empty file
	err = ioutil.WriteFile(filepath.Join(tempDir, "repo.zip"), respBodyBuffer.Bytes(), 0644)
	if err != nil {
		return "", wrapError(err)
	}

	return filepath.Join(tempDir, "repo.zip"), nil
}

// Decompresse the file at zipFileAbsPath and move only those files under filesToExtractFromZipPath to localPath
func extractFiles(zipFilePath, filesToExtractFromZipPath, localPath string) error {

	// Open the zip file for reading.
	r, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return err
	}
	defer r.Close()

	// pathPrefix represents the portion of the local file path we will ignore when copying the file to localPath
	// E.g. full path = fetch-test-public-0.0.3/folder/file1.txt
	//      path prefix = fetch-test-public-0.0.3
	//      file that will eventually get written = <localPath>/folder/file1.txt

	// By convention, the first file in the zip file is the top-level directory
	pathPrefix := r.File[0].Name

	// Add the path from which we will extract files to the path prefix so we can exclude the appropriate files
	pathPrefix = filepath.Join(pathPrefix, filesToExtractFromZipPath)

	// Iterate through the files in the archive,
	// printing some of their contents.
	for _, f := range r.File {

		// If the given file is in the filesToExtractFromZipPath, proceed
		if strings.Index(f.Name, pathPrefix) == 0 {

			// Read the contents of the file in the .zip file
			readCloser, err := f.Open()
			if err != nil {
				return fmt.Errorf("Failed to open file %s: %s", f.Name, err)
			}
			defer readCloser.Close()


			if f.FileInfo().IsDir() {
				// Create a directory
				os.MkdirAll(filepath.Join(localPath, strings.TrimPrefix(f.Name, pathPrefix)), 0777)
			} else {
				// Read the file into a byte array
				var bytesBuffer []byte
				readCloser.Read(bytesBuffer)

				// Write the file
				err = ioutil.WriteFile(filepath.Join(localPath, strings.TrimPrefix(f.Name, pathPrefix)), bytesBuffer, 0644)
				if err != nil {
					return fmt.Errorf("Failed to write file: %s", err)
				}
			}
		}
	}

	return nil
}