package k8s

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	kubernetes "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Client wraps the Kubernetes client-go functionality
type Client struct {
	clientset *kubernetes.Clientset
}

// NewClient creates a new Kubernetes client
func NewClient() (*Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return &Client{
		clientset: clientset,
	}, nil
}

// ListServices lists all Services in the given namespace. If namespace is "", it lists across all namespaces.
func (c *Client) ListServices(ctx context.Context, namespace string, opts metav1.ListOptions) (*v1.ServiceList, error) {
	return c.clientset.CoreV1().Services(namespace).List(ctx, opts)
}

// WatchServices sets up a watch on Services in the given namespace. If namespace is "", it watches across all namespaces.
func (c *Client) WatchServices(ctx context.Context, namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	return c.clientset.CoreV1().Services(namespace).Watch(ctx, opts)
}

// GetService retrieves a specific Service
func (c *Client) GetService(ctx context.Context, namespace, name string) (*v1.Service, error) {
	svc, err := c.clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return svc, nil
}

// UpdateServiceStatus updates the status of the given Service
func (c *Client) UpdateServiceStatus(ctx context.Context, svc *v1.Service) error {
	_, err := c.clientset.CoreV1().Services(svc.Namespace).UpdateStatus(ctx, svc, metav1.UpdateOptions{})
	return err
}

// SetServiceLoadBalancer updates the given Service's status.loadBalancer with an external IP or external hostname
func (c *Client) SetServiceLoadBalancer(ctx context.Context, svc *v1.Service, externalIP, externalHost string) error {
	loadBalancerIngress := []v1.LoadBalancerIngress{}

	if externalIP != "" {
		loadBalancerIngress = append(loadBalancerIngress, v1.LoadBalancerIngress{
			IP: externalIP,
		})
	}
	if externalHost != "" {
		loadBalancerIngress = append(loadBalancerIngress, v1.LoadBalancerIngress{
			Hostname: externalHost,
		})
	}

	svc.Status.LoadBalancer.Ingress = loadBalancerIngress

	if err := c.UpdateServiceStatus(ctx, svc); err != nil {
		return fmt.Errorf("failed to update service loadbalancer status: %w", err)
	}
	return nil
} 