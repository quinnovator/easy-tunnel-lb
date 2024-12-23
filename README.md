# Easy Tunnel Load Balancer

A Kubernetes controller that enables exposing cluster load balancer traffic through a VPS using WireGuard tunnels.

## Overview

`easy-tunnel-lb` is a Kubernetes controller that watches for Ingress resources and creates secure tunnels to expose their traffic through a remote VPS. This is particularly useful when:

- Running Kubernetes clusters in private networks
- Need to expose services without a cloud load balancer
- Want to reduce costs by using a single VPS for multiple services

## Features

- Automatic tunnel creation and management
- Ingress-based configuration
- WireGuard tunnel support
- Automatic status updates for Ingress resources
- Secure API key authentication

## Prerequisites

- Kubernetes cluster (1.21+)
- WireGuard tools installed on the cluster nodes
- A VPS running the `easy-tunnel-lb-agent` server component
- `kubectl` access to the cluster

## Installation

1. Create a namespace for the controller:

```bash
kubectl create namespace easy-tunnel-lb-system
```

2. Create a secret with your API key:

```bash
kubectl create secret generic easy-tunnel-lb-config \
  --namespace easy-tunnel-lb-system \
  --from-literal=api-key=your-api-key
```

3. Apply the RBAC configuration:

```bash
kubectl apply -f deploy/rbac.yaml
```

4. Deploy the controller:

```bash
kubectl apply -f deploy/deployment.yaml
```

## Usage

1. Add the required annotation to your Ingress resource:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-app
  annotations:
    easy-tunnel-lb.quinnovator.com/enabled: "true"
spec:
  rules:
    - host: myapp.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: my-app
                port:
                  number: 80
```

2. The controller will:
   - Detect the annotated Ingress
   - Request a tunnel from the server-side agent
   - Configure the local WireGuard tunnel
   - Update the Ingress status with the external IP/hostname

## Configuration

The controller can be configured using environment variables:

- `SERVER_URL`: URL of the server-side agent (required)
- `API_KEY`: API key for authentication (required)
- `LOG_LEVEL`: Logging level (default: "info")
- `WATCH_INTERVAL`: Interval for checking Ingress updates in seconds (default: 30)

## RBAC Permissions

The controller requires the following permissions:

- List and watch Ingress resources
- Update Ingress status
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
