# Contributing

Run `go test ./...` and `go vet ./...` before submitting a change. Go 1.26 is the current minimum toolchain.

For an authorization defect, include the complete relevant Role, ClusterRole, RoleBinding and ClusterRoleBinding objects, the identity and its groups, the request attributes, the expected result and the observed result. Use synthetic identities and configuration that can be published.

Core matching and aggregation live in `internal/evaluate`. Request partitions and semantic changes live in `internal/diff`. Input validation lives in `internal/input`. CLI behavior is tested through `cli.Run`, the normal executable entry point.

New permission semantics should include an independent regression case. Integration tests in `integration/` compare against a real Kubernetes API server. Use `go run ./scripts/integration` to create and clean up an isolated cluster.

Run `gofmt` on changed Go files. Preserve the documented exit codes and report schema. Describe compatibility implications and update examples when an input or output format changes.

Reports of crashes and wrong authorization conclusions should include the program version, platform, input scope and a minimal reproducer. An issue template is provided in `.github/ISSUE_TEMPLATE/bug.yml`.
