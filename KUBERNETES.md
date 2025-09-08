# Kubernetes

## Deploying a k8s cluster
```shell
DOCKER_HOST=tcp://localhost:2377 docker compose -f configs/cluster.yaml up -d
```

## Interacting with the Cluster

This document explains how to interact with the Kubernetes cluster running inside the Fedora CoreOS VM.

## Cluster Architecture

The cluster consists of:
- **3 Controller nodes**: `k0s-controller-1`, `k0s-controller-2`, `k0s-controller-3`
- **3 Worker nodes**: `k0s-worker-1`, `k0s-worker-2`, `k0s-worker-3`
- **Load balancer**: `k0s-lb` (Traefik)

## Prerequisites

Ensure the cluster is running:
```bash
DOCKER_HOST=tcp://localhost:2377 docker-compose ps
```

## Accessing the Kubernetes API

### 1. Extract kubeconfig from the cluster

Get the kubeconfig from the primary controller:
```bash
DOCKER_HOST=tcp://localhost:2377 docker exec k0s-controller-1 k0s kubeconfig admin > kubeconfig
```

### 2. Update server endpoint

Edit the kubeconfig file to point to the correct endpoint:
```bash
# Replace the server URL in kubeconfig
sed -i '' 's|https://k0s-lb:6443|https://localhost:6443|g' kubeconfig
```

### 3. Use kubectl with the extracted config

```bash
export KUBECONFIG=$(pwd)/kubeconfig
kubectl get nodes
kubectl get pods -A
```

## Common Operations

### Check cluster status
```bash
kubectl get nodes -o wide
kubectl cluster-info
```

### Deploy applications
```bash
# Example: Deploy nginx
kubectl create deployment nginx --image=nginx
kubectl expose deployment nginx --port=80 --type=NodePort
kubectl get services
```

### Access pod logs
```bash
kubectl logs -f deployment/nginx
```

### Execute commands in pods
```bash
kubectl exec -it deployment/nginx -- /bin/bash
```

## Direct Container Access

### Access controller nodes directly
```bash
# Access the primary controller
DOCKER_HOST=tcp://localhost:2377 docker exec -it k0s-controller-1 /bin/sh

# Check k0s status
DOCKER_HOST=tcp://localhost:2377 docker exec k0s-controller-1 k0s status
```

### Access worker nodes directly
```bash
# Access a worker node
DOCKER_HOST=tcp://localhost:2377 docker exec -it k0s-worker-1 /bin/sh
```

### View container logs
```bash
# View controller logs
DOCKER_HOST=tcp://localhost:2377 docker logs k0s-controller-1

# View worker logs
DOCKER_HOST=tcp://localhost:2377 docker logs k0s-worker-1

# View load balancer logs
DOCKER_HOST=tcp://localhost:2377 docker logs k0s-lb
```

## Troubleshooting

### Check if all containers are running
```bash
DOCKER_HOST=tcp://localhost:2377 docker ps
```

### Inspect the k0s network
```bash
DOCKER_HOST=tcp://localhost:2377 docker network ls
DOCKER_HOST=tcp://localhost:2377 docker network inspect compose-cluster_k0s-net
```

### Reset cluster (if needed)
```bash
DOCKER_HOST=tcp://localhost:2377 docker-compose down -v
DOCKER_HOST=tcp://localhost:2377 docker-compose up -d
```

### Check k0s cluster health
```bash
DOCKER_HOST=tcp://localhost:2377 docker exec k0s-controller-1 k0s kubectl get componentstatuses
```

## Port Mappings

- **6443**: Kubernetes API Server (exposed via k0s-lb)
- **9443**: k0s API (internal, not exposed to host)
- **8132**: Konnectivity (internal, not exposed to host)

## Notes

- The cluster uses k0s v1.33.4
- All containers run in privileged mode for full Kubernetes functionality
- Data persistence is handled through Docker volumes
- The load balancer (Traefik) handles API server load balancing across controllers
- Worker tokens are automatically generated and distributed by the primary controller