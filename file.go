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
// Returns the absolute path to the downloaded zip file.
// IMPORTANT: You must call "defer os.RemoveAll(dir)" in the calling function when done with the downloaded zip file!
func downloadGithubZipFile(gitHubCommit GitHubCommit, gitHubToken string) (string, *FetchError) {

	var zipFilePath string

	// Create a temp directory
	// Note that ioutil.TempDir has a peculiar interface. We need not specify any meaningful values to achieve our
	// goal of getting a temporary directory.
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		return zipFilePath, wrapError(err)
	}

	// Download the zip file, possibly using the GitHub oAuth Token
	httpClient := &http.Client{}
	req, err := MakeGitHubZipFileRequest(gitHubCommit, gitHubToken)
	if err != nil {
		return zipFilePath, wrapError(err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return zipFilePath, wrapError(err)
	}
	if resp.StatusCode != 200 {
		return zipFilePath, newError(FAILED_TO_DOWNLOAD_FILE, fmt.Sprintf("Failed to download file at the url %s. Received HTTP Response %d.", req.URL.String(), resp.StatusCode))
	}
	if resp.Header.Get("Content-Type") != "application/zip" {
		return zipFilePath, newError(FAILED_TO_DOWNLOAD_FILE, fmt.Sprintf("Failed to download file at the url %s. Expected HTTP Response's \"Content-Type\" header to be \"application/zip\", but was \"%s\"", req.URL.String(), resp.Header.Get("Content-Type")))
	}

	// Copy the contents of the downloaded file to our empty file
	respBodyBuffer := new(bytes.Buffer)
	respBodyBuffer.ReadFrom(resp.Body)
	err = ioutil.WriteFile(filepath.Join(tempDir, "repo.zip"), respBodyBuffer.Bytes(), 0644)
	if err != nil {
		return zipFilePath, wrapError(err)
	}

	zipFilePath = filepath.Join(tempDir, "repo.zip")

	return zipFilePath, nil
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

	// Iterate through the files in the archive.
	for _, f := range r.File {

		// If the given file is in the filesToExtractFromZipPath, proceed
		if strings.Index(f.Name, pathPrefix) == 0 {

			path := filepath.Join(localPath, strings.TrimPrefix(f.Name, pathPrefix))

			if f.FileInfo().IsDir() {
				os.MkdirAll(path, 0777)
			} else {
				err = writeFileFromZip(f, path)
				if err != nil {
					return err
				}
			}
		}
	}

	// Sym links may refer to files within the repo, in which case we first need to copy all files, and then process symlinks
	for _, f := range r.File {

		// If the given file is in the filesToExtractFromZipPath, proceed
		if strings.Index(f.Name, pathPrefix) == 0 {

			path := filepath.Join(localPath, strings.TrimPrefix(f.Name, pathPrefix))

			if IsSymLink(f) {
				err := evalSymLinkAndCopyFiles(f, path)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Returns true if the given file is a symlink to a file or dir
func IsSymLink(f *zip.File) bool {
	return f.FileInfo().Mode() & os.ModeSymlink == os.ModeSymlink
}

// Read the contents of a ZIP archive file as a byte array
func readFileFromZip(f *zip.File) ([]byte, error) {
	readCloser, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("Failed to open file %s: %s", f.Name, err)
	}

	byteArray, err := ioutil.ReadAll(readCloser)
	if err != nil {
		return nil, fmt.Errorf("Failed to read file %s: %s", f.Name, err)
	}

	return byteArray, nil
}

// Given a *zip.File (a file located in a ZIP archive), write the the file to the given path
func writeFileFromZip(f *zip.File, path string) error {
	if IsSymLink(f) {
		return writeFileAsSymLink(f, path)
	}

	bytes, err := readFileFromZip(f)
	if err != nil {
		return fmt.Errorf("Failed to read contents of file: %s", err)
	}

	// When we download a ZIP file, permissions are lost from what committed to git, so we assign a sane set of default permissions.
	err = ioutil.WriteFile(path, bytes, 0644)
	if err != nil {
		return fmt.Errorf("Failed to write file: %s", err)
	}

	return nil
}

func writeFileAsSymLink(f *zip.File, path string) error {
	bytes, err := readFileFromZip(f)
	if err != nil {
		return fmt.Errorf("Failed to read contents of file: %s", err)
	}

	target := filepath.Join(filepath.Dir(path), string(bytes))

	err = os.Symlink(target, path)
	if err != nil {
		return fmt.Errorf("Failed to create symlink: %s", err)
	}

	return nil
}

// Resolve the given symlink to its target file or directory, and replace the symlink with the contents of its target
func evalSymLinkAndCopyFiles(f *zip.File, path string) error {
	targetPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("Failed to resolve symlink: %s", err)
	}

	symlinkPath := filepath.Join(filepath.Dir(path), f.FileInfo().Name())

	fileInfo, err := os.Lstat(targetPath)
	if err != nil {
		return fmt.Errorf("Failed to lstat file: %s", err)
	}

	if fileInfo.IsDir() {
		err = replaceSymLinkWithDir(symlinkPath)
		if err != nil {
			return err
		}

		err = copyFolderContents(targetPath, symlinkPath)
		if err != nil {
			return err
		}
	} else {
		err = replaceSymLinkWithFile(symlinkPath, targetPath)
		if err != nil {
			return err
		}
	}

	return nil
}

// Return an HTTP request that will fetch the given GitHub repo's zip file for the given tag, possibly with the gitHubOAuthToken in the header
// Respects the GitHubCommit hierarchy as defined in the code comments for GitHubCommit (e.g. GitTag > CommitSha)
func MakeGitHubZipFileRequest(gitHubCommit GitHubCommit, gitHubToken string) (*http.Request, error) {
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

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/zipball/%s", gitHubCommit.Repo.Owner, gitHubCommit.Repo.Name, gitRef)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return request, wrapError(err)
	}

	if gitHubToken != "" {
		request.Header.Set("Authorization", fmt.Sprintf("token %s", gitHubToken))
	}

	return request, nil
}

// Replace the symlink file with an empty directory using the same permissions as the original symlink
func replaceSymLinkWithDir(path string) error {
	fileInfo, err := os.Lstat(path)
	if err != nil {
		return err
	}

	err = os.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("Failed to replace delete symlink at %s: %s", path, err)
	}

	err = os.MkdirAll(path, fileInfo.Mode())
	if err != nil {
		return fmt.Errorf("Failed to create new directory at %s: %s", path, err)
	}

	return nil
}

