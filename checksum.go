package main

import (
	"fmt"
	"crypto/sha256"
	"crypto/sha512"
	"os"
	"io"
	"hash"
	"encoding/hex"
)

func verifyChecksumOnReleaseAssets(assetPaths, checksums, algorithms []string) *FetchError {
	if ! isAllEqual(len(assetPaths), len(checksums), len(algorithms)) {
		return newError(VARYING_LIST_SIZES_OF_RELEASE_ASSETS_CHECKSUMS_AND_ALGOS, "--release-asset, --release-asset-checksum, and --release-asset-checksum-algo must all be specified an equal number of times.")
	}

	for i := 0; i < len(assetPaths); i++ {
		assetPath := assetPaths[i]
		checksum := checksums[i]
		algo := algorithms[i]

		computedChecksum, err := computeChecksum(assetPath, algo)
		if err != nil {
			return newError(ERROR_WHILE_COMPUTING_CHECKSUM, fmt.Sprintf("Error while computing checksum: %s\n", err))
		}
		if computedChecksum != checksum {
			return newError(CHECKSUM_DOES_NOT_MATCH, fmt.Sprintf("Expected to receive checksum value %s, but instead got %s for Release Asset at %s", computedChecksum, checksum, assetPath))
		}
	}

	return &FetchError{
		err: fmt.Errorf("Placholder error."),
	}
}

func computeChecksum(filePath string, algorithm string) (string, error) {
	var checksum string

	file, err := os.Open(filePath)
	if err != nil {
		return checksum, err
	}
	defer file.Close()

	switch algorithm {
	case "sha256":
		hasher := sha256.New()
		if _, err := io.Copy(hasher, file); err != nil {
			return checksum, err
		}

		checksum = hasherToString(hasher)
	case "sha512":
		hasher := sha512.New()
		if _, err := io.Copy(hasher, file); err != nil {
			return checksum, err
		}

		checksum = hasherToString(hasher)
	default:
		return checksum, newError(UNSUPPORTED_CHECKSUM_ALGO, fmt.Sprintf("The checksum algorithm \"%s\" is not supported", algorithm))
	}

	return checksum, nil
}

// Convert a hasher instance (the common interface used by all Golang hashing functions) to the string value of that hasher
func hasherToString(hasher hash.Hash) string {
	return hex.EncodeToString(hasher.Sum(nil))
}

func isAllEqual(nums... int) bool {
	if len(nums) == 0 {
		return true
	}

	firstNum := nums[0]
	for _, num := range nums {
		if firstNum != num {
			return false
		}
	}

	return true
}