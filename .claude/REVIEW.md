# Review criteria

Read by the `/review-pr` skill (Scality agent hub) and by anyone reviewing by hand.
Flag problems only — see "What not to flag" at the end.

## What this repo is

`metalk8s-registry-node-agent` is the node-side half of solution archive
distribution: a Go Kubernetes agent that stores archives on the node and serves
them, reconciling the custom resources that track replication and serving state.
Layers follow the domain/usecase/service/infrastructure split.

## Criteria

| Area | What to check |
|------|---------------|
| Error wrapping | Use `fmt.Errorf("...: %w", err)`, not `%v`; preserve error chains for `errors.Is`/`errors.As` |
| Context propagation | Pass `context.Context` through call chains, respect cancellation |
| Goroutine leaks | Ensure goroutines have exit conditions, use `errgroup` where appropriate |
| Kubernetes operator | RBAC scoping, reconciler idempotency, status subresource updates, proper use of controller-runtime |
| Interface compliance | Check that implementations satisfy interfaces at compile time |
| CRD changes | Backward-compatible schema evolution, proper markers, deep copy generation |
| Clean architecture | Respect layer boundaries (domain/usecase/service/infrastructure), no import cycles |
| Test quality | Ginkgo/Gomega assertions, table-driven tests where appropriate, no test pollution |
| Security | OWASP-relevant issues: path traversal in file operations, injection in shell commands, proper TLS usage |
| Breaking changes | Changes to CRD spec, API contracts, or public Go interfaces |
| Code changes | Check that the documentation is still relevant regarding the changes |

## What not to flag

- Anything the linters already own: `golangci-lint` (revive, staticcheck, errcheck,
  govet), `gofmt`, `goimports`.
- Generated files (deepcopy, CRD manifests) except when they are stale with respect
  to the sources changed in the same PR.
- Markdown or comment wording preferences.
- Refactors unrelated to the PR's purpose.
