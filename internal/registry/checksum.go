package registry

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

func CalculateSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file for checksum: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("calculate checksum: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func VerifySHA256(path, expected string) error {
	expected = strings.ToLower(strings.TrimSpace(expected))
	expectedBytes, err := hex.DecodeString(expected)
	if err != nil || len(expectedBytes) != sha256.Size {
		return fmt.Errorf("invalid SHA-256 checksum %q", expected)
	}
	actual, err := CalculateSHA256(path)
	if err != nil {
		return err
	}
	actualBytes, _ := hex.DecodeString(actual)
	if subtle.ConstantTimeCompare(actualBytes, expectedBytes) != 1 {
		return fmt.Errorf("%w: expected %s, got %s", ErrChecksumMismatch, expected, actual)
	}
	return nil
}
