package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Download the zip file at the given URL to a temporary local directory.
// Returns the absolute path to the downloaded zip file.
// IMPORTANT: You must call "defer os.RemoveAll(dir)" in the calling function when done with the downloaded zip file!
func downloadGithubZipFile(gitHubCommit GitHubCommit, gitHubToken string, instance GitHubInstance) (string, *FetchError) {
	// Create a temp directory
	// Note that ioutil.TempDir has a peculiar interface. We need not specify any meaningful values to achieve our
	// goal of getting a temporary directory.
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", wrapError(err)
	}

	var zipFilePath = filepath.Join(tempDir, "repo.zip")
	tempFile,err := os.OpenFile(zipFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0544)
	if err != nil {
		return zipFilePath, wrapError(err)
	}

	// Download the zip file, possibly using the GitHub oAuth Token
	httpClient := &http.Client{}
	req, err := MakeGitHubZipFileRequest(gitHubCommit, gitHubToken, instance)
	if err != nil {
		return zipFilePath, wrapError(err)
	}

	done := make(chan int64)
	headResp, err := httpClient.Head(req.URL.String())
	if err != nil {
		return zipFilePath, wrapError(err)
	}
	defer headResp.Body.Close()
	if headResp.StatusCode != 200 {
		return "", newError(FAILED_TO_DOWNLOAD_FILE_HEADER, fmt.Sprintf("Failed to download file header at the url %s. Received HTTP Response %d.", req.URL.String(), headResp.StatusCode))
	}

	size, err := strconv.Atoi(headResp.Header.Get("Content-Length"))
	go func() {
		err := PrintDownloadPercent(done, zipFilePath, int64(size))
		if err != nil {
			fmt.Printf("Failed printing progress: %v", err)
		}
	}()

	resp, err := httpClient.Do(req)
	if err != nil {
		return zipFilePath, wrapError(err)
	}
	if resp.StatusCode != 200 {
		return "", newError(FAILED_TO_DOWNLOAD_FILE, fmt.Sprintf("Failed to download file at the url %s. Received HTTP Response %d.", req.URL.String(), resp.StatusCode))
	}
	if resp.Header.Get("Content-Type") != "application/zip" {
		return "", newError(FAILED_TO_DOWNLOAD_FILE, fmt.Sprintf("Failed to download file at the url %s. Expected HTTP Response's \"Content-Type\" header to be \"application/zip\", but was \"%s\"", req.URL.String(), resp.Header.Get("Content-Type")))
	}

	// Copy the contents of the downloaded file to our empty file
	respBodyBuffer := new(bytes.Buffer)
	_, err = respBodyBuffer.ReadFrom(resp.Body)
	if err != nil {
		return "", wrapError(err)
	}

	_, err = tempFile.Write(respBodyBuffer.Bytes())
	if err != nil {
		return "", wrapError(err)
	}
	return zipFilePath, nil
}

func shouldExtractPathInZip(pathPrefix string, zipPath *zip.File) bool {
	//
	// We need to return true (i.e extract file) based on the following conditions:
	//
	// The current archive item is a directory.
	//     Archive item's path name will always be appended with a "/", so we use
	//     this fact to ensure we are working with a full directory name.
	//     Extract the file if (pathPrefix + "/") is a prefix in path name
	//
	// The current archive item is a file.
	// 		There are two things possible here:
	//		1  User specified a filename that is an exact match for the current archive file,
	//         we need to extract this file.
	//      2  The current archive filename is not a exact match to the user supplied filename.
	//		   Check if (pathPrefix + "/") is a prefix in f.Name, if yes, we extract this file.

	zipPathIsFile := !zipPath.FileInfo().IsDir()
	return (zipPathIsFile && zipPath.Name == pathPrefix) || strings.Index(zipPath.Name, pathPrefix + "/") == 0
}

// Decompress the file at zipFileAbsPath and move only those files under filesToExtractFromZipPath to localPath
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

		// check if current archive file needs to be extracted
		if shouldExtractPathInZip(pathPrefix, f) {

			if f.FileInfo().IsDir() {
				// Create a directory
				path := filepath.Join(localPath, strings.TrimPrefix(f.Name, pathPrefix))
				err = os.MkdirAll(path, 0777)
				if err != nil {
					return fmt.Errorf("Failed to create local directory %s: %s", path, err)
				}
			} else {
				// Read the file into a byte array
				readCloser, err := f.Open()
				if err != nil {
					return fmt.Errorf("Failed to open file %s: %s", f.Name, err)
				}

				byteArray, err := ioutil.ReadAll(readCloser)
				if err != nil {
					return fmt.Errorf("Failed to read file %s: %s", f.Name, err)
				}

				// Write the file
				err = ioutil.WriteFile(filepath.Join(localPath, strings.TrimPrefix(f.Name, pathPrefix)), byteArray, 0644)
				if err != nil {
					return fmt.Errorf("Failed to write file: %s", err)
				}
			}
		}
	}

	return nil
}

// Return an HTTP request that will fetch the given GitHub repo's zip file for the given tag, possibly with the gitHubOAuthToken in the header
// Respects the GitHubCommit hierachy as defined in the code comments for GitHubCommit (e.g. GitTag > CommitSha)
func MakeGitHubZipFileRequest(gitHubCommit GitHubCommit, gitHubToken string, instance GitHubInstance) (*http.Request, error) {
	var request *http.Request

	// This represents either a commit, branch, or git tag
	var gitRef string
	if gitHubCommit.CommitSha != "" {
		gitRef = gitHubCommit.CommitSha
	} else if gitHubCommit.BranchName != "" {
		gitRef = gitHubCommit.BranchName
	} else if gitHubCommit.GitTag != "" {
		gitRef = gitHubCommit.GitTag
	} else {
		return request, fmt.Errorf("Neither a GitCommitSha nor a GitTag nor a BranchName were specified so impossible to identify a specific commit to download.")
	}

	url := fmt.Sprintf("https://%s/repos/%s/%s/zipball/%s", instance.ApiUrl, gitHubCommit.Repo.Owner, gitHubCommit.Repo.Name, gitRef)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return request, wrapError(err)
	}

	if gitHubToken != "" {
		request.Header.Set("Authorization", fmt.Sprintf("token %s", gitHubToken))
	}

	return request, nil
}

// Print the progress once per second on all github fetch requests
func PrintDownloadPercent(done chan int64, path string, total int64) error {
	var stop = false
	for {
		select {
		case <-done:
			stop = true
		default:
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			fi, err := file.Stat()
			if err != nil {
				return err
			}
			size := fi.Size()
			if size == 0 {
				size = 1
			}
			var percent = float64(size) / float64(total) * 100
			fmt.Printf("%.0f %", percent)
		}
		if stop {
			break
		}
		time.Sleep(time.Second)
	}
	return nil
}
