package k8s

import (
	"context"
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
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

// GetIngress retrieves an Ingress resource
func (c *Client) GetIngress(ctx context.Context, namespace, name string) (*networkingv1.Ingress, error) {
	return c.clientset.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
}

// UpdateIngressStatus updates the status of an Ingress resource
func (c *Client) UpdateIngressStatus(ctx context.Context, ingress *networkingv1.Ingress) error {
	_, err := c.clientset.NetworkingV1().Ingresses(ingress.Namespace).UpdateStatus(ctx, ingress, metav1.UpdateOptions{})
	return err
}

// ListIngresses lists all Ingress resources in a namespace
func (c *Client) ListIngresses(ctx context.Context, namespace string) (*networkingv1.IngressList, error) {
	return c.clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
}

// WatchIngresses returns a watcher for Ingress resources
func (c *Client) WatchIngresses(ctx context.Context, namespace string) (watch.Interface, error) {
	return c.clientset.NetworkingV1().Ingresses(namespace).Watch(ctx, metav1.ListOptions{})
}

// SetIngressLoadBalancer updates the Ingress status with load balancer information
func (c *Client) SetIngressLoadBalancer(ctx context.Context, ingress *networkingv1.Ingress, hostname string) error {
	ingress.Status.LoadBalancer.Ingress = []networkingv1.IngressLoadBalancerIngress{
		{
			Hostname: hostname,
		},
	}
	
	return c.UpdateIngressStatus(ctx, ingress)
} 