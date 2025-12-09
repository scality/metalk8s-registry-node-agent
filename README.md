# MetalK8s Registry Node Agent

A Kubernetes operator and HTTP service for managing solution archive uploads and distribution in MetalK8s clusters. This agent runs on each node labelled as `node-role.kubernetes.io/registry` and provides a local registry for MetalK8s components, enabling efficient solution archive distribution and management.

## Overview

The MetalK8s Registry Node Agent is designed to:

- **Manage SolutionArchive Uploads**: Handle multipart uploads of MetalK8s ISO solutions
- **Session Management**: Initialize and manage upload sessions with proper validation
- **Storage Management**: Provide filesystem-based storage with bucket organization
- **Kubernetes Integration**: Operate as a Kubernetes operator with custom resource definitions
- **Health Monitoring**: Provide health checks and metrics for cluster monitoring
- **Clean unused SolutionArchives**: Provide a garbagge collector to clean unused files and directories

## Key Features

### 1. Multipart Upload System
- **Chunked Uploads**: Support for uploading large files in parts
- **Checksum Validation**: SHA256 validation for data integrity
- **Session Initialization**: Create isolated upload sessions
- **SolutionArchive Validation**: Validate solution archive metadata and requirements

### 2. Storage Provider
- **Filesystem Backend**: Local filesystem storage implementation
- **Bucket Organization**: Organized storage using buckets for different sessions
- **Concurrent Access**: Thread-safe operations with proper locking

### 3. Kubernetes Integration
- **Custom Resource Definition**: `NodeSolutionArchive` CRD for solution archive management
- **Controller Pattern**: Reconciles desired state with actual state
- **RBAC Support**: Proper role-based access control
- **Metrics Integration**: Prometheus metrics for monitoring

## Configuration

The agent can be configured using environment variables:

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SOLUTION_ARCHIVES_LOCATION` | Root directory for solution archive storage | `/archives` |
| `SOLUTIONS_LOCATION` | Root directory for solution archive storage | `/solutions` |
| `NODE_NAME` | Node name for which the controller listens for events | none |
| `EXTERN_ADDR` | HTTP server address for upload feature | `:5001` |
| `INTERN_ADDR` | HTTP server address for download between nodes | `:5002` |
| `NODE_IP` | Node IP on which to expose the download API | none |
| `EXTERN_SERVER_TLS_CERT_FILE_PATH` | Path to TLS Certificate for upload endpoint | none |
| `EXTERN_SERVER_TLS_KEY_FILE_PATH` | Path to TLS Key for upload endpoint | none |
| `EXTERN_SERVER_AUTHN_CA_CERT_FILE_PATH` | Path to CA Certificate for mTLS on upload endpoint | none |
| `INTERN_SERVER_TLS_CERT_FILE_PATH` | Path to TLS Certificate for download endpoint | none |
| `INTERN_SERVER_TLS_KEY_FILE_PATH` | Path to TLS Key for download endpoint | none |
| `INTERN_SERVER_AUTHN_CA_CERT_FILE_PATH` | Path to CA Certificate for mTLS on download endpoint | none |
| `INTERN_CLIENT_TLS_CA_CERT_FILE_PATH` | Path to TLS CA Certificate for internal client for download feature | none |
| `INTERN_CLIENT_AUTHN_CERT_FILE_PATH` | Path to mTLS Client Certificate for download feature | none |
| `INTERN_CLIENT_AUTHN_KEY_FILE_PATH` | Path to mTLS Client Key for download feature | none |
| `LOGGER_LOG_LEVEL` | Logging level | `info` |

### Kubernetes Configuration

The agent includes comprehensive Kubernetes manifests:

- **CRD**: Custom Resource Definition for `NodeSolutionArchive`
- **RBAC**: Role-based access control configuration
- **Manager**: Deployment and service configuration
- **Metrics**: Prometheus monitoring setup

## API Usage

### Initialize an Upload Session

**Create a NodeSolutionArchive resource**:
```yaml
apiVersion: metalk8s.scality.com/v1alpha1
kind: NodeSolutionArchive
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

Endpoint: `POST /api/v1/uploads/{solution-archive}`

Upload a chunk of an solution archive to the registry.  
Review the API specification in `pkg/presentation/http/extern/uploads-openapi.yaml`

**Upload chunks via API**:
```bash
curl -X POST \
    --http1.1 \
    -H "X-Target-Version: 1.25.3" \
    -H "X-Sha256-checksum: ce775a33b30ae640d521df1fad60868fa701707ffdc4d8b4ca7ab60edfd05c26" \
    -H "Content-Range: bytes 0-1048575/20971520" \
    -H "Content-Type: application/octet-stream" \
    --cert "/path/to/mtls/tls.crt" \
    --key "/path/to/mtls/tls.key" \
    --cacert "/path/to/tls/ca.crt" \
    --data-binary @chunk1.bin \
    https://localhost:5001/api/v1/uploads/metalk8s
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
