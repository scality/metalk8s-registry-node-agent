# Contributing to MetalK8s Registry Node Agent

This document provides guidelines for contributing to the MetalK8s Registry Node Agent project.

## Repository Structure

The project follows a clean architecture pattern with clear separation of concerns:

```
├── api/                   # Kubernetes API definitions (CRDs)
│   └── v1alpha1/          # API version v1alpha1 types
├── cmd/                   # Application entry point
│   └── config/            # Environment configuration
├── config/                # Kubernetes manifests and kustomize overlays
├── hack/                  # Build and development scripts
├── internal/              # Internal packages (not importable)
│   ├── controller/        # Kubernetes controller logic
│   └── webhook/           # Admission webhook handlers
├── pkg/
│   ├── domain/            # Core business entities and types
│   ├── usecase/           # Business logic and use cases
│   ├── service/           # Service interfaces
│   ├── infrastructure/    # Infrastructure implementations and DI
│   ├── library/           # Shared utilities (filesystem, filters)
│   └── presentation/      # HTTP API layer
└── test/                  # Test suites
    ├── e2e/               # End-to-end tests
    ├── integration/       # Integration tests
    └── utils/             # Test utilities
```

## Development Setup

### Prerequisites

- Go 1.25.0 or later
- Docker for containerization
- Kubernetes cluster
- kubectl for Kubernetes integration
- Git for version control
- cert-manager for kubernetes deployment

## Building

```bash
# Build the binary
make build

# Build the Docker image
make docker-build IMG=registry.localhost:5000/nodeagent:0.1.0
```

### Generating Certificates

