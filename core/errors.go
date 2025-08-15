package core

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/x/authz"
)

// ErrNotFound is a predefined error representing a "not found" state.
var ErrNotFound = errors.New("not found")

// NewErrNotFound wraps an existing error with the predefined ErrNotFound.
func NewErrNotFound(err error) error {
	return fmt.Errorf("%w: %v", ErrNotFound, err)
}

// IsWrongSequenceErr checks if the error message indicates an account sequence mismatch error.
func IsWrongSequenceErr(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "incorrect account sequence")
}

// HandleQueryErr processes query errors and returns nil for specific "not found" cases.
// Returns the original error for all other cases.
func HandleQueryErr(err error) error {
	if err == nil {
		return nil
	}

	if strings.Contains(err.Error(), "rpc error: code = NotFound") {
		return nil
	}
	if strings.Contains(err.Error(), authz.ErrNoAuthorizationFound.Error()) {
		return nil
	}
	if strings.Contains(err.Error(), "fee-grant not found") {
		return nil
	}

	return err
}

// HandleBroadcastTxSyncErr processes broadcast transaction errors.
// Returns nil if transaction already exists in cache, otherwise returns the original error.
func HandleBroadcastTxSyncErr(err error) error {
	if err == nil {
		return nil
	}

	if strings.Contains(strings.ToLower(err.Error()), "tx already exists in cache") {
		return nil
	}

	return err
}
