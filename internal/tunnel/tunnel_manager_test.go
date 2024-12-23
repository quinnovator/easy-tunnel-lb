package tunnel

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockCmd replaces exec.Command for testing
func mockCmd(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

// TestHelperProcess is not a real test, it's used to mock exec.Command
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	os.Exit(0)
}

func TestTunnelManager(t *testing.T) {
	// Replace exec.Command with our mock
	execCommand = mockCmd
	defer func() { execCommand = exec.Command }()

	ctx := context.Background()
	manager := NewManager()

	// Test creating a tunnel
	config := &TunnelConfig{
		TunnelID: "test-tunnel",
		WGConfig: "test-config",
	}

	err := manager.CreateTunnel(ctx, config)
	assert.NoError(t, err)

	// Test getting the tunnel
	tunnel, err := manager.GetTunnel("test-tunnel")
	assert.NoError(t, err)
	assert.NotNil(t, tunnel)
	assert.Equal(t, "test-tunnel", tunnel.id)
	assert.Equal(t, "test-config", tunnel.config)

	// Test updating the tunnel
	newConfig := &TunnelConfig{
		TunnelID: "test-tunnel",
		WGConfig: "new-config",
	}

	err = manager.UpdateTunnel(ctx, newConfig)
	assert.NoError(t, err)

	// Verify update
	tunnel, err = manager.GetTunnel("test-tunnel")
	assert.NoError(t, err)
	assert.Equal(t, "new-config", tunnel.config)

	// Test listing tunnels
	tunnels := manager.ListTunnels()
	assert.Len(t, tunnels, 1)
	assert.Equal(t, "test-tunnel", tunnels[0].id)

	// Test deleting the tunnel
	err = manager.DeleteTunnel(ctx, "test-tunnel")
	assert.NoError(t, err)

	// Verify deletion
	_, err = manager.GetTunnel("test-tunnel")
	assert.Error(t, err)
	assert.Empty(t, manager.ListTunnels())
}

func TestTunnelManagerErrors(t *testing.T) {
	// Replace exec.Command with our mock
	execCommand = mockCmd
	defer func() { execCommand = exec.Command }()

	ctx := context.Background()
	manager := NewManager()

	// Test getting non-existent tunnel
	_, err := manager.GetTunnel("non-existent")
	assert.Error(t, err)

	// Test updating non-existent tunnel
	err = manager.UpdateTunnel(ctx, &TunnelConfig{TunnelID: "non-existent"})
	assert.Error(t, err)

	// Test deleting non-existent tunnel
	err = manager.DeleteTunnel(ctx, "non-existent")
	assert.Error(t, err)

	// Test creating duplicate tunnel
	config := &TunnelConfig{TunnelID: "test-tunnel"}
	err = manager.CreateTunnel(ctx, config)
	assert.NoError(t, err)

	err = manager.CreateTunnel(ctx, config)
	assert.Error(t, err)
} 