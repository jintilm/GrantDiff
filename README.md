# GrantDiff

Explain Kubernetes RBAC changes with permission paths.

Review Kubernetes RBAC changes with the paths that grant each permission. Run locally on Windows or Linux, or use the same CLI in CI.

Removing a binding can leave a service account authorized through a group. The included example reports `retained_via_other_path` and identifies the remaining binding.

```text
ServiceAccount finance/report-bot
  → Group system:serviceaccounts:finance
  → RoleBinding finance/group-read
  → Role finance/report-reader
  → get configmaps/billing-config
```

[中文说明](README.zh-CN.md) · [Design](docs/design.zh-CN.md) · [Compatibility](docs/compatibility.md) · [Validation](docs/validation.md)

Version `1.0.0-rc.1` is the first release candidate. It defines the candidate support contract, verifies complete snapshot files and automates native archive checks. Pinned ExternalDNS and kube-state-metrics examples demonstrate added and removed permissions with reviewed expected reports. [Candidate notes](docs/releases/v1.0.0-rc.1.md)

## Build and try

Release archives include the executable, example inputs and documentation. After extracting an archive, run the example with `./grantdiff` on Linux or `./grantdiff.exe` on Windows, followed by the `diff` arguments below.

The release page provides both platform archives and `SHA256SUMS`. Verify a downloaded archive with `Get-FileHash -Algorithm SHA256` on Windows or `sha256sum` on Linux. [Release downloads](https://github.com/jintilm/GrantDiff/releases)

Use Go 1.26 or newer. Dependency versions are pinned in `go.mod` and `go.sum`.

```sh
git clone git@github.com:jintilm/GrantDiff.git
cd GrantDiff
go build -o bin/grantdiff ./cmd/grantdiff
go run ./cmd/grantdiff diff --before examples/retained/before --after examples/retained/after --identities examples/retained/identities.yaml --catalog examples/retained/catalog.yaml
```

On Windows, build the executable as `bin/grantdiff.exe`. Offline review runs with the executable and input files. Git mode also uses Git. Kubernetes and Docker are used by snapshot and integration workflows as described below.

## Commands

```sh
grantdiff diff --before before/ --after after/ --identities identities.yaml --catalog catalog.yaml --format json --output report.json
grantdiff diff --repo . --base BASE_COMMIT --head HEAD_COMMIT --path deploy/rbac --identities identities.yaml --catalog catalog.yaml --format markdown
grantdiff query --input examples/retained/after --identities examples/retained/identities.yaml --catalog examples/retained/catalog.yaml --identity report-bot --verb get --resource configmaps --name billing-config --namespace finance
grantdiff snapshot --context kind-grantdiff-dev --kubeconfig kubeconfig --output snapshot/
grantdiff diff --baseline snapshot/ --changes changes.yaml --identities identities.yaml --format json
grantdiff catalog --input examples/crd/after --output widgets-catalog.json
grantdiff diff --before examples/crd/before --after examples/crd/after --summary --format markdown
grantdiff diff --before examples/external-dns/before --after examples/external-dns/after --catalog examples/external-dns/catalog.yaml --compact --format markdown
grantdiff diff --before examples/kube-state-metrics/before --after examples/kube-state-metrics/after --catalog examples/kube-state-metrics/catalog.json --compact --fail-on removed
```

Run `grantdiff COMMAND --help` for the complete flags. Choose `text`, `markdown`, or `json`. A comparison budget defaults to one million request classes and is configurable with `--max-classes`. Inputs are limited to 10,000 files, 64 MiB total and 64 YAML nesting levels. Expand YAML aliases before review. Empty before or after states require `--allow-empty-before` or `--allow-empty-after`.

Git mode reads both complete committed trees. Unchanged bindings participate in authorization, and local working-tree edits are preserved. Review already-rendered manifests from Helm or Kustomize. Namespace defaults are explicit through `--default-namespace`.

## Interpreting results

The authorization model is `kubernetes-rbac`. Conclusions apply to the supplied manifests and identity memberships. Authentication, other authorizers, admission policies and actual object existence also affect live requests.

| Change | Meaning |
| --- | --- |
| `added` | Previously ungranted permission becomes granted |
| `removed` | Previously granted permission becomes ungranted within the supplied scope |
| `retained_via_other_path` | A granting path disappears and another existing path still grants access |
| `path_changed` | Access remains granted through a different set of paths |
| `indeterminate` | Missing policy or identity information prevents a decision |

JSON ranges describe request sets. `value` means exact equality, `except` means all values except the listed ones, and URL `prefix` / `exceptPrefixes` describe prefix regions. `representativeRequest` supplies one example within a range. A `range` carries the full change scope, including unnamed or future resources covered by wildcards. The default reporter emits individual equivalence classes, which can produce large reports for broad rules.

`--compact` joins compatible ranges with identical identities, change types, before/after decisions, risk findings and complete evidence, including source locations. Each merge fills an explicitly excluded value in one dimension while preserving every other dimension. URL prefix exclusions remain enforced. Counts describe the resulting ranges and IDs are derived from those ranges. `comparedClasses` records the original analysis work, and incomplete status, diagnostics and CI failure conditions remain applicable. Compaction uses the existing schema 1.0 selectors and can be combined with `--summary`.

Text and Markdown start with counts by change type and affected identity or group, followed by the bindings and roles behind new access. Markdown places each detailed range inside an expandable section. `--summary` shows the overview and diagnostics while applying the same CI failure conditions. Counts refer to request ranges, which may include many object names. JSON retains the complete schema 1.0 report.

A missing role, unknown explicit resource scope, incomplete user groups or exhausted budget produces an incomplete report. Independent positive evidence remains visible. Aggregation is evaluated at its stable state for acyclic dependencies, including snapshot-based queries. Cyclic aggregate dependencies produce diagnostics.

User groups come from the identity file. ServiceAccount system groups are derived automatically. Inferred ordinary users have incomplete membership until explicitly supplied. Group-only results use `group_grant`, while user and service account results use `identity_access`. Supply an identities file to focus the review on identities of interest.

Resource catalog examples are in `examples/retained/catalog.yaml`. Every resource requires an explicit `namespaced: true` or `namespaced: false`. A missing or null scope is rejected. Snapshots include a discovery-derived catalog. Custom resources can be added to an explicit catalog.

Manifests containing `apiextensions.k8s.io/v1` CRDs automatically contribute resource names, API groups, scope and served `status` / `scale` subresources. Run `grantdiff catalog --input crds/` to export this knowledge as JSON, optionally merging a base catalog with `--catalog base.yaml`. Diff and query merge explicit catalogs with CRD knowledge and reject conflicting scopes. CRD extraction validates catalog-relevant fields, and deployment validity is checked by Kubernetes. The catalog unions served-version subresources because RBAC permissions apply across API versions. See the [CRD example](examples/crd/README.md).

If the before and after inputs disagree on a resource's scope, the diff returns an incomplete result with `catalog_scope_changed` so the conflict can be resolved explicitly.

Manifest loading accepts generic Kubernetes `List` objects and typed RBAC lists such as `RoleList` and `RoleBindingList`. Items in a typed list inherit omitted kind and API version from the envelope. Conflicting item metadata is rejected with its source location.

| Exit code | Meaning |
| --- | --- |
| 0 | Review completed and configured conditions passed |
| 1 | `--fail-on added`, `--fail-on removed`, or `--fail-on high` matched |
| 2 | Invalid arguments, input or output failure |
| 3 | Incomplete review or unknown query |
| 4 | Snapshot API operation failed |

## Snapshots and candidate changes

Snapshot uses a user-selected trusted kubeconfig and context. Required reads include all namespaces, ServiceAccounts, four RBAC kinds and API discovery. The output contains policy and identity metadata, a resource catalog and capture metadata. Two observations detect changes during collection. A snapshot describes a capture interval across resource types.

New snapshot schema 1.1 records SHA-256 hashes of the complete `rbac.json` and `catalog.json` files, covering captured identities and resource scope as well as policy. Legacy schema 1.0 snapshots remain readable with their original RBAC-only integrity check. The compatibility document describes the trust and consistency boundaries.

```yaml
schemaVersion: "1.0"
baselineDigest: REPLACE_WITH_SNAPSHOT_DIGEST
changes:
  - action: delete
    key: RoleBinding/finance/direct-read
```

`upsert` operations supply a complete `object` with a matching key. An optional `beforeDigest` guards an individual object. The baseline digest is required. Every unmentioned baseline object continues to participate in review. A candidate deleting all RBAC objects requires `--allow-empty-after`.

## Development and verification

```sh
go test ./...
go vet ./...
go run ./scripts/integration --kind kind --result .work/verification.json
go run ./scripts/benchmark --mode files
go run ./scripts/benchmark --mode files --scenario wildcard --compact
go run ./scripts/release --version 1.0.0-rc.1
python -m pip install -r scripts/packagecheck/requirements.txt
python scripts/packagecheck/verify.py --version 1.0.0-rc.1 --result .work/package-native.json
```

The integration runner creates a uniquely named kind cluster, runs real authorization comparisons, and deletes its own cluster. A cached or mirrored image with the pinned digest can be supplied with `--image`. It uses kind v0.33.0 and defaults to Kubernetes v1.37.0.

Generated scenarios use eight fixed PCG seeds to vary roles, bindings and group membership. They compare before/after decisions with SubjectAccessReview, check that the diff covers each changed request exactly once, and verify compacted coverage, risks and evidence for every sampled request. Set `GRANTDIFF_TEST_SEED` to replay one seed. A failure writes input manifests, identities, catalog, request and report under `.work/failures/`. CI uploads these files and the verification JSON results for each Kubernetes version.

The benchmark offers `sparse` and `wildcard` scenarios. File mode records loading, comparison, compaction, output time and report size separately. `--max-classes` exercises the budget limit, with the resulting calculation status recorded in the output.

The release command creates Windows x86-64 ZIP and Linux x86-64 tar.gz packages and SHA-256 checksums under `dist/`. CI runs native unit tests on Windows and Ubuntu, API oracle tests on three Kubernetes versions, extracted-package checks on both platforms and output fingerprint parity. The tagged release workflow publishes the archives after those jobs succeed. Actual verification records and current limits are maintained in `docs/validation.md`.

## Contributing

Add an input that reproduces the incorrect permission conclusion, an expected result and, where applicable, a Kubernetes API comparison. See [CONTRIBUTING.md](CONTRIBUTING.md). The project is licensed under Apache-2.0. Dependency notices are tracked in [NOTICE](NOTICE).