```bash
export CERT_DIR="/certs"
export DAYS_VALID="365"

# Create directory structure
sudo mkdir -p "${CERT_DIR}"
sudo chown -R ${USER} "${CERT_DIR}"
mkdir -p "${CERT_DIR}/server/external"
mkdir -p "${CERT_DIR}/server/internal"
mkdir -p "${CERT_DIR}/client/external"
mkdir -p "${CERT_DIR}/client/internal"

# Generate CA for external server
echo "Generating external server CA..."
openssl req -x509 -newkey rsa:4096 -days ${DAYS_VALID} -nodes \
  -keyout "${CERT_DIR}/server/external/ca.key" \
  -out "${CERT_DIR}/server/external/ca.crt" \
  -subj "/CN=External-Server-CA/O=MetalK8s-Test"

# Generate external server certificate
echo "Generating external server certificate..."
openssl req -newkey rsa:4096 -nodes \
  -keyout "${CERT_DIR}/server/external/tls.key" \
  -out "${CERT_DIR}/server/external/tls.csr" \
  -subj "/CN=localhost/O=MetalK8s-Test"

# Create SAN config for localhost
cat > "${CERT_DIR}/server/external/san.cnf" <<EOF
[v3_req]
keyUsage = keyEncipherment, dataEncipherment, digitalSignature
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = *.localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

# Sign external server certificate
openssl x509 -req -in "${CERT_DIR}/server/external/tls.csr" \
  -CA "${CERT_DIR}/server/external/ca.crt" \
  -CAkey "${CERT_DIR}/server/external/ca.key" \
  -CAcreateserial \
  -out "${CERT_DIR}/server/external/tls.crt" \
  -days ${DAYS_VALID} \
  -extensions v3_req \
  -extfile "${CERT_DIR}/server/external/san.cnf"

# Generate CA for external client (mTLS)
echo "Generating external client CA..."
openssl req -x509 -newkey rsa:4096 -days ${DAYS_VALID} -nodes \
  -keyout "${CERT_DIR}/client/external/ca.key" \
  -out "${CERT_DIR}/client/external/ca.crt" \
  -subj "/CN=External-Client-CA/O=MetalK8s-Test"

# Generate external client certificate
echo "Generating external client certificate..."
openssl req -newkey rsa:4096 -nodes \
  -keyout "${CERT_DIR}/client/external/tls.key" \
  -out "${CERT_DIR}/client/external/tls.csr" \
  -subj "/CN=test-client/O=MetalK8s-Test"

# Create SAN config for client
cat > "${CERT_DIR}/client/external/san.cnf" <<EOF
[v3_req]
keyUsage = keyEncipherment, dataEncipherment, digitalSignature
extendedKeyUsage = clientAuth
EOF

# Sign external client certificate
openssl x509 -req -in "${CERT_DIR}/client/external/tls.csr" \
  -CA "${CERT_DIR}/client/external/ca.crt" \
  -CAkey "${CERT_DIR}/client/external/ca.key" \
  -CAcreateserial \
  -out "${CERT_DIR}/client/external/tls.crt" \
  -days ${DAYS_VALID} \
  -extensions v3_req \
  -extfile "${CERT_DIR}/client/external/san.cnf"

# Generate CA for internal server
echo "Generating internal server CA..."
openssl req -x509 -newkey rsa:4096 -days ${DAYS_VALID} -nodes \
  -keyout "${CERT_DIR}/server/internal/ca.key" \
  -out "${CERT_DIR}/server/internal/ca.crt" \
  -subj "/CN=Internal-Server-CA/O=MetalK8s-Test"

# Generate internal server certificate
echo "Generating internal server certificate..."
openssl req -newkey rsa:4096 -nodes \
  -keyout "${CERT_DIR}/server/internal/tls.key" \
  -out "${CERT_DIR}/server/internal/tls.csr" \
  -subj "/CN=localhost/O=MetalK8s-Test"

# Use same SAN config pattern for internal server
cat > "${CERT_DIR}/server/internal/san.cnf" <<EOF
[v3_req]
keyUsage = keyEncipherment, dataEncipherment, digitalSignature
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = *.localhost
IP.1 = 127.0.0.1
IP.2 = ::1
IP.3 = 172.18.0.1
IP.4 = 172.18.0.2
IP.5 = 172.18.0.3
IP.6 = 172.18.0.4
IP.7 = 172.19.0.1
IP.8 = 172.19.0.2
IP.9 = 172.19.0.3
IP.10 = 172.19.0.4
IP.11 = 172.20.0.1
IP.12 = 172.20.0.2
IP.13 = 172.20.0.3
IP.14 = 172.20.0.4
IP.15 = 172.21.0.1
IP.16 = 172.21.0.2
IP.17 = 172.21.0.3
IP.18 = 172.21.0.4
IP.19 = 172.22.0.1
IP.20 = 172.22.0.2
IP.21 = 172.22.0.3
IP.22 = 172.22.0.4
EOF

# Sign internal server certificate
openssl x509 -req -in "${CERT_DIR}/server/internal/tls.csr" \
  -CA "${CERT_DIR}/server/internal/ca.crt" \
  -CAkey "${CERT_DIR}/server/internal/ca.key" \
  -CAcreateserial \
  -out "${CERT_DIR}/server/internal/tls.crt" \
  -days ${DAYS_VALID} \
  -extensions v3_req \
  -extfile "${CERT_DIR}/server/internal/san.cnf"

# Generate CA for internal client (mTLS)
echo "Generating internal client CA..."
openssl req -x509 -newkey rsa:4096 -days ${DAYS_VALID} -nodes \
  -keyout "${CERT_DIR}/client/internal/ca.key" \
  -out "${CERT_DIR}/client/internal/ca.crt" \
  -subj "/CN=Internal-Client-CA/O=MetalK8s-Test"

# Generate internal client certificate
echo "Generating internal client certificate..."
openssl req -newkey rsa:4096 -nodes \
  -keyout "${CERT_DIR}/client/internal/tls.key" \
  -out "${CERT_DIR}/client/internal/tls.csr" \
  -subj "/CN=test-client-internal/O=MetalK8s-Test"

# Create SAN config for internal client
cat > "${CERT_DIR}/client/internal/san.cnf" <<EOF
[v3_req]
keyUsage = keyEncipherment, dataEncipherment, digitalSignature
extendedKeyUsage = clientAuth
EOF

# Sign internal client certificate
openssl x509 -req -in "${CERT_DIR}/client/internal/tls.csr" \
  -CA "${CERT_DIR}/client/internal/ca.crt" \
  -CAkey "${CERT_DIR}/client/internal/ca.key" \
  -CAcreateserial \
  -out "${CERT_DIR}/client/internal/tls.crt" \
  -days ${DAYS_VALID} \
  -extensions v3_req \
  -extfile "${CERT_DIR}/client/internal/san.cnf"

# Clean up CSR and config files
rm -f "${CERT_DIR}"/server/*/tls.csr
rm -f "${CERT_DIR}"/client/*/tls.csr
rm -f "${CERT_DIR}"/server/*/san.cnf
rm -f "${CERT_DIR}"/client/*/san.cnf
rm -f "${CERT_DIR}"/server/*/ca.srl
rm -f "${CERT_DIR}"/client/*/ca.srl
```

### Running Locally

