package core

import (
	"fmt"
	"net/http"
	"time"
)

// Transport wraps an existing http.RoundTripper and adds custom headers to each request.
type Transport struct {
	http.RoundTripper

	Headers map[string]string
}

// RoundTrip executes a single HTTP transaction, adding custom headers to the request.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Add the custom headers to the request
	for key, value := range t.Headers {
		req.Header.Add(key, value)
	}

	// Execute the HTTP request and handle errors
	res, err := t.RoundTripper.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("custom transport: %w", err)
	}

	return res, nil
}

func newHTTPClient(headers map[string]string, timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: &Transport{
			RoundTripper: http.DefaultTransport,
			Headers:      headers,
		},
		Timeout: timeout,
	}
}
