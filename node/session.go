package node

import (
	"context"
	"fmt"
	"net/http"
)

// AddSessionRequestBody represents the request payload for adding a session.
type AddSessionRequestBody struct {
	Data      string `json:"data" binding:"required,base64,gt=0"`      // Encoded session data (Base64 format), must be present and non-empty.
	ID        uint64 `json:"id" binding:"required,gt=0"`               // Unique identifier for the session, must be greater than zero.
	PubKey    string `json:"pub_key" binding:"required,gt=0"`          // Public key associated with the session, required and non-empty.
	Signature string `json:"signature" binding:"required,base64,gt=0"` // Digital signature to verify the integrity, must be in Base64 format.
}

// AddSessionResult represents the response for adding a session.
type AddSessionResult struct {
	Addrs []string    `json:"addrs"` // List of addresses (IPv4, IPv6, or domain names).
	Data  interface{} `json:"data"`  // Additional response data, format depends on the service.
}

// AddSession adds a session to a node.
func (c *Client) AddSession(ctx context.Context, body *AddSessionRequestBody) (*AddSessionResult, error) {
	path, err := c.getURL(ctx, "sessions")
	if err != nil {
		return nil, fmt.Errorf("failed to get url: %w", err)
	}

	var res AddSessionResult
	if err := c.do(ctx, http.MethodPost, path, body, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
