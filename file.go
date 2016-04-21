package main

import (
	"io/ioutil"
	"os"
	"fmt"
	"net/http"
	"io"
	"path/filepath"
	"bytes"
	"archive/zip"
	"strings"
)

// Download the zip file at the given URL to a temporary local directory.
// Returns the asbolute path to the downloaded zip file.
// IMPORTANT: You must call defer os.RemoveAll(dir) when done with this directory!
func downloadGithubZipFile(repoOwner, repoName, gitTag, githubToken string) (string, *fetchError) {
	// Create a temp directory
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", newErr(err)
	}

	// Create an empty file to write to
	file, err := os.Create(filepath.Join(tempDir, "repo.zip"))
	if err != nil {
		return "", newErr(err)
	}
	defer file.Close()

	// Define the url
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/zipball/%s", repoOwner, repoName, gitTag)

	// Download the file, possibly using the GitHub oAuth Token
	httpClient := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", newErr(err)
	}

	if githubToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", githubToken))
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", newErr(err)
	}
	if resp.StatusCode != 200 {
		// Convert the resp.Body to a string
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		respBody := buf.String()

		return "", newError(500, fmt.Sprintf("Failed to download file at the url %s. Received HTTP Response %d. Body: %s", url, resp.StatusCode, respBody))
	}
	if resp.Header.Get("Content-Type") != "application/zip" {
		return "", newError(500, fmt.Sprintf("Failed to download file at the url %s. Expected HTTP Response's \"Content-Type\" header to be \"application/zip\", but was \"%s\"", url, resp.Header.Get("Content-Type")))
	}

	// Copy the contents of the downloaded file to our empty file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", newErr(err)
	}

	return filepath.Join(tempDir, "repo.zip"), newEmptyError()
}

// extractFiles decompresses the file at zipFileAbsPath and moves only those files under filesToExtractFromZipPath to localPath
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
				// Create a new empty file
				fmt.Printf("Writing file %s\n", filepath.Join(localPath, strings.TrimPrefix(f.Name, pathPrefix)))
				file, err := os.Create(filepath.Join(localPath, strings.TrimPrefix(f.Name, pathPrefix)))
				if err != nil {
					return fmt.Errorf("Failed to create new file: %s", err)
				}
				defer file.Close()

				// Copy the contents to it
				_, err = io.Copy(file, readCloser)
				if err != nil {
					return fmt.Errorf("Failed to copy file: %s", err)
				}
			}
		}
	}

	return nil
}

// getZipFileName extracts the zip file name from the absolute file path of the zip file
// It returns the name without the .zip suffix
func getZipFileName(zipFilePath string) string {
	exploded := strings.Split(zipFilePath, "/")
	return strings.TrimSuffix(exploded[len(exploded)-1], ".zip")
}