package controller

import (
	"context"
	"testing"
	"time"

	"github.com/quinnovator/easy-tunnel-lb/internal/api_client"
	"github.com/quinnovator/easy-tunnel-lb/internal/tunnel"
	"github.com/quinnovator/easy-tunnel-lb/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type mockK8sClient struct {
	mock.Mock
}

func (m *mockK8sClient) ListServices(ctx context.Context, namespace string, opts metav1.ListOptions) (*v1.ServiceList, error) {
	args := m.Called(ctx, namespace, opts)
	if list := args.Get(0); list != nil {
		return list.(*v1.ServiceList), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockK8sClient) WatchServices(ctx context.Context, namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	args := m.Called(ctx, namespace, opts)
	if w := args.Get(0); w != nil {
		return w.(watch.Interface), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockK8sClient) GetService(ctx context.Context, namespace, name string) (*v1.Service, error) {
	args := m.Called(ctx, namespace, name)
	if svc := args.Get(0); svc != nil {
		return svc.(*v1.Service), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockK8sClient) SetServiceLoadBalancer(ctx context.Context, svc *v1.Service, externalIP, externalHost string) error {
	args := m.Called(ctx, svc, externalIP, externalHost)
	return args.Error(0)
}

type mockWatcher struct {
	mock.Mock
	resultChan chan watch.Event
}

func newMockWatcher() *mockWatcher {
	return &mockWatcher{
		resultChan: make(chan watch.Event, 10),
	}
}

func (m *mockWatcher) Stop() {
	close(m.resultChan)
}

func (m *mockWatcher) ResultChan() <-chan watch.Event {
	return m.resultChan
}

func TestServiceWatcher_HandleService(t *testing.T) {
	tests := []struct {
		name           string
		service        *v1.Service
		shouldProcess  bool
		setupMocks     func(*mockK8sClient, *MockAPIClient, *MockTunnelManager)
		expectedError  bool
	}{
		{
			name: "process LoadBalancer service with annotation",
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: "default",
					Annotations: map[string]string{
						TunnelAnnotation: "true",
					},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
					Ports: []v1.ServicePort{
						{Port: 80},
					},
				},
			},
			shouldProcess: true,
			setupMocks: func(k8s *mockK8sClient, api *MockAPIClient, tm *MockTunnelManager) {
				testSvc := &v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-service",
						Namespace: "default",
						Annotations: map[string]string{
							TunnelAnnotation: "true",
						},
					},
					Spec: v1.ServiceSpec{
						Type: v1.ServiceTypeLoadBalancer,
						Ports: []v1.ServicePort{
							{Port: 80},
						},
					},
				}

				// Mock for ListServices call when starting the watcher
				k8s.On("ListServices", mock.Anything, "", mock.Anything).
					Return(&v1.ServiceList{Items: []v1.Service{}}, nil)

				// Mock for WatchServices call when starting the watcher
				mockWatcher := newMockWatcher()
				k8s.On("WatchServices", mock.Anything, "", mock.Anything).
					Return(mockWatcher, nil)

				// Mock for the initial GetService call in processNextWorkItem
				k8s.On("GetService", mock.Anything, "default", "test-service").
					Return(testSvc, nil)

				// Mock for the Reconcile call
				api.On("CreateTunnel", &api_client.TunnelRequest{
					IngressName:      "test-service",
					IngressNamespace: "default",
					Hostname:         "",
					Ports:           []int{80},
					Annotations: map[string]string{
						TunnelAnnotation: "true",
					},
				}).Return(
					&api_client.TunnelResponse{
						TunnelID:     "test-tunnel",
						ExternalIP:   "1.2.3.4",
						ExternalHost: "test.example.com",
						WGConfig:     "test-config",
					}, nil)

				tm.On("CreateTunnel", mock.Anything, &tunnel.TunnelConfig{
					TunnelID: "test-tunnel",
					WGConfig: "test-config",
				}).Return(nil)

				k8s.On("SetServiceLoadBalancer", mock.Anything, testSvc, "1.2.3.4", "test.example.com").Return(nil)
			},
			expectedError: false,
		},
		{
			name: "ignore non-LoadBalancer service",
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: "default",
					Annotations: map[string]string{
						TunnelAnnotation: "true",
					},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeClusterIP,
				},
			},
			shouldProcess: false,
			setupMocks: func(k8s *mockK8sClient, api *MockAPIClient, tm *MockTunnelManager) {
				// Mock for ListServices call when starting the watcher
				k8s.On("ListServices", mock.Anything, "", mock.Anything).
					Return(&v1.ServiceList{Items: []v1.Service{}}, nil)

				// Mock for WatchServices call when starting the watcher
				mockWatcher := newMockWatcher()
				k8s.On("WatchServices", mock.Anything, "", mock.Anything).
					Return(mockWatcher, nil)
			},
			expectedError: false,
		},
		{
			name: "ignore LoadBalancer service without annotation",
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: "default",
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
				},
			},
			shouldProcess: false,
			setupMocks: func(k8s *mockK8sClient, api *MockAPIClient, tm *MockTunnelManager) {
				// Mock for ListServices call when starting the watcher
				k8s.On("ListServices", mock.Anything, "", mock.Anything).
					Return(&v1.ServiceList{Items: []v1.Service{}}, nil)

				// Mock for WatchServices call when starting the watcher
				mockWatcher := newMockWatcher()
				k8s.On("WatchServices", mock.Anything, "", mock.Anything).
					Return(mockWatcher, nil)
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sMock := &mockK8sClient{}
			apiMock := &MockAPIClient{}
			tunnelMock := &MockTunnelManager{}
			
			reconciler := NewServiceReconciler(k8sMock, apiMock, tunnelMock, utils.NewLogger("test"))
			
			tt.setupMocks(k8sMock, apiMock, tunnelMock)
			
			watcher := NewServiceWatcher(k8sMock, reconciler, utils.NewLogger("test"))
			
			// Start the worker goroutine
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			
			go func() {
				err := watcher.Start(ctx)
				assert.NoError(t, err)
			}()
			
			// Give the worker goroutine time to start
			time.Sleep(100 * time.Millisecond)
			
			// Test handleService
			watcher.handleService(tt.service)
			
			// Give the worker goroutine time to process
			time.Sleep(100 * time.Millisecond)
			
			if tt.shouldProcess {
				k8sMock.AssertExpectations(t)
				apiMock.AssertExpectations(t)
				tunnelMock.AssertExpectations(t)
			}
		})
	}
}

