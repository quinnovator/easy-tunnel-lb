package tunnel

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// execCommand allows us to replace exec.Command during testing
var execCommand = exec.Command

// Tunnel represents a WireGuard tunnel instance
type Tunnel struct {
	id     string
	config string
	cmd    *exec.Cmd
}

// NewTunnel creates a new WireGuard tunnel instance
func NewTunnel(config *TunnelConfig) (*Tunnel, error) {
	return &Tunnel{
		id:     config.TunnelID,
		config: config.WGConfig,
	}, nil
}

// Start initializes and starts the WireGuard tunnel
func (t *Tunnel) Start(ctx context.Context) error {
	configPath, err := t.writeConfig()
	if err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	cmd := execCommand("wg-quick", "up", configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start wg-quick: %w", err)
	}

	t.cmd = cmd
	return nil
}

// Stop terminates the WireGuard tunnel
func (t *Tunnel) Stop(ctx context.Context) error {
	if t.cmd != nil && t.cmd.Process != nil {
		if err := t.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill wg-quick process: %w", err)
		}
	}

	configPath := t.getConfigPath()
	cmd := execCommand("wg-quick", "down", configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop wg-quick: %w", err)
	}

	if err := os.Remove(configPath); err != nil {
		return fmt.Errorf("failed to remove config file: %w", err)
	}

	return nil
}

// Update updates the tunnel configuration
func (t *Tunnel) Update(ctx context.Context, config *TunnelConfig) error {
	if err := t.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop tunnel for update: %w", err)
	}

	t.config = config.WGConfig
	return t.Start(ctx)
}

// writeConfig writes the WireGuard configuration to a temporary file
func (t *Tunnel) writeConfig() (string, error) {
	configPath := t.getConfigPath()
	if err := os.WriteFile(configPath, []byte(t.config), 0600); err != nil {
		return "", fmt.Errorf("failed to write config file: %w", err)
	}
	return configPath, nil
}

// getConfigPath returns the path for the WireGuard configuration file
func (t *Tunnel) getConfigPath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("wg-%s.conf", t.id))
} 