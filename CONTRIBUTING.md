# Contributing to MetalK8s Registry Node Agent

This document provides guidelines for contributing to the MetalK8s Registry Node Agent project.

## Repository Structure

The project follows a clean architecture pattern with clear separation of concerns:

```
├── api/                    # Kubernetes API definitions (CRDs)
├── cmd/                    # Application entry point and configuration
├── internal/controller/    # Kubernetes controller logic
├── pkg/
│   ├── domain/            # Core business entities and types
│   ├── usecase/           # Business logic and use cases
│   ├── service/           # Service interfaces
│   ├── infrastructure/    # Infrastructure implementations
│   └── presentation/      # HTTP API layer
└── config/                # Kubernetes manifests and configuration
```

## Development Setup

### Prerequisites

- Go 1.25.0 or later
- Docker for containerization
- Kubernetes cluster
- kubectl for Kubernetes integration
- Git for version control

## Building

```bash
# Build the binary
make build

# Build the Docker image
make docker-build
```

### Running Locally

```bash
# Set environment variables
export ARTIFACT_STORAGE_ROOT_LOCATION=/tmp/artifacts
export HTTP_ADDR=:5001

# Run the application
go run cmd/main.go
```

### Deploying to Kubernetes

```bash
# Install CRDs
kubectl apply -f config/crd/

# Deploy the operator
kubectl apply -f config/default/

# Check deployment status
kubectl get pods -n metalk8s-registry-node-agent-system
```

## Complete example

**1. Create a fake ISO file**
```shell
dd if=/dev/urandom of=./test.iso bs=1M count=20
    20+0 records in
    20+0 records out
    20971520 bytes (21 MB, 20 MiB) copied, 0.0453264 s, 463 MB/s
```

```shell
sha256sum test.iso
    ce775a33b30ae640d521df1fad60868fa701707ffdc4d8b4ca7ab60edfd05c26  test.iso
```

**2. Split it in 5 parts**
```shell
split -n 5 -x test.iso "test.iso."
    -rw-rw-r--  1 user group  4194304 Sep 25 15:46 test.iso.00
    -rw-rw-r--  1 user group  4194304 Sep 25 15:46 test.iso.01
    -rw-rw-r--  1 user group  4194304 Sep 25 15:46 test.iso.02
    -rw-rw-r--  1 user group  4194304 Sep 25 15:46 test.iso.03
    -rw-rw-r--  1 user group  4194304 Sep 25 15:46 test.iso.04
```

**3. Create a NodeArtifact Custom Resource**

```shell
kubectl apply -f config/samples/metalk8s_v1alpha1_nodeartifact.yaml
```

**4. Upload parts**
```shell
wget -qO- \
  --header="X-Target-Version: 1.25.3" \
  --header="X-Sha256-checksum: ce775a33b30ae640d521df1fad60868fa701707ffdc4d8b4ca7ab60edfd05c26" \
  --header "Content-Range: bytes 0-4194303/20971520" \
  --post-file=test.iso.00 http://localhost:5001/api/v1/uploads/metalk8s
```
```json
{
  "artifact": "metalk8s",
  "isCompleted": false,
  "sha256sum": "ce775a33b30ae640d521df1fad60868fa701707ffdc4d8b4ca7ab60edfd05c26",
  "size": 20971520,
  "uploadedChunks": [
    {
      "rangeEndIndex": 4194303,
      "rangeSize": 4194304,
      "rangeStartIndex": 0
    }
  ],
  "version": "1.25.3"
}
```
```shell
wget -qO- --header="X-Target-Version: 1.25.3" --header="X-Sha256-checksum: ce775a33b30ae640d521df1fad60868fa701707ffdc4d8b4ca7ab60edfd05c26" --header "Content-Range: bytes 4194304-8388607/20971520" --post-file=test.iso.01 http://localhost:5001/api/v1/uploads/metalk8s

wget -qO- --header="X-Target-Version: 1.25.3" --header="X-Sha256-checksum: ce775a33b30ae640d521df1fad60868fa701707ffdc4d8b4ca7ab60edfd05c26" --header "Content-Range: bytes 8388608-12582911/20971520" --post-file=test.iso.02 http://localhost:5001/api/v1/uploads/metalk8s

wget -qO- --header="X-Target-Version: 1.25.3" --header="X-Sha256-checksum: ce775a33b30ae640d521df1fad60868fa701707ffdc4d8b4ca7ab60edfd05c26" --header "Content-Range: bytes 12582912-16777215/20971520" --post-file=test.iso.03 http://localhost:5001/api/v1/uploads/metalk8s
```
```json
{
  "artifact": "metalk8s",
  "isCompleted": false,
  "sha256sum": "ce775a33b30ae640d521df1fad60868fa701707ffdc4d8b4ca7ab60edfd05c26",
  "size": 20971520,
  "uploadedChunks": [
    {
      "rangeEndIndex": 4194303,
      "rangeSize": 4194304,
      "rangeStartIndex": 0
    },
    {
      "rangeEndIndex": 8388607,
      "rangeSize": 4194304,
      "rangeStartIndex": 4194304
    },
    {
      "rangeEndIndex": 12582911,
      "rangeSize": 4194304,
      "rangeStartIndex": 8388608
    },
    {
      "rangeEndIndex": 16777215,
      "rangeSize": 4194304,
      "rangeStartIndex": 12582912
    }
  ],
  "version": "1.25.3"
}
```
Last part uploaded:
```shell
wget -qO- \
  --header="X-Target-Version: 1.25.3" \
  --header="X-Sha256-checksum: ce775a33b30ae640d521df1fad60868fa701707ffdc4d8b4ca7ab60edfd05c26" \
  --header "Content-Range: bytes 16777216-20971519/20971520" \
  --post-file=test.iso.04 http://localhost:5001/api/v1/uploads/metalk8s
```
```json
{
  "artifact": "metalk8s",
  "isCompleted": true,
  "sha256sum": "ce775a33b30ae640d521df1fad60868fa701707ffdc4d8b4ca7ab60edfd05c26",
  "size": 20971520,
  "uploadedChunks": [
    {
      "rangeEndIndex": 4194303,
      "rangeSize": 4194304,
      "rangeStartIndex": 0
    },
    {
      "rangeEndIndex": 8388607,
      "rangeSize": 4194304,
      "rangeStartIndex": 4194304
    },
    {
      "rangeEndIndex": 12582911,
      "rangeSize": 4194304,
      "rangeStartIndex": 8388608
    },
    {
      "rangeEndIndex": 16777215,
      "rangeSize": 4194304,
      "rangeStartIndex": 12582912
    },
    {
      "rangeEndIndex": 20971519,
      "rangeSize": 4194304,
      "rangeStartIndex": 16777216
    }
  ],
  "version": "1.25.3"
}
```
