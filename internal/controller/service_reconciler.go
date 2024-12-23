package controller

import (
	"context"
	"fmt"

	"github.com/quinnovator/easy-tunnel-lb/internal/api_client"
	"github.com/quinnovator/easy-tunnel-lb/internal/tunnel"
	"github.com/quinnovator/easy-tunnel-lb/internal/utils"
	v1 "k8s.io/api/core/v1"
)

// K8sClient interface for Kubernetes operations on Services
type K8sClient interface {
	SetServiceLoadBalancer(ctx context.Context, svc *v1.Service, externalIP, externalHost string) error
}

// APIClient interface for tunnel server operations
type APIClient interface {
	CreateTunnel(req *api_client.TunnelRequest) (*api_client.TunnelResponse, error)
	UpdateTunnel(tunnelID string, req *api_client.TunnelRequest) (*api_client.TunnelResponse, error)
	DeleteTunnel(tunnelID string) error
}

// TunnelManager interface for local WireGuard tunnel operations
type TunnelManager interface {
	CreateTunnel(ctx context.Context, config *tunnel.TunnelConfig) error
	UpdateTunnel(ctx context.Context, config *tunnel.TunnelConfig) error
	DeleteTunnel(ctx context.Context, tunnelID string) error
}

// ServiceReconciler handles the reconciliation of a Service resource
type ServiceReconciler struct {
	k8sClient  K8sClient
	apiClient  APIClient
	tunnelMgr  TunnelManager
	logger     *utils.Logger
}

func NewServiceReconciler(k8sClient K8sClient, apiClient APIClient, tunnelMgr TunnelManager, logger *utils.Logger) *ServiceReconciler {
	return &ServiceReconciler{
		k8sClient: k8sClient,
		apiClient: apiClient,
		tunnelMgr: tunnelMgr,
		logger:    logger,
	}
}

// Reconcile ensures the tunnel is created/updated for the given service
func (r *ServiceReconciler) Reconcile(ctx context.Context, svc *v1.Service) error {
	// Retrieve or create the tunnel
	tunnelID := svc.Annotations["easy-tunnel-lb.quinnovator.com/tunnel-id"]
	ports := []int{}

	for _, sp := range svc.Spec.Ports {
		ports = append(ports, int(sp.Port))
	}

	req := &api_client.TunnelRequest{
		IngressName:      svc.Name,
		IngressNamespace: svc.Namespace,
		Hostname:         "", // not relevant for service LB, but left in for API
		Ports:            ports,
		Annotations:      svc.Annotations,
	}

	var resp *api_client.TunnelResponse
	var err error

	if tunnelID == "" {
		// create
		resp, err = r.apiClient.CreateTunnel(req)
		if err != nil {
			return fmt.Errorf("failed to create tunnel: %w", err)
		}
	} else {
		// update
		resp, err = r.apiClient.UpdateTunnel(tunnelID, req)
		if err != nil {
			return fmt.Errorf("failed to update tunnel: %w", err)
		}
	}

	// Configure local WireGuard tunnel
	tunnelConfig := &tunnel.TunnelConfig{
		TunnelID: resp.TunnelID,
		WGConfig: resp.WGConfig,
	}

	if tunnelID == "" {
		if err := r.tunnelMgr.CreateTunnel(ctx, tunnelConfig); err != nil {
			return fmt.Errorf("failed to create local wireguard tunnel: %w", err)
		}
	} else {
		if err := r.tunnelMgr.UpdateTunnel(ctx, tunnelConfig); err != nil {
			return fmt.Errorf("failed to update local wireguard tunnel: %w", err)
		}
	}

	// Update the Service's status.loadBalancer with external IP or host
	err = r.k8sClient.SetServiceLoadBalancer(ctx, svc, resp.ExternalIP, resp.ExternalHost)
	if err != nil {
		return fmt.Errorf("failed to update service loadbalancer: %w", err)
	}

	return nil
}

// HandleDelete ensures the tunnel is removed when the Service is deleted
func (r *ServiceReconciler) HandleDelete(ctx context.Context, svc *v1.Service) error {
	tunnelID := svc.Annotations["easy-tunnel-lb.quinnovator.com/tunnel-id"]
	if tunnelID == "" {
		return nil
	}

	if err := r.apiClient.DeleteTunnel(tunnelID); err != nil {
		return fmt.Errorf("failed to delete tunnel from server: %w", err)
	}

	if err := r.tunnelMgr.DeleteTunnel(ctx, tunnelID); err != nil {
		return fmt.Errorf("failed to delete local wireguard tunnel: %w", err)
	}
	return nil
}