// Replace the symlink file with the file it resolves to.
func replaceSymLinkWithFile(symlinkPath string, targetPath string) error {
	err := os.Remove(symlinkPath)
	if err != nil {
		return fmt.Errorf("Failed to replace delete symlink at %s: %s", symlinkPath, err)
	}

	err = copyFile(targetPath, symlinkPath)
	if err != nil {
		return fmt.Errorf("Failed to copy file: %s", err)
	}

	return nil
}

// Copy the files and folders within the source folder into the destination folder.
// This function adapted from https://github.com/gruntwork-io/terragrunt/blob/0786b0b1882917a540e5ccef3f797d3b4c3e2bad/util/file.go#L134-L184
func copyFolderContents(source string, destination string) error {
	files, err := ioutil.ReadDir(source)
	if err != nil {
		return err
	}

	for _, file := range files {
		src := filepath.Join(source, file.Name())
		dest := filepath.Join(destination, file.Name())

		if file.IsDir() {
			if err := os.MkdirAll(dest, file.Mode()); err != nil {
				return err
			}

			if err := copyFolderContents(src, dest); err != nil {
				return err
			}
		} else {
			if err := copyFile(src, dest); err != nil {
				return err
			}
		}
	}

	return nil
}

// Copy a file from source to destination
func copyFile(source string, destination string) error {
	contents, err := ioutil.ReadFile(source)
	if err != nil {
		return err
	}

	// When we download a ZIP file, permissions are lost from what committed to git, so we assign sane default permissions..
	return ioutil.WriteFile(destination, contents, 0644)
}