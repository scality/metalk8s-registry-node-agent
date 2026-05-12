# metalk8s-registry-node-agent

This is a **Go Kubernetes operator** for managing solution archive distribution across MetalK8s cluster nodes. It contains:

- A `NodeSolutionArchive` CRD and controller (`internal/controller/`)
- Validation webhooks (`internal/webhook/`)
- Clean architecture with domain/usecase/service/infrastructure layers (`pkg/`)
- OpenAPI-generated HTTP handlers for internal and external APIs (`pkg/presentation/http/`)
- Built with Kubebuilder (operator-sdk), controller-runtime, Ginkgo/Gomega for testing
- Linted with golangci-lint v2 (revive, staticcheck, errcheck, govet, etc.)
- Architecture is described here: @DESIGN.md
