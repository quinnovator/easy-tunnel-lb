package api_client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents an API client for the tunnel server
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new API client instance
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateTunnel sends a request to create a new tunnel
func (c *Client) CreateTunnel(req *TunnelRequest) (*TunnelResponse, error) {
	resp := &TunnelResponse{}
	err := c.doRequest("POST", "/api/tunnels", req, resp)
	if err != nil {
		return nil, fmt.Errorf("create tunnel request failed: %w", err)
	}
	return resp, nil
}

// UpdateTunnel updates an existing tunnel
func (c *Client) UpdateTunnel(tunnelID string, req *TunnelRequest) (*TunnelResponse, error) {
	resp := &TunnelResponse{}
	err := c.doRequest("PUT", fmt.Sprintf("/api/tunnels/%s", tunnelID), req, resp)
	if err != nil {
		return nil, fmt.Errorf("update tunnel request failed: %w", err)
	}
	return resp, nil
}

// DeleteTunnel removes an existing tunnel
func (c *Client) DeleteTunnel(tunnelID string) error {
	err := c.doRequest("DELETE", fmt.Sprintf("/api/tunnels/%s", tunnelID), nil, nil)
	if err != nil {
		return fmt.Errorf("delete tunnel request failed: %w", err)
	}
	return nil
}

// GetTunnelStatus retrieves the current status of a tunnel
func (c *Client) GetTunnelStatus(tunnelID string) (*TunnelStatus, error) {
	resp := &TunnelStatus{}
	err := c.doRequest("GET", fmt.Sprintf("/api/tunnels/%s/status", tunnelID), nil, resp)
	if err != nil {
		return nil, fmt.Errorf("get tunnel status failed: %w", err)
	}
	return resp, nil
}

func (c *Client) doRequest(method, path string, reqBody interface{}, respBody interface{}) error {
	var bodyReader io.Reader
	if reqBody != nil {
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if respBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
} 