```bash
# Install CRDs
make install

# Set environment variables
export CERT_DIR="/certs"
export LOGGER_LOG_LEVEL= "debug"
export ENABLE_WEBHOOKS="false"
export SOLUTION_ARCHIVES_LOCATION="/tmp/archives"
export SOLUTIONS_LOCATION="/tmp/solutions"
export NODE_NAME="k3d-k3d-agent-1"
export DOWNLOAD_HOST="localhost"
export EXTERN_SERVER_TLS_CERT_FILE_PATH="${CERT_DIR}/server/external/tls.crt"
export EXTERN_SERVER_TLS_KEY_FILE_PATH="${CERT_DIR}/server/external/tls.key"
export EXTERN_SERVER_AUTHN_CA_CERT_FILE_PATH="${CERT_DIR}/client/external/ca.crt"
export INTERN_SERVER_TLS_CERT_FILE_PATH="${CERT_DIR}/server/internal/tls.crt"
export INTERN_SERVER_TLS_KEY_FILE_PATH="${CERT_DIR}/server/internal/tls.key"
export INTERN_SERVER_AUTHN_CA_CERT_FILE_PATH="${CERT_DIR}/client/internal/ca.crt"
export INTERN_CLIENT_TLS_CA_CERT_FILE_PATH="${CERT_DIR}/server/internal/ca.crt"
export INTERN_CLIENT_AUTHN_CERT_FILE_PATH="${CERT_DIR}/client/internal/tls.crt"
export INTERN_CLIENT_AUTHN_KEY_FILE_PATH="${CERT_DIR}/client/internal/tls.key"

# Run the application
make run
```

### Deploying to Kubernetes

```bash
# Install CRDs
make install

# Deploy certificates
kubectl create ns metalk8s-registry-node-agent-system
kubectl apply -n metalk8s-registry-node-agent-system -f test/e2e/e2e-certs.yaml
export CERT_DIR="/tmp/certs"
kubectl get secret -n metalk8s-registry-node-agent-system tls-cert -o jsonpath="{.data.tls\.crt}" | base64 -d > ${CERT_DIR}/tls.crt
kubectl get secret -n metalk8s-registry-node-agent-system tls-cert -o jsonpath="{.data.tls\.key}" | base64 -d > ${CERT_DIR}/tls.key
kubectl get secret -n metalk8s-registry-node-agent-system tls-cert -o jsonpath="{.data.ca\.crt}" | base64 -d > ${CERT_DIR}/ca.crt

# Deploy the operator
make deploy IMG=registry.localhost:5000/nodeagent:0.1.0

# Check deployment status
kubectl get pods -n metalk8s-registry-node-agent-system

# Activate port-forward to upload
kubectl port-forward -n metalk8s-registry-node-agent-system \
  $(kubectl get pod -n metalk8s-registry-node-agent-system -o name) \
  5001.5001
```

## Complete example

**1. Create a fake ISO file**
```shell
export TMP_DIR="/tmp/iso"
mkdir -p ${TMP_DIR}
dd if=/dev/urandom of=${TMP_DIR}/file1.txt bs=1M count=10
dd if=/dev/urandom of=${TMP_DIR}/file2.txt bs=1M count=2
dd if=/dev/urandom of=${TMP_DIR}/file3.txt bs=1M count=5
dd if=/dev/urandom of=${TMP_DIR}/file4.txt bs=1M count=12
genisoimage -o /tmp/test.iso /tmp/iso
```
```shell
ls -l /tmp/test.iso
  -rw-rw-r-- 1 user group 30765056 Dec  9 11:44  test.iso
```

```shell
sha256sum /tmp/test.iso
    81c931664a390272a230b5c56b23c5652559f5bed085282b70fcc56e9c3c2b2f  /tmp/test.iso
```

**2. Split it in 5 parts**
```shell
split -n 5 -x /tmp/test.iso "/tmp/test.iso."
  -rw-rw-r-- 1 user group  6153012 Dec  9 11:46  test.iso.00
  -rw-rw-r-- 1 user group  6153011 Dec  9 11:46  test.iso.01
  -rw-rw-r-- 1 user group  6153011 Dec  9 11:46  test.iso.02
  -rw-rw-r-- 1 user group  6153011 Dec  9 11:46  test.iso.03
  -rw-rw-r-- 1 user group  6153011 Dec  9 11:46  test.iso.04
```

**3. Create a NodeSolutionArchive Custom Resource**

```shell
kubectl apply -f config/samples/metalk8s_v1alpha1_nodesolutionarchive.yaml
```

