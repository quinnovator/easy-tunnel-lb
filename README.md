# Easy Tunnel Load Balancer

A Kubernetes controller that enables exposing cluster load balancer traffic through a VPS using WireGuard tunnels.

## Overview

`easy-tunnel-lb` is a Kubernetes controller that watches for Service resources of type LoadBalancer and creates secure tunnels to expose their traffic through a remote VPS. This is particularly useful when:

- Running Kubernetes clusters in private networks
- Need to expose services without a cloud load balancer
- Want to reduce costs by using a single VPS for multiple services

## Features

- Automatic tunnel creation and management
- Service-based configuration (LoadBalancer type)
- WireGuard tunnel support
- Automatic status updates for Service resources
- Secure API key authentication

## Prerequisites

- Kubernetes cluster (1.21+)
- WireGuard tools installed on the cluster nodes
- A VPS running the `easy-tunnel-lb-agent` server component
- `kubectl` access to the cluster

## Installation

### Using Helm (Recommended)

1. Add the Helm repository:

```bash
helm repo add easy-tunnel-lb https://quinnovator.github.io/easy-tunnel-lb
helm repo update
```

2. Install the chart:

```bash
helm install easy-tunnel-lb easy-tunnel-lb/easy-tunnel-lb \
  --namespace easy-tunnel-lb-system \
  --create-namespace \
  --set config.apiKey=your-api-key
```

### Upgrading

To upgrade to the latest version of the chart:

```bash
# Update the repository
helm repo update

# Upgrade the installation
helm upgrade easy-tunnel-lb easy-tunnel-lb/easy-tunnel-lb --namespace easy-tunnel-lb-system
```

## Usage

1. Create a Service of type LoadBalancer with the required annotation:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-app
  annotations:
    easy-tunnel-lb.quinnovator.com/enabled: "true"
spec:
  type: LoadBalancer
  ports:
    - port: 80
      targetPort: 80
      protocol: TCP
  selector:
    app: my-app
```

2. The controller will:

    - Detect the annotated Service
    - Request a tunnel from the server-side agent
    - Configure the local WireGuard tunnel
    - Update the Service status with the external IP/hostname

## Configuration

The controller can be configured using environment variables:

- `SERVER_URL`: URL of the server-side agent (required)
- `API_KEY`: API key for authentication (required)
- `LOG_LEVEL`: Logging level (default: "info")
- `WATCH_INTERVAL`: Interval for checking Service updates in seconds (default: 30)

## RBAC Permissions

The controller requires the following permissions:

- List and watch Service resources
- Update Service status
- Create and manage ConfigMaps (for tunnel state)

See `deploy/rbac.yaml` for the complete RBAC configuration.

## Building

```bash
go build -o bin/easy-tunnel-lb cmd/main.go
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

AGPL-v3
