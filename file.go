package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

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
	return (zipPathIsFile && zipPath.Name == pathPrefix) || strings.Index(zipPath.Name, pathPrefix+"/") == 0
}

// Decompress the file at zipFileAbsPath and move only those files under filesToExtractFromZipPath to localPath
func extractFiles(zipFilePath, filesToExtractFromZipPath, localPath string) (int, error) {

	// Open the zip file for reading.
	r, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return 0, err
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

	// Count the number of files (not directories) unpacked
	fileCount := 0

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
					return fileCount, fmt.Errorf("Failed to create local directory %s: %s", path, err)
				}
			} else {
				// Read the file into a byte array
				readCloser, err := f.Open()
				if err != nil {
					return fileCount, fmt.Errorf("Failed to open file %s: %s", f.Name, err)
				}

				byteArray, err := io.ReadAll(readCloser)
				if err != nil {
					return fileCount, fmt.Errorf("Failed to read file %s: %s", f.Name, err)
				}

				// Write the file
				err = os.WriteFile(filepath.Join(localPath, strings.TrimPrefix(f.Name, pathPrefix)), byteArray, 0644)
				if err != nil {
					return fileCount, fmt.Errorf("Failed to write file: %s", err)
				}
				fileCount++
			}
		}
	}

	return fileCount, nil
}
