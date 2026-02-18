# Cloud Provider for NVIDIA Bare Metal Manager (BMM)

Kubernetes Cloud Controller Manager (CCM) for NVIDIA Bare Metal Manager platform.

## Overview

This repository implements the Kubernetes Cloud Provider interface for NVIDIA BMM, enabling native integration between Kubernetes and the NVIDIA BMM bare-metal infrastructure platform.

### What is a Cloud Provider?

The Cloud Provider interface allows Kubernetes to interact with underlying cloud infrastructure. The Cloud Controller Manager (CCM) is a Kubernetes control plane component that runs cloud-specific control loops.

### What Does This Provider Do?

The NVIDIA BMM Cloud Provider implements:

1. **Node Controller**: Manages node lifecycle
   - Initializes nodes with provider IDs (`nvidia-bmm://org/tenant/site/instance-id`)
   - Labels nodes with zone and region information
   - Updates node addresses based on NVIDIA BMM instance network configuration
   - Removes nodes that have been terminated in NVIDIA BMM

2. **Zone Support**: Provides zone and region information for scheduling
   - Maps NVIDIA BMM sites to Kubernetes zones
   - Enables zone-aware pod scheduling and volume topology

3. **Instance Metadata**: Queries NVIDIA BMM API for node/instance information
   - Checks if instances exist
   - Detects shutdown/terminated instances
   - Retrieves instance network configuration

**Note**: Load balancer and routes are not currently supported. Use external solutions like MetalLB or kube-vip for LoadBalancer services.

## Architecture

```
+----------------------------------------------------------+
|            Kubernetes Control Plane                      |
|  +----------------------------------------------------+  |
|  |   kube-controller-manager (built-in controllers)   |  |
|  +----------------------------------------------------+  |
|                                                          |
|  +----------------------------------------------------+  |
|  |   NVIDIA BMM Cloud Controller Manager (CCM)        |  |
|  |   +------------------------------------------+     |  |
|  |   |  Node Controller                         |     |  |
|  |   |  - Initialize nodes with provider IDs    |     |  |
|  |   |  - Update node addresses                 |     |  |
|  |   |  - Remove terminated nodes               |     |  |
|  |   +------------------------------------------+     |  |
|  +------------------+---------------------------------+  |
+---------------------+------------------------------------+
                      | Watches Nodes
                      | Updates Node Status
                      v
+----------------------------------------------------------+
|            Kubernetes API Server                         |
|                   (Node Objects)                         |
+----------------------------------------------------------+
                      |
                      | Queries Instance Info
                      v
+----------------------------------------------------------+
|         NVIDIA BMM REST API Client                       |
|         (github.com/NVIDIA/carbide-rest/client)          |
+----------------------------------------------------------+
                      |
                      v
+----------------------------------------------------------+
|            NVIDIA Bare Metal Manager Platform            |
|       (Bare-Metal Infrastructure Management)             |
+----------------------------------------------------------+
```

## Dependencies

- **[github.com/NVIDIA/carbide-rest/client](../carbide-rest/client)** - Auto-generated REST API client
- **k8s.io/cloud-provider** - Kubernetes cloud provider framework
- **k8s.io/component-base** - Kubernetes component utilities

## Installation

### Prerequisites

1. **Kubernetes cluster** (v1.28+) deployed on NVIDIA BMM infrastructure
2. **NVIDIA BMM API credentials**:
   - API endpoint URL
   - Organization name
   - Authentication token
   - Site UUID
   - Tenant UUID
3. **Control plane nodes** with network access to NVIDIA BMM API

### Step 1: Build the Image

```bash
# Clone repository
git clone https://github.com/NVIDIA/cloud-provider-nvidia-bmm.git
cd cloud-provider-nvidia-bmm

# Build binary locally
make build

# Build Docker image
make docker-build IMG=your-registry/cloud-provider-nvidia-bmm:latest

# Push to registry
make docker-push IMG=your-registry/cloud-provider-nvidia-bmm:latest
```

### Step 2: Configure Cloud Provider

Create a cloud configuration file with your NVIDIA BMM credentials:

```yaml
# config/cloud-config.yaml
endpoint: "https://api.carbide.nvidia.com"
orgName: "your-org-name"
token: "your-api-token"
siteId: "550e8400-e29b-41d4-a716-446655440000"
tenantId: "660e8400-e29b-41d4-a716-446655440001"
```

**Alternatively**, use environment variables (takes precedence over file config):

```bash
export NVIDIA_BMM_ENDPOINT="https://api.carbide.nvidia.com"
export NVIDIA_BMM_ORG_NAME="your-org-name"
export NVIDIA_BMM_TOKEN="your-api-token"
export NVIDIA_BMM_SITE_ID="550e8400-e29b-41d4-a716-446655440000"
export NVIDIA_BMM_TENANT_ID="660e8400-e29b-41d4-a716-446655440001"
```

