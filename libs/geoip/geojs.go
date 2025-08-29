package geoip

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// Ensure GeoJSClient implements the Client interface.
var _ Client = (*GeoJSClient)(nil)

// GeoJSClient is a client for retrieving location data using the GeoJS API.
type GeoJSClient struct {
	c *http.Client
}

// NewGeoJSClient creates and returns a new instance of GeoJSClient.
func NewGeoJSClient() *GeoJSClient {
	return &GeoJSClient{
		c: &http.Client{},
	}
}

// Get retrieves location data for the specified IP address using the GeoJS API.
func (c *GeoJSClient) Get(ctx context.Context, ip string) (*Location, error) {
	// Construct the URL for the API request. Use the provided IP address if it is not empty.
	apiURL := "https://get.geojs.io/v1/ip/geo.json"
	if ip != "" {
		apiURL = fmt.Sprintf("https://get.geojs.io/v1/ip/geo/%s.json", ip)
	}

	// Create the HTTP GET request to the get-geojs.io service.
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request with context: %w", err)
	}

	// Make the request.
	resp, err := c.c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting data for IP %q: %w", ip, err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	// Check if the response status code indicates success.
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request for IP %q failed with status %s", ip, resp.Status)
	}

	// Parse the JSON response into a temporary structure.
	var result struct {
		City        string `json:"city"`
		Country     string `json:"country"`
		CountryCode string `json:"country_code"`
		IP          string `json:"ip"`
		Latitude    string `json:"latitude"`
		Longitude   string `json:"longitude"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response body: %w", err)
	}

	// Convert latitude and longitude from string to float.
	latitude, err := strconv.ParseFloat(result.Latitude, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing latitude: %w", err)
	}
	longitude, err := strconv.ParseFloat(result.Longitude, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing longitude: %w", err)
	}

	// Return the location information as a Location struct.
	return &Location{
		City:        result.City,
		Country:     result.Country,
		CountryCode: result.CountryCode,
		IP:          result.IP,
		Latitude:    latitude,
		Longitude:   longitude,
	}, nil
}
