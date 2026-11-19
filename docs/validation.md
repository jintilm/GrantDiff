# Validation record

The local preview release is `0.4.1-preview`. Tests were run on 2026-11-19. Results here describe the listed environments and cases.

The project now uses the GrantDiff name, the `grantdiff` executable and the Go module `github.com/jintilm/GrantDiff`. After renaming, Windows and Linux unit tests, `go vet`, Windows and Ubuntu archive execution, report equality and the Kubernetes v1.37.0 integration suite were rerun successfully. The v1.35.8 and v1.36.4 entries below retain the preceding verification of the same authorization implementation.

GitHub hosted verification also passed for initial source commit `45a0441fd26cb7f94796f9cc862decc94d597af8`. All five jobs succeeded, covering Windows, Ubuntu and Kubernetes v1.35.8, v1.36.4 and v1.37.0 under the GrantDiff module and command name. [Hosted verification run](https://github.com/jintilm/GrantDiff/actions/runs/44538682781)

## Environments

| Environment | Purpose |
| --- | --- |
| Windows x86-64, Go 1.26.4 | CLI development, unit tests, vet, Git input tests and release builds |
| Linux x86-64, golang:1.26-bookworm container | Native Go test execution |
| Ubuntu 24.04 x86-64 container | Standalone Linux executable and report comparison |
| Docker Linux engine 29.8.2, kind v0.33.0 | Isolated Kubernetes API oracle environments |

Dependencies were downloaded through an accessible Go proxy after the default endpoint timed out. Node images were retrieved through a mirror using the exact upstream release digests below. Test clusters are created with unique names and removed after each run.

## Kubernetes authorization checks

| Kubernetes | Node image SHA-256 | Authorization comparisons | Actual HTTP requests |
| --- | --- | --- | --- |
| v1.35.8 | `07b2536e30b803ed61d1677a79df6115f798ce64c80f9e22f6ed45afd09323c0` | 1,262 | 2 |
| v1.36.4 | `099e049362a1526b2db71494e1947aae99bd16290d7c895f2b7ea312e3cbfaed` | 1,262 | 2 |
| v1.37.0 | `a1ed56cfb0e7b93589bdf97c8cd566405a265939e3620fc4f5de89adff580ae5` | 1,262 | 2 |

Each scenario loads role and binding objects into an isolated cluster, captures default permissions, and compares three identities against 209 request attribute sets in each of two states. The generated request sequence uses PCG seeds `20261119` and `7`. It includes positive and negative decisions across namespaces, resources, names, subresources, API groups and non-resource URLs.

The state transition removes a direct ServiceAccount binding while retaining its group binding, and changes a leaf role that feeds an aggregated ClusterRole. The diff must report retained access, an added aggregated permission and a removed aggregated permission. Real ConfigMap list calls verify that a matching name selector is accepted and an unfiltered list is forbidden.

The additional wildcard scenario performs eight comparisons around a namespace binding to a ClusterRole with an API-group wildcard. Catalogued API groups must be represented separately, namespace-scoped Deployment access must appear, and cluster-scoped CertificateSigningRequest access must respect its scope. Any indeterminate report must be marked incomplete.

The original integration run hit the client rate limiter timeout. The test client now uses a fixed throughput appropriate for the isolated cluster, and the complete checks were rerun successfully. The production client configuration retains its standard request limits.

## Local regressions

Tests cover retained access, missing roles, incomplete identity groups, aggregate cycles, named-resource narrowing, subresource wildcard matching, URL prefix partitions, duplicate object keys, invalid fields, explicit namespaces, input path escapes, UTF-8 BOM, CRLF and Unicode paths. CLI tests exercise report generation, output replacement and exit codes, including incomplete wildcard reports. Query tests reject missing object names, unknown subresources and mixed URL/resource attributes. Git tests read both complete committed trees and verify that uncommitted working-tree content survives the review. List exports and multi-document input are checked for identical normalized policy objects and preserved source locations.

A partition invariant test checks that each sampled request maps to exactly one reported request class and that each rule has the same truth value throughout that class. This covers complement regions used for wildcard and future-resource permissions.

Reproduction commands are available in README. Environment-specific tests are opt-in through the isolated integration runner. A plain `go test ./...` skips the live-cluster test when no isolated kubeconfig is supplied.

## Performance

The reproducible workload has 10,000 RBAC objects in each state, 1,000 ServiceAccount identities and one modified role. It evaluates 33 request classes and reports two changes. The earlier in-memory comparison took 58 ms. File-mode measurements now include policy and identity input loading, comparison, rendering and writing the JSON report.

| Windows file-mode measurement | Before List optimization | After List optimization |
| --- | --- | --- |
| Timed pipeline | 2,429 ms | 1,507 ms |
| Sampled process peak working set | 786 MiB | 171 MiB |

The improvement removes a redundant whole-List conversion before per-item decoding. Working-set measurements sampled Windows process peak counters every 10 ms and cover the entire benchmark process, including fixture generation. Fixture generation and compilation are outside the timed pipeline. The fixed workload meets the design's provisional 10-second and 512-MiB targets on this machine. Broad wildcard changes and different input shapes require separate characterization.

Run `go run ./scripts/benchmark --mode files` to time the full pipeline, or use `--mode memory` for analysis only. The benchmark reports Go heap and allocation counters. The recorded process working-set samples were collected by an external process monitor.

## Preview boundaries

- Windows x86-64 and Linux x86-64 have executable builds and local runtime checks. ARM64 and macOS remain planned targets.
- Ordinary user group memberships require explicit input. External identity systems and additional Kubernetes authorizers need their own data sources.
- Acyclic aggregate roles use stable-state semantics. Cycles produce incomplete results, and snapshot queries also use stable-state aggregation.
- Broad wildcard changes may produce lengthy reports. The comparison budget stops work with an incomplete result instead of exhausting unbounded work.
- Resource catalogs are supplied explicitly or captured from discovery. CRD manifest metadata is currently handled through an explicit catalog.
- The public repository includes CI definitions and an integration example. Hosted verification runs execute Windows and Linux unit jobs together with the three-version Kubernetes matrix. [GitHub verification runs](https://github.com/jintilm/GrantDiff/actions/workflows/ci.yml)

The preview version records these limits while the remaining v1.0 acceptance goals in the design document continue to guide development.

## Readiness after the 0.4.1 review

The review reproduced and fixed inconsistent completion status, missing wildcard catalog partitions and incomplete query validation. It also added full-pipeline performance evidence and reduced List input memory use. These are covered by targeted regressions and the runtime checks above.

The next development stage should extend policy-combination coverage and report usability. Useful acceptance cases include randomized roles and bindings compared with the API server, public controller RBAC examples with reproducible expected results, readable summaries for broad changes, and resource-catalog generation from CRD manifests. Stable-release decisions should use these cases together with the existing verification matrix.

Machine-readable local results are stored in `verification-results.json`. Package validation compared Windows and Ubuntu JSON output, validated the report against `report.schema.json`, checked archive checksums and executable permissions, ran the extracted Windows package, exercised snapshot-based upserts and verified that a stale baseline is rejected. Linux unit tests executed with networking disabled and dependencies mounted from the local module cache.
