FROM mcr.microsoft.com/devcontainers/go:1.27-trixie@sha256:17bf9e845391f8dfad8d967f203ecd66e2be1c1b67016e56292ea37001edc23f

# genisoimage is used to generate an ISO file for test E2E
RUN apt-get update && apt-get install -y --no-install-recommends \
    genisoimage \
  && rm -rf /var/lib/apt/lists/*

USER vscode

# renovate: datasource=github-releases depName=operator-framework/operator-sdk extractVersion=^v(?<version>.*)$
ARG OPERATOR_SDK_VERSION=1.42.2

RUN curl -sSfLO https://github.com/operator-framework/operator-sdk/releases/download/v${OPERATOR_SDK_VERSION}/operator-sdk_linux_amd64 && \
    chmod +x operator-sdk_linux_amd64 && \
    sudo mv operator-sdk_linux_amd64 /usr/local/bin/operator-sdk

# renovate: datasource=github-releases depName=golangci/golangci-lint extractVersion=^v(?<version>.*)$
ARG GOLANGCI_LINT_VERSION=2.13.2

RUN curl -sSfL  https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sudo sh -s -- v${GOLANGCI_LINT_VERSION}

# renovate: datasource=github-releases depName=kubernetes-sigs/kind extractVersion=^v(?<version>.*)$
ARG KIND_VERSION=0.33.0

RUN curl -sSfLO https://kind.sigs.k8s.io/dl/v${KIND_VERSION}/kind-linux-amd64 && \
    chmod +x kind-linux-amd64 && \
    sudo mv kind-linux-amd64 /usr/local/bin/kind

# renovate: datasource=github-releases depName=kubernetes/kubernetes extractVersion=^v(?<version>.*)$
ARG KUBECTL_VERSION=1.37.0

RUN curl -sSfLO https://dl.k8s.io/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl && \
    chmod +x kubectl && \
    sudo mv kubectl /usr/local/bin/kubectl
