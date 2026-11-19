# Changelog

## 0.4.1-preview — 2026-11-19

- Adopted GrantDiff as the project name, `grantdiff` as the CLI and `github.com/jintilm/GrantDiff` as the Go module path.
- Corrected wildcard reports that contained indeterminate results while returning a successful CI exit code.
- Partitioned wildcard API groups using catalogued resource scopes to avoid attributing cluster-scoped access to namespace bindings.
- Added query validation for required object names, known subresources and conflicting non-resource attributes.
- Removed redundant whole-List conversion before per-object parsing, reducing memory use for exported policy lists.
- Added file-loading-through-report-writing benchmarks and extended Kubernetes comparisons with wildcard scope regressions.

## 0.4.0-preview — 2026-11-19

- Added offline manifest and complete Git-tree comparisons.
- Added user, group and ServiceAccount authorization paths, request matching and acyclic ClusterRole aggregation.
- Added symbolic permission ranges, retained-path detection, diagnostics and configurable CI failure conditions.
- Added named-context snapshots and digest-guarded candidate changes.
- Added text, Markdown and JSON reports with a report schema.
- Added Windows and Linux builds, packaged examples, dependency licenses and checksums.
- Added Kubernetes API comparisons on v1.35.8, v1.36.4 and v1.37.0, plus local parser, authorization, partition and CLI regressions.

The current scope and preview limits are recorded in `docs/validation.md`.