### Step 3: Deploy to Kubernetes

1. **Update the cloud config secret:**

Edit `deploy/manifests/cloud-config-secret.yaml` with your actual credentials:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: nvidia-bmm-cloud-config
  namespace: kube-system
stringData:
  cloud-config: |
    endpoint: "https://api.carbide.nvidia.com"
    orgName: "your-org-name"
    token: "your-api-token"
    siteId: "550e8400-e29b-41d4-a716-446655440000"
    tenantId: "660e8400-e29b-41d4-a716-446655440001"
```

2. **Update the deployment image:**

Edit `deploy/manifests/deployment.yaml` to use your image:

```yaml
image: your-registry/cloud-provider-nvidia-bmm:latest
```

3. **Deploy:**

```bash
# Deploy RBAC and cloud controller manager
make deploy

# Or manually:
kubectl apply -f deploy/rbac/
kubectl apply -f deploy/manifests/
```

4. **Verify deployment:**

```bash
# Check pod status
kubectl get pods -n kube-system -l app=nvidia-bmm-cloud-controller-manager

# Check logs
kubectl logs -n kube-system -l app=nvidia-bmm-cloud-controller-manager -f

# Verify nodes have provider IDs
kubectl get nodes -o custom-columns=NAME:.metadata.name,PROVIDER-ID:.spec.providerID
```

Expected output:
```
NAME        PROVIDER-ID
worker-1    nvidia-bmm://myorg/mytenant/mysite/instance-uuid-1
worker-2    nvidia-bmm://myorg/mytenant/mysite/instance-uuid-2
master-1    nvidia-bmm://myorg/mytenant/mysite/instance-uuid-3
```

### Step 4: Configure Kubelet

For proper integration, kubelets should be started with the cloud provider flag:

```bash
kubelet \
  --cloud-provider=external \
  --provider-id=nvidia-bmm://org/tenant/site/instance-id \
  ...
```

The provider ID format: `nvidia-bmm://<org-name>/<tenant-name>/<site-id>/<instance-uuid>`

## Configuration Reference

### Cloud Config File

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `endpoint` | string | Yes | NVIDIA BMM API endpoint URL |
| `orgName` | string | Yes | Organization name in NVIDIA BMM |
| `token` | string | Yes | API authentication token |
| `siteId` | string | Yes | Site UUID where cluster is deployed |
| `tenantId` | string | Yes | Tenant UUID for the cluster |

### Environment Variables

Environment variables override cloud config file values:

- `NVIDIA_BMM_ENDPOINT` - API endpoint
- `NVIDIA_BMM_ORG_NAME` - Organization name
- `NVIDIA_BMM_TOKEN` - Authentication token
- `NVIDIA_BMM_SITE_ID` - Site UUID
- `NVIDIA_BMM_TENANT_ID` - Tenant UUID

### Command Line Flags

The cloud controller manager accepts standard Kubernetes CCM flags:

```bash
--cloud-provider=nvidia-bmm        # Cloud provider name (required)
--cloud-config=/path/to/config     # Path to cloud config file
--use-service-account-credentials  # Use service account for cloud API
--leader-elect                     # Enable leader election (multi-replica)
--leader-elect-resource-name       # Leader election lock name
--v=2                              # Log verbosity level
```

## Usage

### Node Lifecycle

When a new node joins the cluster:

1. Kubelet starts with `--cloud-provider=external` and `--provider-id=nvidia-bmm://...`
2. CCM Node Controller detects the new node
3. CCM queries NVIDIA BMM API for instance metadata
4. CCM updates node with:
   - Provider ID
   - Node addresses (InternalIP from NVIDIA BMM interfaces)
   - Zone labels (`topology.kubernetes.io/zone`)
   - Region labels (`topology.kubernetes.io/region`)

When an instance is terminated in NVIDIA BMM:

1. CCM periodically checks instance status
2. If instance is in "Terminating", "Terminated", or "Error" state
3. CCM marks the node as shutdown
4. Kubernetes evicts pods and eventually removes the node

### Zone-Aware Scheduling

With zone information from NVIDIA BMM, you can use zone-aware features:

**Pod topology spread:**
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
spec:
  topologySpreadConstraints:
    - maxSkew: 1
      topologyKey: topology.kubernetes.io/zone
      whenUnsatisfiable: DoNotSchedule
      labelSelector:
        matchLabels:
          app: my-app
