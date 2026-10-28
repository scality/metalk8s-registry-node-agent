# AGENTS.md

This is a **Go Kubernetes operator** for managing solution archive distribution across MetalK8s cluster nodes. It contains:

- A `NodeSolutionArchive` CRD and controller (`internal/controller/`)
- Validation webhooks (`internal/webhook/`)
- Clean architecture with domain/usecase/service/infrastructure layers (`pkg/`)
- OpenAPI-generated HTTP handlers for internal and external APIs (`pkg/presentation/http/`)
- Built with Kubebuilder, controller-runtime, Ginkgo/Gomega for testing
- Linted with golangci-lint v2 (revive, staticcheck, errcheck, govet, etc.)

**Before you change anything, open and read `DESIGN.md` in full.** It describes the architecture,
so you work from it instead of guessing.

`CLAUDE.md` is the same guide for Claude Code -- it imports `DESIGN.md` into context directly
(`@DESIGN.md`); this file is the equivalent for every other agent, pointing at it explicitly.
Keep the two in sync.
