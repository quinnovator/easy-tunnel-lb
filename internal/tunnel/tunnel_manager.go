package tunnel

import (
	"context"
	"fmt"
	"sync"
)

// TunnelConfig represents the configuration for a tunnel
type TunnelConfig struct {
	TunnelID string
	WGConfig string
}

// Manager manages the lifecycle of tunnels
type Manager struct {
	mu      sync.RWMutex
	tunnels map[string]*Tunnel
}

// NewManager creates a new tunnel manager
func NewManager() *Manager {
	return &Manager{
		tunnels: make(map[string]*Tunnel),
	}
}

// CreateTunnel creates and starts a new tunnel
func (m *Manager) CreateTunnel(ctx context.Context, config *TunnelConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tunnels[config.TunnelID]; exists {
		return fmt.Errorf("tunnel %s already exists", config.TunnelID)
	}

	tunnel, err := NewTunnel(config)
	if err != nil {
		return fmt.Errorf("failed to create tunnel: %w", err)
	}

	if err := tunnel.Start(ctx); err != nil {
		return fmt.Errorf("failed to start tunnel: %w", err)
	}

	m.tunnels[config.TunnelID] = tunnel
	return nil
}

// UpdateTunnel updates an existing tunnel's configuration
func (m *Manager) UpdateTunnel(ctx context.Context, config *TunnelConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tunnel, exists := m.tunnels[config.TunnelID]
	if !exists {
		return fmt.Errorf("tunnel %s not found", config.TunnelID)
	}

	if err := tunnel.Update(ctx, config); err != nil {
		return fmt.Errorf("failed to update tunnel: %w", err)
	}

	return nil
}

// DeleteTunnel stops and removes a tunnel
func (m *Manager) DeleteTunnel(ctx context.Context, tunnelID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tunnel, exists := m.tunnels[tunnelID]
	if !exists {
		return fmt.Errorf("tunnel %s not found", tunnelID)
	}

	if err := tunnel.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop tunnel: %w", err)
	}

	delete(m.tunnels, tunnelID)
	return nil
}

// GetTunnel retrieves a tunnel by ID
func (m *Manager) GetTunnel(tunnelID string) (*Tunnel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tunnel, exists := m.tunnels[tunnelID]
	if !exists {
		return nil, fmt.Errorf("tunnel %s not found", tunnelID)
	}

	return tunnel, nil
}

// ListTunnels returns all active tunnels
func (m *Manager) ListTunnels() []*Tunnel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tunnels := make([]*Tunnel, 0, len(m.tunnels))
	for _, t := range m.tunnels {
		tunnels = append(tunnels, t)
	}

	return tunnels
} 