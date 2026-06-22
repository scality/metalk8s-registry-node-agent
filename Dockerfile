################################################
########### Build the manager binary ###########
################################################
# Build the manager binary
FROM golang:1.26.4-alpine3.23@sha256:f23e8b227fb4493eabe03bede4d5a32d04092da71962f1fb79b5f7d1e6c2a17f AS builder
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

################################################
########### Build the setup binary #############
################################################
# Build the setup binary
FROM golang:1.26.4-alpine3.23@sha256:f23e8b227fb4493eabe03bede4d5a32d04092da71962f1fb79b5f7d1e6c2a17f AS builder-setup
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests and the source
COPY cmd/setup/go.mod go.mod
COPY cmd/setup/ cmd/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o setup \
    cmd/main.go

################################################
########### Build the final image ##############
################################################
# Use Alpine as minimal base image with mount/umount utilities
# Alpine includes util-linux package with mount/umount by default
FROM alpine:3.23@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11
WORKDIR /
# Install util-linux for mount/umount utilities, libcap for capabilities
# and ca-certificates for HTTPS connections
RUN apk add --no-cache util-linux ca-certificates libcap && \
    # Create non-root user matching the distroless nonroot user (even if we use alpine as base image)
    adduser -D -u 65532 -g 65532 nonroot
COPY --from=builder /workspace/manager .
COPY --from=builder-setup /workspace/setup .
# capabilities required to create loop devices and mount/umount files
RUN setcap 'cap_sys_admin,cap_mknod=+ep' /manager
USER 65532:65532

ENTRYPOINT ["/manager"]
