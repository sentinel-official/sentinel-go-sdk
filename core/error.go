package core

import (
	"errors"
	"fmt"
	"strings"
)

// ErrNotFound is a predefined error representing a "not found" state.
var ErrNotFound = errors.New("not found")

// newErrNotFound wraps an existing error with the predefined ErrNotFound,
func newErrNotFound(err error) error {
	return fmt.Errorf("%w: %v", ErrNotFound, err)
}

// IsTxInMempoolCacheError checks if the error message indicates that the transaction is already present in the mempool cache.
func IsTxInMempoolCacheError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "tx already exists in cache")
}

// IsCodeNotFound checks if the given error string indicates a gRPC NotFound error.
func IsCodeNotFound(err error) error {
	if strings.Contains(err.Error(), "rpc error: code = NotFound") {
		return nil
	}

	return err
}

// IsWrongSequenceError checks if the error message indicates an account sequence mismatch error.
func IsWrongSequenceError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "incorrect account sequence")
}
