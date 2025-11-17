# MetalK8s Registry Node Agent

A Kubernetes operator and HTTP service for managing artifact uploads and distribution in MetalK8s clusters. This agent runs on each node labelled as `node-role.kubernetes.io/registry` and provides a local registry for MetalK8s components, enabling efficient artifact distribution and management.

## Overview

The MetalK8s Registry Node Agent is designed to:

- **Manage Artifact Uploads**: Handle multipart uploads of MetalK8s ISO artifacts and components
- **Session Management**: Initialize and manage upload sessions with proper validation
- **Storage Management**: Provide filesystem-based storage with bucket organization
- **Kubernetes Integration**: Operate as a Kubernetes operator with custom resource definitions
- **Health Monitoring**: Provide health checks and metrics for cluster monitoring

## Key Features

### 1. Multipart Upload System
- **Chunked Uploads**: Support for uploading large files in parts
- **Checksum Validation**: SHA256 validation for data integrity
- **Session Initialization**: Create isolated upload sessions
- **Artifact Validation**: Validate artifact metadata and requirements

### 2. Storage Provider
- **Filesystem Backend**: Local filesystem storage implementation
- **Bucket Organization**: Organized storage using buckets for different sessions
- **Concurrent Access**: Thread-safe operations with proper locking

### 3. Kubernetes Integration
- **Custom Resource Definition**: `NodeArtifact` CRD for artifact management
- **Controller Pattern**: Reconciles desired state with actual state
- **RBAC Support**: Proper role-based access control
- **Metrics Integration**: Prometheus metrics for monitoring

## Configuration

The agent can be configured using environment variables:

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ARTIFACT_STORAGE_ROOT_LOCATION` | Root directory for artifact storage | `/tmp/uploads/artifacts` |
| `HTTP_EXTERN_ADDR` | HTTP server address | `:5001` |
| `LOGGER_LOG_LEVEL` | Logging level | `info` |

### Kubernetes Configuration

The agent includes comprehensive Kubernetes manifests:

- **CRD**: Custom Resource Definition for `NodeArtifact`
- **RBAC**: Role-based access control configuration
- **Manager**: Deployment and service configuration
- **Metrics**: Prometheus monitoring setup

## API Usage

### Initialize an Upload Session

**Create a NodeArtifact resource**:
```yaml
apiVersion: metalk8s.scality.com/v1alpha1
kind: NodeArtifact
metadata:
    name: metalk8s-1.25.3
spec:
    name: metalk8s
    version: 1.25.3
    nodeName: worker-node-1
    validation:
    checksum:
        type: sha256
        value: ce775a33b30ae640d521df1fad60868fa701707ffdc4d8b4ca7ab60edfd05c26
```

### Upload API

Endpoint: `POST /api/v1/uploads/{artifact}`

Upload a chunk of an artifact to the registry.  
Review the API specification in `pkg/presentation/http/generated/uploads-openapi.yaml`

**Upload chunks via API**:
```bash
curl -X POST \
    -H "X-Target-Version: 1.25.3" \
    -H "X-Sha256-checksum: ce775a33b30ae640d521df1fad60868fa701707ffdc4d8b4ca7ab60edfd05c26" \
    -H "Content-Range: bytes 0-1048575/20971520" \
    -H "Content-Type: application/octet-stream" \
    --data-binary @chunk1.bin \
    http://localhost:5001/api/v1/uploads/metalk8s
``` 

### Health Endpoints

#### GET `/healthz`
Liveness probe endpoint for Kubernetes.

#### GET `/readyz`
Readiness probe endpoint for Kubernetes.

### Metrics Endpoint

#### GET `/metrics`
Prometheus metrics endpoint (if enabled).
Not yet implemented

## Contributing

See [contributing](CONTRIBUTING.md) for details.

## Design

See [design](DESIGN.md) for details.
