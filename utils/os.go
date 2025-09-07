package utils

import (
	"fmt"
	"os"
)

// IsFileExists checks whether the given file path exists and is accessible.
func IsFileExists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, fmt.Errorf("stat %q: %w", path, err)
	}

	return true, nil
}

// RemoveFile deletes the file at the specified path.
// It returns nil if the file does not exist or is successfully deleted.
// If the file removal fails, it returns an error.
func RemoveFile(path string) error {
	// Check if the file exists at the given path.
	exists, err := IsFileExists(path)
	if err != nil {
		return fmt.Errorf("checking if file %q exists: %w", path, err)
	}

	if !exists {
		return nil
	}

	// Remove the file and return the resulting error, if any.
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("removing file %q: %w", path, err)
	}

	return nil
}
