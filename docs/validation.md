# Validation record

The preview release is `0.6.0-preview`. Local tests were run on 2026-11-20. Results describe the listed environments and cases. Machine-readable evidence records authorization, generated-policy, compaction, CRD and package checks independently.

The preceding 0.5 release passed all five hosted jobs at source commit `2e2d4b355b7a3f1a39dd63f1078a708191bfc696`. Hosted verification for 0.6 will be recorded after the source is pushed. [Previous hosted verification](https://github.com/jintilm/GrantDiff/actions/runs/44557124555)

## Environments

| Environment | Purpose |
| --- | --- |
| Windows x86-64, Go 1.26.4 | CLI development, unit tests, vet, Git input tests and release builds |
| Linux x86-64, golang:1.26-bookworm container | Native Go test execution |
| Ubuntu 24.04 x86-64 container | Standalone Linux executable and report comparison |
| Docker Linux engine 29.8.2, kind v0.33.0 | Isolated Kubernetes API oracle environments |

Dependencies were downloaded through an accessible Go proxy after the default endpoint timed out. Node images were retrieved through a mirror using the exact upstream release digests below. Test clusters are created with unique names and removed after each run.

## Kubernetes authorization checks

| Kubernetes | Node image SHA-256 | Authorization comparisons | Generated diff checks | Compact coverage checks | Actual HTTP requests |
| --- | --- | --- | --- | --- | --- |
| v1.35.8 | `07b2536e30b803ed61d1677a79df6115f798ce64c80f9e22f6ed45afd09323c0` | 3,763 | 1,248 | 1,248 | 3 |
| v1.36.4 | `099e049362a1526b2db71494e1947aae99bd16290d7c895f2b7ea312e3cbfaed` | 3,763 | 1,248 | 1,248 | 3 |
| v1.37.0 | `a1ed56cfb0e7b93589bdf97c8cd566405a265939e3620fc4f5de89adff580ae5` | 3,763 | 1,248 | 1,248 | 3 |

Each scenario loads role and binding objects into an isolated cluster, captures default permissions, and compares three identities against 209 request attribute sets in each of two states. The generated request sequence uses PCG seeds `20261119` and `7`. It includes positive and negative decisions across namespaces, resources, names, subresources, API groups and non-resource URLs.

The state transition removes a direct ServiceAccount binding while retaining its group binding, and changes a leaf role that feeds an aggregated ClusterRole. The diff must report retained access, an added aggregated permission and a removed aggregated permission. Real ConfigMap list calls verify that a matching name selector is accepted and an unfiltered list is forbidden.

The additional wildcard scenario performs eight comparisons around a namespace binding to a ClusterRole with an API-group wildcard. Catalogued API groups must be represented separately, namespace-scoped Deployment access must appear, and cluster-scoped CertificateSigningRequest access must respect its scope. Any indeterminate report must be marked incomplete.

Each version also runs eight generated policy scenarios with PCG seed pairs `(N, 41)`, using N values `17`, `73`, `101`, `307`, `997`, `65537`, `20261119` and `20261120`. Generated roles, direct and group bindings, ServiceAccount groups, namespace bindings to ClusterRoles, resource-name restrictions and wildcards are exercised. Every scenario mutates a role and deletes one binding. The suite compares 2,496 before/after authorization decisions and 1,248 diff classifications per version. Each version records 288 granted and 2,208 ungranted generated decisions.

Every generated request also compares the original and compact reports. The matching changes must carry identical subjects, types, before/after decisions, risk findings and complete source evidence. This adds 1,248 compact checks per version and 3,744 across the matrix. The three versions collectively perform 11,289 authorization comparisons and 3,744 original diff checks.

Set `GRANTDIFF_TEST_SEED` to run one generated seed. Failure fixtures include before/after manifests, identities, catalog, request and report under `.work/failures/seed-N/`. GitHub CI uploads these alongside machine-readable results. The seed suite is a finite reproducible sample of policy combinations.

The CRD suite adds five authorization comparisons per version. It creates a real namespaced CRD, checks its scope and status subresource against API discovery, and performs one actual status patch with the evaluated identity. The four raw typed RBAC export checks also run on each version.

## Local regressions

Tests cover retained access, missing roles, incomplete identity groups, aggregate cycles, named-resource narrowing, subresource wildcard matching, URL prefix partitions, duplicate object keys, invalid fields, explicit namespaces, input path escapes, UTF-8 BOM, CRLF and Unicode paths. CLI tests exercise report generation, output replacement and exit codes, including incomplete wildcard reports. Query tests reject missing object names, unknown subresources and mixed URL/resource attributes. Git tests read both complete committed trees and verify that uncommitted working-tree content survives the review. List exports and multi-document input are checked for identical normalized policy objects and preserved source locations.

A partition invariant test checks that each sampled request maps to exactly one reported request class and that each rule has the same truth value throughout that class. This covers complement regions used for wildcard and future-resource permissions.

CRD regressions check served-version subresource unions, invalid or duplicate definitions, scope conflicts and the normal catalog, diff and query entry points. A before/after catalog scope conflict produces an incomplete result. Report regressions check summary counts, evidence attribution, Markdown escaping, expandable details and identical CI failure decisions with `--summary`.

The ExternalDNS fixture pins upstream RBAC bytes to v0.15.1 commit `c55b1a1127df024b95f927869a1bcd662ec4337a`. Tests verify source hashes and compare the complete Markdown output with `examples/external-dns/expected.md`. A locally authored candidate rule adds Secret reads and must produce high-risk added ranges through the expected binding and role. This is a synthetic modification of real upstream configuration.

The compact ExternalDNS report has three ranges covering the same requests as the original nine, with a checked expected report in `examples/external-dns/expected-compact.md`. CLI regressions verify schema-compatible JSON, Markdown, summary mode, high-risk exit code 1 and budget-exhausted exit code 3. Compaction tests sample request coverage, excluded object names and nested URL prefixes, and check deterministic IDs, representative requests, idempotence and input immutability. Subresource partitions require the parent object name and the resource's namespace when applicable.

Reproduction commands are available in README. Environment-specific tests are opt-in through the isolated integration runner. A plain `go test ./...` skips the live-cluster test when no isolated kubeconfig is supplied.

## Performance

The reproducible workload has 10,000 RBAC objects in each state, 1,000 ServiceAccount identities and one modified role. It evaluates 33 request classes and reports two changes. File-mode measurements include policy and identity input loading, comparison, rendering and writing the JSON report.

| Windows file-mode measurement | 0.5.0 preview | 0.6.0 preview |
| --- | --- | --- |
| Timed pipeline | 1,564 ms | 1,494 ms |
| Sampled process peak working set | 167 MiB | 199 MiB |

The wildcard workload changes one shared ClusterRole used by 25 ServiceAccounts. Each state contains 26 RBAC objects, with 20 catalogued resource types and resource/verb wildcards. It compares 49,450 request classes.

| Windows wildcard file-mode measurement | Original ranges | With `--compact` |
| --- | --- | --- |
| Reported ranges | 46,700 | 5,225 |
| JSON report bytes | 71,716,493 | 7,916,365 |
| Load | 18 ms | 5 ms |
| Compare | 436 ms | 344 ms |
| Compact | 0 ms | 685 ms |
| Render and write | 631 ms | 57 ms |
| Total pipeline | 1,087 ms | 1,093 ms |
| Sampled process peak working set | 387 MiB | 223 MiB |

The compact output is about 89% smaller in this workload. Compaction adds analysis work and reduces serialization work. With `--max-classes 1000 --compact`, the same workload stops at 1,000 classes, records an incomplete result and retains the budget diagnostic. It emits 134 compacted ranges from 890 changes observed before the limit.

Working-set measurements sample Windows process peak counters every 10 ms and cover the entire benchmark process, including fixture generation. Fixture generation and compilation are outside the timed pipeline. These are individual local runs, with timings affected by caches and other machine activity. The measured workloads meet the design's provisional 10-second and 512-MiB targets on this machine. Larger policies and different input shapes require separate characterization.

Run `go run ./scripts/benchmark --mode files` to time the sparse pipeline. Add `--scenario wildcard`, optionally `--compact`, for the shared-role workload. `--mode memory` measures analysis after fixture construction. File mode records loading, comparison, compaction, output, report bytes, Go heap and allocation counters. The recorded process working-set samples were collected by an external process monitor.

## Preview boundaries

- Windows x86-64 and Linux x86-64 have executable builds and local runtime checks. ARM64 and macOS remain planned targets.
- Ordinary user group memberships require explicit input. External identity systems and additional Kubernetes authorizers need their own data sources.
- Acyclic aggregate roles use stable-state semantics. Cycles produce incomplete results, and snapshot queries also use stable-state aggregation.
- Broad wildcard changes may produce many detailed ranges. `--compact` combines exact cells with compatible complements while retaining full evidence and schema 1.0. Separate exact values and nested URL prefixes may remain as multiple ranges. The comparison budget stops with an incomplete result.
- Resource catalogs come from explicit files, discovery and v1 CRD metadata. CRD subresources are unioned across served versions. CRD extraction validates fields needed for catalog generation, and Kubernetes validates deployment configuration.
- The public repository includes CI definitions and an integration example. Hosted verification runs execute Windows and Linux unit jobs together with the three-version Kubernetes matrix. [GitHub verification runs](https://github.com/jintilm/GrantDiff/actions/workflows/ci.yml)

The preview version records these limits while the remaining v1.0 acceptance goals in the design document continue to guide development.

## 0.6 milestone

The implemented milestone covers optional range compaction, subresource request boundaries and measured wildcard workloads. The preceding input-scope, typed-list, CRD and incomplete-result regressions remain in the test suite. Further stable-release work can broaden real-world fixture coverage and characterize larger workloads using the reproducible harness.

Machine-readable local results are stored in `verification-results.json`. Linux unit tests executed with networking disabled and dependencies mounted from the local module cache. Snapshot-based candidate changes and stale baseline rejection retain their existing regressions. Package validation compared Windows and Ubuntu JSON and Markdown across retained-access, CRD and ExternalDNS scenarios in both default and compact modes. It checked JSON Schema 1.0, catalog generation, summary high-risk exit code 1, compact budget exit code 3, checksums and documentation links. Both extracted archives ran the compact ExternalDNS review and reported the expected version.
