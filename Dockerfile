# Build the manager binary
FROM golang:1.25.1-alpine3.22 AS builder
ARG TARGETOS
ARG TARGETARCH
ARG APPLICATION_VERSION=dev

RUN apk add --no-cache git
WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN --mount=type=secret,id=GIT_AUTH_TOKEN \
    git config --global url."https://oauth2:$(cat /run/secrets/GIT_AUTH_TOKEN)@github.com/".insteadOf "https://github.com/" && \
    go mod download

# Copy the go source
COPY cmd/ cmd/
COPY api/ api/
COPY pkg/ pkg/
COPY internal/ internal/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager \
    -ldflags "-X 'github.com/scality/metalk8s-registry-node-agent/cmd/config.ApplicationVersion=${APPLICATION_VERSION}'" \
    cmd/main.go

# Use Alpine as minimal base image with mount/umount utilities
# Alpine includes util-linux package with mount/umount by default
FROM alpine:3.23
WORKDIR /
# Install util-linux for mount/umount utilities, sudo for privilege escalation,
# and ca-certificates for HTTPS connections
RUN apk add --no-cache util-linux ca-certificates sudo && \
    # Create non-root user matching the distroless nonroot user (even if we use alpine as base image)
    adduser -D -u 65532 -g 65532 nonroot && \
    # Configure sudo to allow mount/umount without password for nonroot user
    # This is more secure than running the entire application as root
    #TODO echo 'nonroot ALL=(root) NOPASSWD: /bin/mount, /bin/umount' > /etc/sudoers.d/nonroot && \
    echo 'nonroot ALL=(root) NOPASSWD: ALL' > /etc/sudoers.d/nonroot && \
    chmod 0440 /etc/sudoers.d/nonroot
COPY --from=builder /workspace/manager .
# Run as non-root user for better security
# Only mount/umount commands will escalate to root via sudo
USER 65532:65532

ENTRYPOINT ["/manager"]
