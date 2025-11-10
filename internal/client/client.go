package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/armyknifelabs-platform/seip-cli/internal/config"
)

// Client is an HTTP client for the SEIP API
type Client struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient(cfg *config.Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success  bool            `json:"success"`
	Data     json.RawMessage `json:"data"`
	Error    *APIError       `json:"error,omitempty"`
	Metadata *APIMetadata    `json:"metadata,omitempty"`
}

// APIError represents an API error
type APIError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// APIMetadata represents API response metadata
type APIMetadata struct {
	Timestamp string `json:"timestamp"`
	RequestID string `json:"requestId,omitempty"`
	Source    string `json:"source,omitempty"`
}

// Get performs a GET request
func (c *Client) Get(path string) (*APIResponse, error) {
	return c.request("GET", path, nil)
}

// Post performs a POST request
func (c *Client) Post(path string, body interface{}) (*APIResponse, error) {
	return c.request("POST", path, body)
}

// Delete performs a DELETE request
func (c *Client) Delete(path string) (*APIResponse, error) {
	return c.request("DELETE", path, nil)
}

// request performs an HTTP request
func (c *Client) request(method, path string, body interface{}) (*APIResponse, error) {
	url := fmt.Sprintf("%s%s", c.cfg.APIURL, path)

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.cfg.AccessToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.cfg.AccessToken))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !apiResp.Success && apiResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", apiResp.Error.Message)
	}

	return &apiResp, nil
}
