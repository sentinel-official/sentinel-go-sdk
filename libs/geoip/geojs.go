package geoip

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Ensure GeoJSClient implements the Client interface.
var _ Client = (*GeoJSClient)(nil)

// GeoJSClient is a client for retrieving location data using the GeoJS API.
type GeoJSClient struct {
	c *http.Client
}

// NewGeoJSClient creates and returns a new instance of GeoJSClient with the specified timeout and optional proxy address.
func NewGeoJSClient(proxyAddr string, timeout time.Duration) (*GeoJSClient, error) {
	transport := &http.Transport{}

	// If a proxy address is provided, configure the HTTP client to use it.
	if proxyAddr != "" {
		proxyURL, err := url.Parse(proxyAddr)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy addr: %w", err)
		}

		transport.Proxy = http.ProxyURL(proxyURL)
	}

	return &GeoJSClient{
		c: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}, nil
}

// Get retrieves location data for the specified IP address using the GeoJS API.
func (c *GeoJSClient) Get(ip string) (*Location, error) {
	// Construct the URL for the API request. Use the provided IP address if it is not empty.
	apiURL := "https://get.geojs.io/v1/ip/geo.json"
	if ip != "" {
		apiURL = fmt.Sprintf("https://get.geojs.io/v1/ip/geo/%s.json", ip)
	}

	// Make the HTTP GET request to the GeoJS API.
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
		City      string `json:"city"`
		Country   string `json:"country"`
		IP        string `json:"ip"`
		Latitude  string `json:"latitude"`
		Longitude string `json:"longitude"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Convert latitude and longitude from string to float.
	latitude, err := strconv.ParseFloat(result.Latitude, 64)
	if err != nil {
		return nil, err
	}
	longitude, err := strconv.ParseFloat(result.Longitude, 64)
	if err != nil {
		return nil, err
	}

	// Return the location information as a Location struct.
	return &Location{
		City:      result.City,
		Country:   result.Country,
		IP:        result.IP,
		Latitude:  latitude,
		Longitude: longitude,
	}, nil
}
