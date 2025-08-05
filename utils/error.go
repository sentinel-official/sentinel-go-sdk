package utils

import (
	"errors"
)

// ErrorIs checks whether the error 'v' matches any of the provided target errors.
func ErrorIs(v error, targets ...error) bool {
	for _, err := range targets {
		if errors.Is(v, err) {
			return true
		}
	}

	return false
}
