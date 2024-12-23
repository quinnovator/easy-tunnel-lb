package controller

import (
	"context"
	"fmt"

	"github.com/quinnovator/easy-tunnel-lb/internal/api_client"
	"github.com/quinnovator/easy-tunnel-lb/internal/tunnel"
	"github.com/quinnovator/easy-tunnel-lb/internal/utils"
	networkingv1 "k8s.io/api/networking/v1"
)

// K8sClient interface for Kubernetes operations
type K8sClient interface {
	SetIngressLoadBalancer(ctx context.Context, ingress *networkingv1.Ingress, hostname string) error
}

// APIClient interface for API operations
type APIClient interface {
	CreateTunnel(req *api_client.TunnelRequest) (*api_client.TunnelResponse, error)
	UpdateTunnel(tunnelID string, req *api_client.TunnelRequest) (*api_client.TunnelResponse, error)
	DeleteTunnel(tunnelID string) error
}

// TunnelManager interface for tunnel operations
type TunnelManager interface {
	CreateTunnel(ctx context.Context, config *tunnel.TunnelConfig) error
	UpdateTunnel(ctx context.Context, config *tunnel.TunnelConfig) error
	DeleteTunnel(ctx context.Context, tunnelID string) error
}

// Reconciler handles the reconciliation of Ingress resources
type Reconciler struct {
	k8sClient K8sClient
	apiClient APIClient
	tunnelMgr TunnelManager
	logger    *utils.Logger
}

// NewReconciler creates a new reconciler instance
func NewReconciler(k8sClient K8sClient, apiClient APIClient, tunnelMgr TunnelManager, logger *utils.Logger) *Reconciler {
	return &Reconciler{
		k8sClient: k8sClient,
		apiClient: apiClient,
		tunnelMgr: tunnelMgr,
		logger:    logger,
	}
}

// Reconcile handles the reconciliation of an Ingress resource
func (r *Reconciler) Reconcile(ctx context.Context, ingress *networkingv1.Ingress) error {
	if _, ok := ingress.Annotations[TunnelAnnotation]; !ok {
		return nil
	}

	// Extract ports from ingress rules
	var ports []int
	for _, rule := range ingress.Spec.Rules {
		if rule.HTTP != nil {
			for _, path := range rule.HTTP.Paths {
				if path.Backend.Service != nil && path.Backend.Service.Port.Number > 0 {
					ports = append(ports, int(path.Backend.Service.Port.Number))
				}
			}
		}
	}

	// Create tunnel request
	req := &api_client.TunnelRequest{
		IngressName:      ingress.Name,
		IngressNamespace: ingress.Namespace,
		Hostname:         ingress.Spec.Rules[0].Host,
		Ports:           ports,
		Annotations:     ingress.Annotations,
	}

	// Get existing tunnel ID from annotations
	tunnelID := ingress.Annotations["easy-tunnel-lb.quinnovator.com/tunnel-id"]
	var resp *api_client.TunnelResponse
	var err error

	if tunnelID == "" {
		// Create new tunnel
		resp, err = r.apiClient.CreateTunnel(req)
		if err != nil {
			return fmt.Errorf("failed to create tunnel: %w", err)
		}
	} else {
		// Update existing tunnel
		resp, err = r.apiClient.UpdateTunnel(tunnelID, req)
		if err != nil {
			return fmt.Errorf("failed to update tunnel: %w", err)
		}
	}

	// Configure local tunnel
	tunnelConfig := &tunnel.TunnelConfig{
		TunnelID: resp.TunnelID,
		WGConfig: resp.WGConfig,
	}

	if tunnelID == "" {
		if err := r.tunnelMgr.CreateTunnel(ctx, tunnelConfig); err != nil {
			return fmt.Errorf("failed to create local tunnel: %w", err)
		}
	} else {
		if err := r.tunnelMgr.UpdateTunnel(ctx, tunnelConfig); err != nil {
			return fmt.Errorf("failed to update local tunnel: %w", err)
		}
	}

	// Update ingress status with external hostname/IP
	if resp.ExternalHost != "" {
		if err := r.k8sClient.SetIngressLoadBalancer(ctx, ingress, resp.ExternalHost); err != nil {
			return fmt.Errorf("failed to update ingress status: %w", err)
		}
	}

	return nil
}

// HandleDelete handles the deletion of an Ingress resource
func (r *Reconciler) HandleDelete(ctx context.Context, ingress *networkingv1.Ingress) error {
	tunnelID := ingress.Annotations["easy-tunnel-lb.quinnovator.com/tunnel-id"]
	if tunnelID == "" {
		return nil
	}

	if err := r.apiClient.DeleteTunnel(tunnelID); err != nil {
		return fmt.Errorf("failed to delete tunnel from server: %w", err)
	}

	if err := r.tunnelMgr.DeleteTunnel(ctx, tunnelID); err != nil {
		return fmt.Errorf("failed to delete local tunnel: %w", err)
	}

	return nil
} 