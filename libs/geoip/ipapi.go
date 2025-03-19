package geoip

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Ensure IPAPIClient implements the Client interface.
var _ Client = (*IPAPIClient)(nil)

// IPAPIClient is a client for retrieving location data using the ip-api.com service.
type IPAPIClient struct {
	c *http.Client
}

// NewIPAPIClient creates and returns a new instance of IPAPIClient with the specified timeout and optional proxy address.
func NewIPAPIClient(proxyAddr string, timeout time.Duration) (*IPAPIClient, error) {
	transport := &http.Transport{}

	// If a proxy address is provided, configure the HTTP client to use it.
	if proxyAddr != "" {
		proxyURL, err := url.Parse(proxyAddr)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy addr: %w", err)
		}

		transport.Proxy = http.ProxyURL(proxyURL)
	}

	return &IPAPIClient{
		c: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}, nil
}

// Get retrieves location data for the specified IP address using the ip-api.com service.
func (c *IPAPIClient) Get(ip string) (*Location, error) {
	// Construct the URL for the API request using the provided IP address.
	apiURL := fmt.Sprintf("http://ip-api.com/json/%s", ip)

	// Make the HTTP GET request to the ip-api.com service.
	resp, err := c.c.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check if the response status code indicates success.
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to retrieve data, status: %s", resp.Status)
	}

	// Parse the JSON response into a temporary structure.
	var result struct {
		City      string  `json:"city"`
		Country   string  `json:"country"`
		IP        string  `json:"query"` // Note: IP field is named "query" in ip-api.com response.
		Latitude  float64 `json:"lat"`
		Longitude float64 `json:"lon"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Return the location information as a Location struct.
	return &Location{
		City:      result.City,
		Country:   result.Country,
		IP:        result.IP,
		Latitude:  result.Latitude,
		Longitude: result.Longitude,
	}, nil
}