**4. Upload parts**
```shell
export CERT_DIR="/tmp/certs"
curl -X PUT \
  --http1.1 \
  -H "X-Target-Version: 1.25.3" \
  -H "Content-Range: bytes 0-6153011/30765056" \
  -H "Content-Type: application/octet-stream" \
  --cert "${CERT_DIR}/tls.crt" \
  --key "${CERT_DIR}/tls.key" \
  --cacert "${CERT_DIR}/ca.crt" \
  --data-binary @test.iso.00 \
  https://localhost:5001/api/v1/uploads/metalk8s
```
```json
{
  "solutionArchive": "metalk8s",
  "isCompleted": false,
  "sha256sum": "81c931664a390272a230b5c56b23c5652559f5bed085282b70fcc56e9c3c2b2f",
  "size": 30765056,
  "uploadedChunks":
  [
    {
      "rangeEndIndex": 6153011,
      "rangeSize": 6153012,
      "rangeStartIndex": 0
    }
  ],
  "version": "1.25.3"
}
```
```shell
export CERT_DIR="/tmp/certs"

curl -X PUT --http1.1 -H "X-Target-Version: 1.25.3" -H "Content-Range: bytes 6153012-12306022/30765056" -H "Content-Type: application/octet-stream" --cert "${CERT_DIR}/tls.crt" --key "${CERT_DIR}/tls.key" --cacert "${CERT_DIR}/ca.crt" --data-binary @test.iso.01 https://localhost:5001/api/v1/uploads/metalk8s

curl -X PUT --http1.1 -H "X-Target-Version: 1.25.3" -H "Content-Range: bytes 12306023-18459033/30765056" -H "Content-Type: application/octet-stream" --cert "${CERT_DIR}/tls.crt" --key "${CERT_DIR}/tls.key" --cacert "${CERT_DIR}/ca.crt" --data-binary @test.iso.02 https://localhost:5001/api/v1/uploads/metalk8s

curl -X PUT --http1.1 -H "X-Target-Version: 1.25.3" -H "Content-Range: bytes 18459034-24612044/30765056" -H "Content-Type: application/octet-stream" --cert "${CERT_DIR}/tls.crt" --key "${CERT_DIR}/tls.key" --cacert "${CERT_DIR}/ca.crt" --data-binary @test.iso.03 https://localhost:5001/api/v1/uploads/metalk8s
```
```json
{
  "solutionArchive": "metalk8s",
  "isCompleted": false,
  "sha256sum": "81c931664a390272a230b5c56b23c5652559f5bed085282b70fcc56e9c3c2b2f",
  "size": 30765056,
  "uploadedChunks":
  [
    {
      "rangeEndIndex": 6153011,
      "rangeSize": 6153012,
      "rangeStartIndex": 0
    },
    {
      "rangeEndIndex": 12306022,
      "rangeSize": 6153011,
      "rangeStartIndex": 6153012
    },
    {
      "rangeEndIndex": 18459033,
      "rangeSize": 6153011,
      "rangeStartIndex": 12306023
    },
    {
      "rangeEndIndex": 24612044,
      "rangeSize": 6153011,
      "rangeStartIndex": 18459034
    }
  ],
  "version": "1.25.3"
}
```
Last part uploaded:
```shell
export CERT_DIR="/tmp/certs"
curl -X PUT \
  --http1.1 \
  -H "X-Target-Version: 1.25.3" \
  -H "Content-Range: bytes 24612045-30765055/30765056" \
  -H "Content-Type: application/octet-stream" \
  --cert "${CERT_DIR}/tls.crt" \
  --key "${CERT_DIR}/tls.key" \
  --cacert "${CERT_DIR}/ca.crt" \
  --data-binary @test.iso.04 \
  https://localhost:5001/api/v1/uploads/metalk8s
```
```json
{
  "solutionArchive": "metalk8s",
  "isCompleted": true,
  "sha256sum": "81c931664a390272a230b5c56b23c5652559f5bed085282b70fcc56e9c3c2b2f",
  "size": 30765056,
  "uploadedChunks":
  [
    {
      "rangeEndIndex": 24612044,
      "rangeSize": 6153011,
      "rangeStartIndex": 18459034
    },
    {
      "rangeEndIndex": 30765055,
      "rangeSize": 6153011,
      "rangeStartIndex": 24612045
    },
    {
      "rangeEndIndex": 6153011,
      "rangeSize": 6153012,
      "rangeStartIndex": 0
    },
    {
      "rangeEndIndex": 12306022,
      "rangeSize": 6153011,
      "rangeStartIndex": 6153012
    },
    {
      "rangeEndIndex": 18459033,
      "rangeSize": 6153011,
      "rangeStartIndex": 12306023
    }
  ],
  "version": "1.25.3"
}
```