```

**Volume topology:**
```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: local-storage
provisioner: kubernetes.io/no-provisioner
volumeBindingMode: WaitForFirstConsumer
allowedTopologies:
  - matchLabelExpressions:
      - key: topology.kubernetes.io/zone
        values:
          - nvidia-bmm-zone-site-123
```

## Development

### Local Development

```bash
# Install dependencies
go mod download

# Run tests
make test

# Run locally (requires kubeconfig and cloud config)
make run

# Format code
make fmt

# Vet code
make vet
```

### Project Structure

```
cloud-provider-nvidia-bmm/
├── cmd/nvidia-bmm-cloud-controller-manager/  # CCM entry point
├── pkg/cloudprovider/                        # Cloud provider implementation
│   ├── nvidia_bmm_cloud.go                   # Main provider interface
│   ├── instances.go                          # InstancesV2 implementation
│   ├── zones.go                              # Zones implementation
│   └── loadbalancer.go                       # Load balancer (not implemented)
├── pkg/providerid/                           # Provider ID parsing
│   └── providerid.go                         # Provider ID types and parsing
├── deploy/                                   # Kubernetes manifests
│   ├── rbac/                                 # ServiceAccount, ClusterRole, etc.
│   └── manifests/                            # Deployment, Secret
├── config/                                   # Sample configurations
├── Dockerfile                                # Container build
└── Makefile                                  # Build automation
```

## Troubleshooting

### Nodes Don't Have Provider IDs

**Symptoms:**
- `kubectl get nodes -o yaml` shows `spec.providerID` is empty
- CCM logs show "node has no provider ID"

**Solutions:**
1. Ensure kubelet is started with `--cloud-provider=external`
2. Ensure kubelet is started with `--provider-id=nvidia-bmm://org/tenant/site/instance-id`
3. Verify the provider ID format matches NVIDIA BMM instance IDs

### CCM Can't Connect to NVIDIA BMM API

**Symptoms:**
- CCM logs show connection errors or authentication failures
- Nodes not being initialized with metadata

**Solutions:**
1. Verify cloud config credentials are correct
2. Check network connectivity from control plane to NVIDIA BMM API
3. Verify API token has not expired
4. Check CCM logs for specific error messages

### Nodes Stuck in "NotReady" State

**Symptoms:**
- Nodes appear in cluster but remain "NotReady"
- CCM can't fetch instance metadata

**Solutions:**
1. Verify instance actually exists in NVIDIA BMM (check instance UUID)
2. Check instance status in NVIDIA BMM is not "Error" or "Terminating"
3. Verify siteID and tenantID in cloud config match instance location
4. Check instance has network interfaces with IP addresses

### Permission Errors

**Symptoms:**
- CCM logs show "forbidden" or permission errors
- CCM can't update node status

**Solutions:**
```bash
# Verify RBAC is deployed
kubectl get clusterrole system:cloud-controller-manager
kubectl get clusterrolebinding system:cloud-controller-manager

# Verify service account can update nodes
kubectl auth can-i update nodes \
  --as=system:serviceaccount:kube-system:cloud-controller-manager
```

### High API Request Rate

**Symptoms:**
- NVIDIA BMM API rate limiting errors
- CCM making excessive API calls

**Solutions:**
1. Increase sync period (default controller intervals)
2. Enable caching in cloud provider (if available)
3. Reduce node count or number of CCM replicas

## Comparison with Other Providers

| Feature | NVIDIA BMM | AWS | Azure | GCP | OpenStack |
|---------|------------|-----|-------|-----|-----------|
| Node Management | Yes | Yes | Yes | Yes | Yes |
| Zone Support | Yes | Yes | Yes | Yes | Yes |
| Load Balancer | No (use MetalLB) | Yes | Yes | Yes | Yes |
| Routes | No | Yes | Yes | Yes | Yes |
| Bare Metal | Yes | No | No | No | Yes |

## License

Apache 2.0

## Related Projects

- [cluster-api-provider-nvidia-bmm](../cluster-api-provider-nvidia-bmm) - Cluster API provider for NVIDIA BMM
- [machine-api-provider-nvidia-bmm](../machine-api-provider-nvidia-bmm) - OpenShift Machine API provider for NVIDIA BMM
- [carbide-rest](../carbide-rest) - REST API client library

## Contributing

Contributions are welcome! Please submit issues and pull requests to the GitHub repository.

## References

- [Kubernetes Cloud Provider Documentation](https://kubernetes.io/docs/concepts/architecture/cloud-controller/)
- [Cloud Provider Interface](https://github.com/kubernetes/cloud-provider)
- [Developing Cloud Controller Manager](https://kubernetes.io/docs/tasks/administer-cluster/developing-cloud-controller-manager/)
