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

func verifyChecksumOfReleaseAsset(assetPath, checksum, algorithm string) *FetchError {
	computedChecksum, err := computeChecksum(assetPath, algorithm)
	if err != nil {
		return newError(ERROR_WHILE_COMPUTING_CHECKSUM, err.Error())
	}
	if computedChecksum != checksum {
		return newError(CHECKSUM_DOES_NOT_MATCH, fmt.Sprintf("Expected to receive checksum value %s, but instead got %s for Release Asset at %s. This means that either you are using the wrong checksum value in your call to fetch (e.g., did you update the version of the module you're installing but not the checksum?) or that someone has replaced the asset with a potentially dangerous one and you should be very careful about proceeding.", computedChecksum, checksum, assetPath))
	}

	fmt.Printf("Checksum matches!")

	return nil
}

func computeChecksum(filePath string, algorithm string) (string, error) {
	fmt.Printf("Computing checksum of release asset\n")

	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher, err := getHasher(algorithm)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(hasher, file)
	if err != nil {
		return "", err
	}

	return hasherToString(hasher), nil
}

// Return a hasher instance, the common interface used by all Golang hashing functions
func getHasher(algorithm string) (hash.Hash, error) {
	switch algorithm {
	case "sha256":
		return sha256.New(), nil
	case "sha512":
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("The checksum algorithm \"%s\" is not supported", algorithm)
	}
}

// Convert a hasher instance to the string value of that hasher
func hasherToString(hasher hash.Hash) string {
	return hex.EncodeToString(hasher.Sum(nil))
}