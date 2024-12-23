package api_client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateTunnel(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "/api/tunnels", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		// Return mock response
		resp := &TunnelResponse{
			TunnelID:     "test-tunnel",
			ExternalHost: "test.example.com",
			Status:       StatusActive,
			WGConfig:     "test-config",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client
	client := NewClient(server.URL, "test-key")

	// Test request
	req := &TunnelRequest{
		IngressName:      "test-ingress",
		IngressNamespace: "default",
		Hostname:         "test.example.com",
		Ports:           []int{80, 443},
	}

	resp, err := client.CreateTunnel(req)
	assert.NoError(t, err)
	assert.Equal(t, "test-tunnel", resp.TunnelID)
	assert.Equal(t, "test.example.com", resp.ExternalHost)
	assert.Equal(t, StatusActive, resp.Status)
	assert.Equal(t, "test-config", resp.WGConfig)
}

func TestDeleteTunnel(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "/api/tunnels/test-tunnel", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client
	client := NewClient(server.URL, "test-key")

	// Test request
	err := client.DeleteTunnel("test-tunnel")
	assert.NoError(t, err)
}

func TestGetTunnelStatus(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "/api/tunnels/test-tunnel/status", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		// Return mock response
		resp := &TunnelStatus{
			TunnelID: "test-tunnel",
			Status:   StatusActive,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client
	client := NewClient(server.URL, "test-key")

	// Test request
	status, err := client.GetTunnelStatus("test-tunnel")
	assert.NoError(t, err)
	assert.Equal(t, "test-tunnel", status.TunnelID)
	assert.Equal(t, StatusActive, status.Status)
} 