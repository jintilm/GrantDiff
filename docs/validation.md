# Validation record

The viewer preview is `1.1.0-preview.1`. Local viewer tests were run on 2026-11-21. Results describe the listed environments and cases. Machine-readable evidence records authorization, generated-policy, compaction, CRD, snapshot, package and browser checks independently. Candidate support and compatibility are defined in `compatibility.md`.

GitHub Actions verified viewer source commit `332dbaf9bd60e6cf428e84ff2faf2b6b1e44711b` successfully. All nine jobs passed, including offline browser interactions on extracted Windows/Linux packages, three Kubernetes versions and output parity. Downloaded artifacts confirmed 34 matching output fingerprints and the browser checks below. [Viewer verification](https://github.com/jintilm/GrantDiff/actions/runs/44892217511)

## Offline viewer checks

The viewer command embeds the interface and report schema in a single HTML file. Unit tests check literal report data containing script terminators, inline content hashes, normalized newlines, byte-order marks, input limits, input/output separation and error preservation of an existing output. CLI regression checks exercise a complete viewer document containing an incomplete diff result.

The Playwright verifier runs the extracted executable and opens its generated files with networking disabled. It covers identity, change type, risk and text filters, namespace complements, empty namespace requests, URL prefix exclusions, before/after evidence, source locations, path disclosure, empty/incomplete states, invalid schema fields, literal HTML-sensitive strings, language switching, pagination and keyboard selection. Desktop and 390-pixel layouts are captured for inspection. A synthetic 5,000-range report checks bounded list rendering and filtering.

The initial local Windows check used Chromium 156.0.8078.4 and completed its 5,000-range load-and-filter exercise in approximately 1.1 seconds. This is one observed browser run including file selection and test synchronization. It describes the sampled report and machine. Browser input is capped at 128 MiB, and larger or more complex reports depend on available memory. Generated HTML uses the report's completeness and authorization findings as supplied.

The hosted Windows and Ubuntu package jobs both passed with Chromium 156.0.8078.4. The 5,000-range exercise took 1,312 ms and 1,080 ms respectively in those runs. Both reported zero network requests and no browser console errors during offline interaction tests. Desktop and narrow-screen screenshots accompany the package-check artifacts.

## Environments

| Environment | Purpose |
| --- | --- |
| Windows x86-64, Go 1.26.4 | CLI development, unit tests, vet, Git input tests and release builds |
| Linux x86-64, golang:1.26-bookworm container | Native Go test execution |
| Ubuntu 24.04 x86-64 container | Standalone Linux executable and report comparison |
| Python 3.12 Debian container | Automated extracted Linux archive checks with networking disabled |
| Docker Linux engine 29.8.2, kind v0.33.0 | Isolated Kubernetes API oracle environments |

Dependencies were downloaded through an accessible Go proxy after the default endpoint timed out. Node images were retrieved through a mirror using the exact upstream release digests below. Test clusters are created with unique names and removed after each run.

A hosted Kubernetes 1.37 run detected changing RBAC or identity objects during the after-state capture for generated seed `101`. The consistency check rejected that capture. The integration harness now retries that specific consistency error for at most five captures, with 100 ms between attempts and the scenario context as a deadline. Regressions verify recovery after one changing capture, failure after persistent drift, immediate API-error propagation and cancellation. CI retains its full integration log. [Observed capture failure](https://github.com/jintilm/GrantDiff/actions/runs/44665243675/job/136052832588)

## Kubernetes authorization checks

The local matrix below was run for the preceding release candidate on 2026-11-20. The viewer's hosted verification reran the matrix successfully on 2026-11-21 and confirmed the same per-version counts, including one snapshot round trip per version. These hosted results are recorded separately in `verification-results.json`.

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

Each version saves a real captured snapshot using schema 1.1, loads it through the production snapshot reader, and compares policy, identities, namespaces and resource knowledge with the captured state. The matrix records three snapshot round trips in addition to the authorization counts above.

## Local regressions

Tests cover retained access, missing roles, incomplete identity groups, aggregate cycles, named-resource narrowing, subresource wildcard matching, URL prefix partitions, duplicate object keys, invalid fields, explicit namespaces, input path escapes, UTF-8 BOM, CRLF and Unicode paths. CLI tests exercise report generation, output replacement and exit codes, including incomplete wildcard reports. Query tests reject missing object names, unknown subresources and mixed URL/resource attributes. Git tests read both complete committed trees and verify that uncommitted working-tree content survives the review. List exports and multi-document input are checked for identical normalized policy objects and preserved source locations.

A partition invariant test checks that each sampled request maps to exactly one reported request class and that each rule has the same truth value throughout that class. This covers complement regions used for wildcard and future-resource permissions.

CRD regressions check served-version subresource unions, invalid or duplicate definitions, scope conflicts and the normal catalog, diff and query entry points. A before/after catalog scope conflict produces an incomplete result. Report regressions check summary counts, evidence attribution, Markdown escaping, expandable details and identical CI failure decisions with `--summary`.

The ExternalDNS fixture pins upstream RBAC bytes to v0.15.1 commit `c55b1a1127df024b95f927869a1bcd662ec4337a`. Tests verify source hashes and compare the complete Markdown output with `examples/external-dns/expected.md`. A locally authored candidate rule adds Secret reads and must produce high-risk added ranges through the expected binding and role. This is a synthetic modification of real upstream configuration.

The compact ExternalDNS report has three ranges covering the same requests as the original nine, with a checked expected report in `examples/external-dns/expected-compact.md`. CLI regressions verify schema-compatible JSON, Markdown, summary mode, high-risk exit code 1 and budget-exhausted exit code 3. Compaction tests sample request coverage, excluded object names and nested URL prefixes, and check deterministic IDs, representative requests, idempotence and input immutability. Subresource partitions require the parent object name and the resource's namespace when applicable.

The second pinned controller fixture uses kube-state-metrics v2.14.0 commit `e763ea2aa37e7fa507775a8506387aaa95ff17d1`. Its locally authored candidate removes Secret list/watch. The compact report records two removed ranges and 24 evidence-path changes for grants in the edited rule. Tests check upstream hashes, subject, resource, verbs, evidence and the complete expected Markdown.

The RC review identified snapshot and candidate-input corrections. Snapshot schema 1.1 now verifies the complete policy/identity and catalog files before parsing the same bytes. CLI tests cover intact and legacy snapshots, modified policy, modified identities, changed catalog scope and missing hashes. Candidate deletion of every policy now requires the documented explicit empty-after flag, with a regression checking preservation of the baseline.

Native Windows package checks exposed short working-directory aliases in runner temporary paths. Both archive verification and manifest input loading now resolve the absolute root before containment comparisons. A Windows regression reproduces the original input-boundary rejection using `GetShortPathName`, then verifies successful loading and relative evidence paths with the correction. Genuine path escapes retain their existing regression. Package-test logs are uploaded on both successful and failed runs.

Reproduction commands are available in README. Environment-specific tests are opt-in through the isolated integration runner. A plain `go test ./...` skips the live-cluster test when no isolated kubeconfig is supplied.

## Performance

The measurements in this section were collected for 0.6.0-preview and retained as the preceding performance baseline. RC changes focus on snapshot integrity and release verification. The authorization and compaction algorithms retain the measured workload definitions.

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

## Candidate boundaries

- Windows x86-64 and Linux x86-64 have executable builds and local runtime checks. ARM64 and macOS remain planned targets.
- Ordinary user group memberships require explicit input. External identity systems and additional Kubernetes authorizers need their own data sources.
- Acyclic aggregate roles use stable-state semantics. Cycles produce incomplete results, and snapshot queries also use stable-state aggregation.
- Broad wildcard changes may produce many detailed ranges. `--compact` combines exact cells with compatible complements while retaining full evidence and schema 1.0. Separate exact values and nested URL prefixes may remain as multiple ranges. The comparison budget stops with an incomplete result.
- Resource catalogs come from explicit files, discovery and v1 CRD metadata. CRD subresources are unioned across served versions. CRD extraction validates fields needed for catalog generation, and Kubernetes validates deployment configuration.
- The public repository includes CI definitions and an integration example. Hosted verification runs execute Windows and Linux unit jobs together with the three-version Kubernetes matrix. [GitHub verification runs](https://github.com/jintilm/GrantDiff/actions/workflows/ci.yml)

The candidate records these limits in its support contract. Feedback from outside users remains pending and will inform the stable-release decision.

## Release candidate acceptance

The candidate milestone covers the support contract, snapshot integrity, permission narrowing in a second pinned controller and automated native archive verification. Existing scope, typed-list, CRD, aggregation, wildcard and incomplete-result regressions run with the candidate. Workflows are checked with actionlint v1.7.7, and the Python verifier passes Ruff checks.

Machine-readable local results are stored in `verification-results.json`. Linux unit tests execute with networking disabled and dependencies mounted from the local module cache. The package verifier runs from the extracted archive in a Unicode path containing spaces. It checks four scenarios in default and compact text, Markdown and JSON, golden reports, Schema 1.0, catalog generation, positive/negative/unknown queries, exit codes 0 through 4, read-only input, BOM/CRLF, checksums, documentation and notices. The viewer extends its 29 output fingerprints with four embedded HTML reports and the empty HTML viewer. The packaged `viewer.html` must equal the executable-generated empty page.

Every branch verification builds one pair of archives and runs them on native Windows and Ubuntu runners before comparing fingerprints. Version tags run that same complete workflow, then publish the tested archives and checksums. Draft publication allows interrupted uploads to be retried before the release becomes visible. A candidate's public CI and downloadable artifacts provide reproducible evidence for review.
