package controller

import (
	"context"
	"testing"

	"github.com/quinnovator/easy-tunnel-lb/internal/api_client"
	"github.com/quinnovator/easy-tunnel-lb/internal/tunnel"
	"github.com/quinnovator/easy-tunnel-lb/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Mock implementations
type mockK8sClient struct {
	mock.Mock
}

func (m *mockK8sClient) SetIngressLoadBalancer(ctx context.Context, ingress *networkingv1.Ingress, hostname string) error {
	args := m.Called(ctx, ingress, hostname)
	return args.Error(0)
}

type mockAPIClient struct {
	mock.Mock
}

func (m *mockAPIClient) CreateTunnel(req *api_client.TunnelRequest) (*api_client.TunnelResponse, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*api_client.TunnelResponse), args.Error(1)
}

func (m *mockAPIClient) UpdateTunnel(tunnelID string, req *api_client.TunnelRequest) (*api_client.TunnelResponse, error) {
	args := m.Called(tunnelID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*api_client.TunnelResponse), args.Error(1)
}

func (m *mockAPIClient) DeleteTunnel(tunnelID string) error {
	args := m.Called(tunnelID)
	return args.Error(0)
}

type mockTunnelManager struct {
	mock.Mock
}

func (m *mockTunnelManager) CreateTunnel(ctx context.Context, config *tunnel.TunnelConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *mockTunnelManager) UpdateTunnel(ctx context.Context, config *tunnel.TunnelConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *mockTunnelManager) DeleteTunnel(ctx context.Context, tunnelID string) error {
	args := m.Called(ctx, tunnelID)
	return args.Error(0)
}

func TestReconcile(t *testing.T) {
	// Create test ingress
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "default",
			Annotations: map[string]string{
				TunnelAnnotation: "true",
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "test.example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Create mocks
	k8sClient := &mockK8sClient{}
	apiClient := &mockAPIClient{}
	tunnelMgr := &mockTunnelManager{}
	logger := utils.NewLogger("debug")

	// Set up expectations
	apiClient.On("CreateTunnel", mock.Anything).Return(&api_client.TunnelResponse{
		TunnelID:     "test-tunnel",
		ExternalHost: "test.example.com",
		Status:       api_client.StatusActive,
		WGConfig:     "test-config",
	}, nil)

	tunnelMgr.On("CreateTunnel", mock.Anything, mock.Anything).Return(nil)
	k8sClient.On("SetIngressLoadBalancer", mock.Anything, mock.Anything, "test.example.com").Return(nil)

	// Create reconciler
	reconciler := NewReconciler(k8sClient, apiClient, tunnelMgr, logger)

	// Test reconciliation
	err := reconciler.Reconcile(context.Background(), ingress)
	assert.NoError(t, err)

	// Verify expectations
	k8sClient.AssertExpectations(t)
	apiClient.AssertExpectations(t)
	tunnelMgr.AssertExpectations(t)
}

func TestHandleDelete(t *testing.T) {
	// Create test ingress
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				TunnelAnnotation:                           "true",
				"easy-tunnel-lb.quinnovator.com/tunnel-id": "test-tunnel",
			},
		},
	}

	// Create mocks
	k8sClient := &mockK8sClient{}
	apiClient := &mockAPIClient{}
	tunnelMgr := &mockTunnelManager{}
	logger := utils.NewLogger("debug")

	// Set up expectations
	apiClient.On("DeleteTunnel", "test-tunnel").Return(nil)
	tunnelMgr.On("DeleteTunnel", mock.Anything, "test-tunnel").Return(nil)

	// Create reconciler
	reconciler := NewReconciler(k8sClient, apiClient, tunnelMgr, logger)

	// Test deletion
	err := reconciler.HandleDelete(context.Background(), ingress)
	assert.NoError(t, err)

	// Verify expectations
	apiClient.AssertExpectations(t)
	tunnelMgr.AssertExpectations(t)
} 