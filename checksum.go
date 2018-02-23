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

func verifyChecksumOnReleaseAsset(assetPath, checksum, algorithm string) *FetchError {
	computedChecksum, err := computeChecksum(assetPath, algorithm)
	if err != nil {
		return newError(ERROR_WHILE_COMPUTING_CHECKSUM, err.Error())
	}
	if computedChecksum != checksum {
		return newError(CHECKSUM_DOES_NOT_MATCH, fmt.Sprintf("Expected to receive checksum value %s, but instead got %s for Release Asset at %s", computedChecksum, checksum, assetPath))
	}

	fmt.Printf("Checksum matches!")

	return nil
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
		fmt.Printf("Computing checksum of release asset using SHA256\n")
		hasher := sha256.New()
		if _, err := io.Copy(hasher, file); err != nil {
			return checksum, err
		}

		checksum = hasherToString(hasher)
	case "sha512":
		fmt.Printf("Computing checksum of release asset using SHA512\n")
		hasher := sha512.New()
		if _, err := io.Copy(hasher, file); err != nil {
			return checksum, err
		}

		checksum = hasherToString(hasher)
	default:
		return checksum, fmt.Errorf("The checksum algorithm \"%s\" is not supported", algorithm)
	}

	return checksum, nil
}

// Convert a hasher instance (the common interface used by all Golang hashing functions) to the string value of that hasher
func hasherToString(hasher hash.Hash) string {
	return hex.EncodeToString(hasher.Sum(nil))
}