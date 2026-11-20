# Contributing

Run `go test ./...` and `go vet ./...` before submitting a change. Go 1.26 is the current minimum toolchain.

For an authorization defect, include the complete relevant Role, ClusterRole, RoleBinding and ClusterRoleBinding objects, the identity and its groups, the request attributes, the expected result and the observed result. Use synthetic identities and configuration that can be published.

Core matching and aggregation live in `internal/evaluate`. Request partitions and semantic changes live in `internal/diff`. Input validation lives in `internal/input`. CLI behavior is tested through `cli.Run`, the normal executable entry point.

New permission semantics should include an independent regression case. Integration tests in `integration/` compare against a real Kubernetes API server. Use `go run ./scripts/integration` to create and clean up an isolated cluster.

Generated policy tests use fixed seeds and write standalone failure inputs under `.work/failures/seed-N/`. Set `GRANTDIFF_TEST_SEED=N` and run the integration command to replay that case. The saved manifests, identities and catalog also work with the normal `diff` entry point. CI uploads verification results and failure fixtures for each Kubernetes version.

The ExternalDNS fixture is pinned to an upstream commit. Changes to its copied files require updating provenance and reviewing the expected report. Report tests cover summary counts, escaping and the CI exit code when `--summary` is selected. CRD tests cover catalog merging, served versions and scope conflicts, with an independent API discovery comparison.

Range compaction must preserve request coverage, decisions, risk findings and source evidence. The compact regression suite checks exclusions, URL prefixes, input immutability, deterministic output and idempotence. Generated cluster tests compare the complete matching change semantics before and after compaction. Exercise both forms of the wildcard benchmark when changing partitioning or compaction.

The integration harness retries `cluster.ErrCaptureChanged` up to five captures while controllers settle. Each retry is logged and bounded by the scenario context. Other API errors return immediately, and permission assertions run once against the captured state. CI uploads the complete integration log alongside result files.

Run `gofmt` on changed Go files. Preserve the documented exit codes and report schema. Describe compatibility implications and update examples when an input or output format changes.

Reports of crashes and wrong authorization conclusions should include the program version, platform, input scope and a minimal reproducer. An issue template is provided in `.github/ISSUE_TEMPLATE/bug.yml`.
