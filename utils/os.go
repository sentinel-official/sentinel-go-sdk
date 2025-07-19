package utils

import (
	"os"
)

// IsFileExists checks whether the given file path exists and is accessible.
func IsFileExists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// RemoveFile deletes the file at the specified path.
// It returns nil if the file does not exist or is successfully deleted.
// If the file removal fails, it returns an error.
func RemoveFile(path string) error {
	// Check if the file exists at the given path.
	exist, err := IsFileExists(path)
	if err != nil {
		return err
	}
	if !exist {
		return nil
	}

	// Remove the file and return the resulting error, if any.
	return os.Remove(path)
}
