package types

import (
	"errors"
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

// Response standardizes API response structures.
type Response struct {
	Success bool        `json:"success"`          // Success status of the operation
	Error   *Error      `json:"error,omitempty"`  // Details of any error that occurred
	Result  interface{} `json:"result,omitempty"` // Result data of the operation
}

func (r *Response) Err() error {
	if r.Success {
		return nil
	}
	if r.Error != nil {
		return errors.New(r.Error.String())
	}

	return errors.New("unknown error")
}

// NewResponseError returns a Response indicating a failure with the specified error details.
// The message parameter can be either an error or a string.
func NewResponseError(code int, v interface{}) *Response {
	var msg string
	switch v := v.(type) {
	case error:
		msg = v.Error()
	case string:
		msg = v
	default:
		msg = "unknown error"
	}

	return &Response{
		Success: false,
		Error:   NewError(code, msg),
	}
}

// NewResponseResult returns a Response indicating a success with the provided result data.
func NewResponseResult(v interface{}) *Response {
	return &Response{
		Success: true,
		Result:  v,
	}
}
