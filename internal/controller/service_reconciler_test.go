package controller

import (
	"context"
	"testing"

	"github.com/quinnovator/easy-tunnel-lb/internal/api_client"
	"github.com/quinnovator/easy-tunnel-lb/internal/tunnel"
	"github.com/quinnovator/easy-tunnel-lb/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Mock implementations
type MockK8sClient struct {
	mock.Mock
}

func (m *MockK8sClient) SetServiceLoadBalancer(ctx context.Context, svc *v1.Service, externalIP, externalHost string) error {
	args := m.Called(ctx, svc, externalIP, externalHost)
	return args.Error(0)
}

type MockAPIClient struct {
	mock.Mock
}

func (m *MockAPIClient) CreateTunnel(req *api_client.TunnelRequest) (*api_client.TunnelResponse, error) {
	args := m.Called(req)
	if resp := args.Get(0); resp != nil {
		return resp.(*api_client.TunnelResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAPIClient) UpdateTunnel(tunnelID string, req *api_client.TunnelRequest) (*api_client.TunnelResponse, error) {
	args := m.Called(tunnelID, req)
	if resp := args.Get(0); resp != nil {
		return resp.(*api_client.TunnelResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAPIClient) DeleteTunnel(tunnelID string) error {
	args := m.Called(tunnelID)
	return args.Error(0)
}

type MockTunnelManager struct {
	mock.Mock
}

func (m *MockTunnelManager) CreateTunnel(ctx context.Context, config *tunnel.TunnelConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockTunnelManager) UpdateTunnel(ctx context.Context, config *tunnel.TunnelConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockTunnelManager) DeleteTunnel(ctx context.Context, tunnelID string) error {
	args := m.Called(ctx, tunnelID)
	return args.Error(0)
}

func TestServiceReconciler_Reconcile(t *testing.T) {
	tests := []struct {
		name    string
		service *v1.Service
		setup   func(*MockK8sClient, *MockAPIClient, *MockTunnelManager)
		wantErr bool
	}{
		{
			name: "successfully create new tunnel",
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: "default",
					Annotations: map[string]string{
						"some-annotation": "value",
					},
				},
				Spec: v1.ServiceSpec{
					Ports: []v1.ServicePort{
						{Port: 80},
						{Port: 443},
					},
				},
			},
			setup: func(k8s *MockK8sClient, api *MockAPIClient, tm *MockTunnelManager) {
				expectedReq := &api_client.TunnelRequest{
					IngressName:      "test-service",
					IngressNamespace: "default",
					Hostname:         "",
					Ports:           []int{80, 443},
					Annotations: map[string]string{
						"some-annotation": "value",
					},
				}
				
				resp := &api_client.TunnelResponse{
					TunnelID:     "new-tunnel-id",
					ExternalIP:   "1.2.3.4",
					ExternalHost: "test.example.com",
					WGConfig:     "test-config",
				}
				
				api.On("CreateTunnel", expectedReq).Return(resp, nil)
				
				tm.On("CreateTunnel", mock.Anything, &tunnel.TunnelConfig{
					TunnelID: "new-tunnel-id",
					WGConfig: "test-config",
				}).Return(nil)
				
				k8s.On("SetServiceLoadBalancer", 
					mock.Anything, 
					mock.AnythingOfType("*v1.Service"),
					"1.2.3.4",
					"test.example.com",
				).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "successfully update existing tunnel",
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: "default",
					Annotations: map[string]string{
						"easy-tunnel-lb.quinnovator.com/tunnel-id": "existing-tunnel-id",
					},
				},
				Spec: v1.ServiceSpec{
					Ports: []v1.ServicePort{
						{Port: 80},
					},
				},
			},
			setup: func(k8s *MockK8sClient, api *MockAPIClient, tm *MockTunnelManager) {
				expectedReq := &api_client.TunnelRequest{
					IngressName:      "test-service",
					IngressNamespace: "default",
					Hostname:         "",
					Ports:           []int{80},
					Annotations: map[string]string{
						"easy-tunnel-lb.quinnovator.com/tunnel-id": "existing-tunnel-id",
					},
				}
				
				resp := &api_client.TunnelResponse{
					TunnelID:     "existing-tunnel-id",
					ExternalIP:   "5.6.7.8",
					ExternalHost: "test2.example.com",
					WGConfig:     "updated-config",
				}
				
				api.On("UpdateTunnel", "existing-tunnel-id", expectedReq).Return(resp, nil)
				
				tm.On("UpdateTunnel", mock.Anything, &tunnel.TunnelConfig{
					TunnelID: "existing-tunnel-id",
					WGConfig: "updated-config",
				}).Return(nil)
				
				k8s.On("SetServiceLoadBalancer", 
					mock.Anything, 
					mock.AnythingOfType("*v1.Service"),
					"5.6.7.8",
					"test2.example.com",
				).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sMock := &MockK8sClient{}
			apiMock := &MockAPIClient{}
			tunnelMock := &MockTunnelManager{}
			
			tt.setup(k8sMock, apiMock, tunnelMock)
			
			reconciler := NewServiceReconciler(
				k8sMock,
				apiMock,
				tunnelMock,
				utils.NewLogger("test"),
			)
			
			err := reconciler.Reconcile(context.Background(), tt.service)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			
			k8sMock.AssertExpectations(t)
			apiMock.AssertExpectations(t)
			tunnelMock.AssertExpectations(t)
		})
	}
}

func TestServiceReconciler_HandleDelete(t *testing.T) {
	tests := []struct {
		name    string
		service *v1.Service
		setup   func(*MockAPIClient, *MockTunnelManager)
		wantErr bool
	}{
		{
			name: "successfully delete tunnel",
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"easy-tunnel-lb.quinnovator.com/tunnel-id": "tunnel-to-delete",
					},
				},
			},
			setup: func(api *MockAPIClient, tm *MockTunnelManager) {
				api.On("DeleteTunnel", "tunnel-to-delete").Return(nil)
				tm.On("DeleteTunnel", mock.Anything, "tunnel-to-delete").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "no tunnel ID - no action needed",
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			setup: func(api *MockAPIClient, tm *MockTunnelManager) {
				// No expectations - nothing should be called
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiMock := &MockAPIClient{}
			tunnelMock := &MockTunnelManager{}
			
			tt.setup(apiMock, tunnelMock)
			
			reconciler := NewServiceReconciler(
				&MockK8sClient{},
				apiMock,
				tunnelMock,
				utils.NewLogger("test"),
			)
			
			err := reconciler.HandleDelete(context.Background(), tt.service)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			
			apiMock.AssertExpectations(t)
			tunnelMock.AssertExpectations(t)
		})
	}
} 