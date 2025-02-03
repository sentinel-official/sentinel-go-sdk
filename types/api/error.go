package api

import (
	"fmt"
)

// Error represents an API error with optional code and message.
type Error struct {
	Code    int    `json:"code,omitempty"`    // Error code
	Message string `json:"message,omitempty"` // Description of the error
}

func (e *Error) String() string {
	return fmt.Sprintf("code=%d, message=%s", e.Code, e.Message)
}

// NewError creates a new Error with the given code and message.
func NewError(code int, msg string) *Error {
	return &Error{
		Code:    code,
		Message: msg,
	}
}
