# Changelog

## 0.6.0-preview — 2026-11-20

- Add `diff --compact` to merge compatible request ranges with identical decisions, risks and complete evidence using schema 1.0 selectors.
- Preserve range coverage, representative requests, exclusions, diagnostics and CI failure conditions during compaction.
- Exclude subresource request ranges without a parent name or required namespace.
- Compare compact and original change semantics for every generated Kubernetes request sample.
- Add a compact ExternalDNS expected report and normal CLI regressions for high-risk and incomplete results.
- Extend the benchmark with a wildcard workload, per-stage timing, report size and budget-limited runs.
- Retry observed snapshot drift within the isolated integration harness, with five attempts and regression checks for persistent drift, API failures and cancellation.
- Preserve integration logs in CI artifacts with the test command's exit status.

## 0.5.0-preview — 2026-11-19

- Derive resource catalogs from v1 CRDs, including served status and scale subresources, and export them with `grantdiff catalog`.
- Merge explicit, snapshot and CRD resource knowledge while rejecting conflicting scopes.
- Add per-identity change summaries, new-access attribution, expandable Markdown evidence and `--summary` with the same CI failure behavior.
- Add eight deterministic generated policy scenarios, authorization and diff comparisons, seed replay and saved failure fixtures.
- Verify CRD-derived scope and subresources against Kubernetes discovery and an actual status patch.
- Include a pinned ExternalDNS RBAC example with source hashes, its upstream license and an expected report.
- Upload per-version verification results and failure fixtures from GitHub CI.

## 0.4.2-preview — 2026-11-19

- Reject missing or null resource-catalog scopes instead of silently treating resources as cluster-scoped and omitting namespace permission changes.
- Decode typed Kubernetes RBAC lists with inherited item kind and API version, while rejecting conflicting item metadata and preserving source positions.
- Add CLI regressions for both input paths and compare four raw typed API exports with the snapshot model in integration tests.

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
