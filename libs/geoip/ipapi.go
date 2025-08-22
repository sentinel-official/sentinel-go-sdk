package geoip

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Ensure IPAPIClient implements the Client interface.
var _ Client = (*IPAPIClient)(nil)

// IPAPIClient is a client for retrieving location data using the ip-api.com service.
type IPAPIClient struct {
	c *http.Client
}

// NewIPAPIClient creates and returns a new instance of IPAPIClient.
func NewIPAPIClient() *IPAPIClient {
	return &IPAPIClient{
		c: &http.Client{},
	}
}

// Get retrieves location data for the specified IP address using the ip-api.com service.
func (c *IPAPIClient) Get(ctx context.Context, ip string) (*Location, error) {
	// Construct the URL for the API request using the provided IP address.
	apiURL := fmt.Sprintf("http://ip-api.com/json/%s", ip)

	// Create the HTTP GET request to the ip-api.com service.
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request with context: %w", err)
	}

	// Make the request.
	resp, err := c.c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting geolocation data for %q: %w", ip, err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	// Check if the response status code indicates success.
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geolocation request for %q failed with status %s", ip, resp.Status)
	}

	// Parse the JSON response into a temporary structure.
	var result struct {
		City        string  `json:"city"`
		Country     string  `json:"country"`
		CountryCode string  `json:"countryCode"`
		IP          string  `json:"query"` // Note: IP field is named "query" in ip-api.com response.
		Latitude    float64 `json:"lat"`
		Longitude   float64 `json:"lon"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding geolocation response body: %w", err)
	}

	// Return the location information as a Location struct.
	return &Location{
		City:        result.City,
		Country:     result.Country,
		CountryCode: result.CountryCode,
		IP:          result.IP,
		Latitude:    result.Latitude,
		Longitude:   result.Longitude,
	}, nil
}
