# MetalK8s Registry Node Agent

A Kubernetes operator and HTTP service for managing solution archive uploads and distribution in MetalK8s clusters. This agent runs on each node labelled as `node-role.kubernetes.io/registry` and provides a local registry for MetalK8s components, enabling efficient solution archive distribution and management.

## Overview

The MetalK8s Registry Node Agent is designed to:

- **Manage SolutionArchive Uploads**: Handle multipart uploads of MetalK8s ISO solutions
- **Manage SolutionArchive distribution**: Handle Solution Archives replication between labeled nodes through its internal download API
- **Session Management**: Initialize and manage upload sessions with proper validation
- **Storage Management**: Provide filesystem-based storage with bucket organization
- **Kubernetes Integration**: Operate as a Kubernetes operator with custom resource definitions
- **Health Monitoring**: Provide health checks and metrics for cluster monitoring
- **Clean unused SolutionArchives**: Provide a garbagge collector to clean unused files and directories
- **Care about security**: Upload and Download API are protected with TLS and mTLS authentication

## Key Features

### 1. Multipart Upload System
- **Chunked Uploads**: Support for uploading large files in parts
- **Checksum Validation**: SHA256 validation for data integrity
- **Session Initialization**: Create isolated upload sessions
- **SolutionArchive Validation**: Validate solution archive metadata and requirements

### 2. Internal Multipart Replication System
- **Chunked Downloads**: Support for replicating large files through multipart downloads
- **Checksum Validation**: SHA256 validation for data integrity for all chunks

### 3. Storage Provider
- **Filesystem Backend**: Local filesystem storage implementation
- **Bucket Organization**: Organized storage using buckets for different sessions
- **Concurrent Access**: Thread-safe operations with proper locking

### 4. Kubernetes Integration
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
| `DOWNLOAD_HOST` | Node IP on which to expose the download API | none |
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

Endpoint: `PUT /api/v1/uploads/{solution-archive}/{version}`

Upload a chunk of an solution archive to the registry.  
Review the API specification in `pkg/presentation/http/extern/uploads-openapi.yaml`

**Upload chunks via API**:
```bash
curl -X PUT \
    --http1.1 \
    -H "Content-Range: bytes 0-1048575/20971520" \
    -H "Content-Type: application/octet-stream" \
    --cert "/path/to/mtls/tls.crt" \
    --key "/path/to/mtls/tls.key" \
    --cacert "/path/to/tls/ca.crt" \
    --data-binary @chunk1.bin \
    https://localhost:5001/api/v1/uploads/metalk8s/1.25.3
```

### Download API

The download API is the internal endpoint used by registry nodes to replicate solution archives between each other. Once a solution archive has been uploaded and validated to one node (via the Upload API on `EXTERN_ADDR`), the controllers running on the other registry nodes pull the missing chunks from the source node through this API, exposed on `INTERN_ADDR` (default `:5002`) and bound to the node IP defined by `DOWNLOAD_HOST`.

The endpoint is protected by mTLS using the `INTERN_SERVER_*` certificates on the server side and the `INTERN_CLIENT_*` certificates on the client side, ensuring that only legitimate registry nodes of the cluster can fetch archive data.

Two operations are exposed under `/api/v1/downloads/{solution-archive}/{version}`:

- `HEAD`: returns the total size of the archive (`Content-Length`) so the caller can plan its chunked download.
- `GET`: returns a single chunk of the archive as `application/octet-stream`. The byte range is selected via the `Range` header (`bytes=<start>-<end>`), and the response is a `206 Partial Content` with a `Content-Range` header and a `Content-Digest` HTTP trailer carrying the SHA-256 of the chunk for end-to-end integrity verification.

The full specification is available in `pkg/presentation/http/intern/downloads-openapi.yaml`.

**Describe the archive (size discovery)**:
```bash
curl -I \
    --http1.1 \
    --cert "/path/to/mtls/tls.crt" \
    --key "/path/to/mtls/tls.key" \
    --cacert "/path/to/tls/ca.crt" \
    https://<peer-node-ip>:5002/api/v1/downloads/metalk8s/1.25.3
```

**Download a chunk**:
```bash
curl -X GET \
    --http1.1 \
    -H "Range: bytes=0-1048575" \
    --cert "/path/to/mtls/tls.crt" \
    --key "/path/to/mtls/tls.key" \
    --cacert "/path/to/tls/ca.crt" \
    -o chunk1.bin \
    https://<peer-node-ip>:5002/api/v1/downloads/metalk8s/1.25.3
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