func TestServiceWatcher_HandleDelete(t *testing.T) {
	tests := []struct {
		name          string
		service       *v1.Service
		shouldProcess bool
		setupMocks    func(*mockK8sClient, *MockAPIClient, *MockTunnelManager)
	}{
		{
			name: "handle delete for LoadBalancer service with annotation",
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: "default",
					Annotations: map[string]string{
						TunnelAnnotation: "true",
						"easy-tunnel-lb.quinnovator.com/tunnel-id": "test-tunnel",
					},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
				},
			},
			shouldProcess: true,
			setupMocks: func(k8s *mockK8sClient, api *MockAPIClient, tm *MockTunnelManager) {
				api.On("DeleteTunnel", "test-tunnel").Return(nil)
				tm.On("DeleteTunnel", mock.Anything, "test-tunnel").Return(nil)
			},
		},
		{
			name: "ignore delete for non-LoadBalancer service",
			service: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: "default",
					Annotations: map[string]string{
						TunnelAnnotation: "true",
					},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeClusterIP,
				},
			},
			shouldProcess: false,
			setupMocks: func(k8s *mockK8sClient, api *MockAPIClient, tm *MockTunnelManager) {
				// No mocks needed as service should be ignored
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sMock := &mockK8sClient{}
			apiMock := &MockAPIClient{}
			tunnelMock := &MockTunnelManager{}
			
			reconciler := NewServiceReconciler(k8sMock, apiMock, tunnelMock, utils.NewLogger("test"))
			
			tt.setupMocks(k8sMock, apiMock, tunnelMock)
			
			watcher := NewServiceWatcher(k8sMock, reconciler, utils.NewLogger("test"))
			
			// Test handleServiceDelete
			watcher.handleServiceDelete(tt.service)
			
			if tt.shouldProcess {
				apiMock.AssertExpectations(t)
				tunnelMock.AssertExpectations(t)
			}
		})
	}
}

func TestServiceWatcher_Start(t *testing.T) {
	k8sMock := &mockK8sClient{}
	apiMock := &MockAPIClient{}
	tunnelMock := &MockTunnelManager{}
	mockWatcher := newMockWatcher()
	
	reconciler := NewServiceReconciler(k8sMock, apiMock, tunnelMock, utils.NewLogger("test"))
	
	// Setup expectations
	k8sMock.On("ListServices", mock.Anything, "", mock.Anything).
		Return(&v1.ServiceList{Items: []v1.Service{}}, nil)
	k8sMock.On("WatchServices", mock.Anything, "", mock.Anything).
		Return(mockWatcher, nil)
	
	testSvc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
			Annotations: map[string]string{
				TunnelAnnotation: "true",
			},
		},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceTypeLoadBalancer,
			Ports: []v1.ServicePort{
				{Port: 80},
			},
		},
	}
	
	// Setup expectation for GetService call that will happen in processNextWorkItem
	k8sMock.On("GetService", mock.Anything, "default", "test-service").
		Return(testSvc, nil)
	
	// Setup expectations for Reconcile
	apiMock.On("CreateTunnel", &api_client.TunnelRequest{
		IngressName:      "test-service",
		IngressNamespace: "default",
		Hostname:         "",
		Ports:           []int{80},
		Annotations: map[string]string{
			TunnelAnnotation: "true",
		},
	}).Return(
		&api_client.TunnelResponse{
			TunnelID:     "test-tunnel",
			ExternalIP:   "1.2.3.4",
			ExternalHost: "test.example.com",
			WGConfig:     "test-config",
		}, nil)
	
	tunnelMock.On("CreateTunnel", mock.Anything, &tunnel.TunnelConfig{
		TunnelID: "test-tunnel",
		WGConfig: "test-config",
	}).Return(nil)
	
	k8sMock.On("SetServiceLoadBalancer", mock.Anything, testSvc, "1.2.3.4", "test.example.com").Return(nil)
	
	watcher := NewServiceWatcher(k8sMock, reconciler, utils.NewLogger("test"))
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Start the watcher in a goroutine
	go func() {
		err := watcher.Start(ctx)
		assert.NoError(t, err)
	}()
	
	// Give it time to start
	time.Sleep(100 * time.Millisecond)
	
	mockWatcher.resultChan <- watch.Event{
		Type:   watch.Added,
		Object: testSvc,
	}
	
	// Give it time to process
	time.Sleep(100 * time.Millisecond)
	
	// Cleanup
	cancel()
	mockWatcher.Stop()
	
	k8sMock.AssertExpectations(t)
	apiMock.AssertExpectations(t)
	tunnelMock.AssertExpectations(t)
} 