# Candidate support and compatibility

GrantDiff `1.0.0-rc.1` is a release candidate for the following v1.0 scope. Findings from candidate testing can require corrections before the stable release. The acceptance record is maintained in `validation.md`.

## Runtime and inputs

| Area | Candidate support |
| --- | --- |
| Executables | Windows x86-64 and Linux x86-64, built with `CGO_ENABLED=0` |
| Runtime verification | Native GitHub Windows runner and Ubuntu 24.04, local Windows and Linux containers |
| Kubernetes verification | Native RBAC compared with v1.35.8, v1.36.4 and v1.37.0 API servers |
| Policies | Rendered YAML or JSON with v1 Role, ClusterRole, RoleBinding and ClusterRoleBinding |
| Sources | Files, directories, complete Git trees, cluster snapshots and explicit candidate operations |
| Identities | User, Group, ServiceAccount and explicit user group memberships |
| Resource knowledge | Explicit catalogs, API discovery and v1 CRD metadata |
| Outputs | Text, Markdown and diff JSON Schema 1.0, with optional compact ranges |

Offline commands require the executable and input files. Git comparison requires Git. Snapshot capture requires the named Kubernetes context, a trusted kubeconfig and access to list the needed resources. Integration development requires Docker and kind. Packaging verification requires Python 3.12 or newer and the pinned packages in `scripts/packagecheck/requirements.txt`.

Other operating systems, architectures, Kubernetes versions and external authorizers can be evaluated through additional test environments. The supported matrix expands when those environments have recorded runtime checks.

## Authorization contract

Each report describes Kubernetes RBAC authorization within its supplied inputs. `granted` requires a known matching path. `not_granted` requires complete relevant policy and identity information with no matching path. Missing references, incomplete user groups and uncertain aggregation can produce `unknown`. Diff reports carry source, identity and calculation completeness independently.

RoleBindings apply within their namespace. ClusterRoleBindings apply to the resource scopes in the catalog. Wildcards retain symbolic coverage for future values. Concrete resource scope must be supplied when needed for a definitive request. Subresource requests carry the parent object's name and required namespace.

ServiceAccount system groups are derived automatically. Ordinary user group membership is supplied by the caller. Group results describe grants to that group. Acyclic aggregate roles use controller stable-state semantics in both queries and comparisons. Cycles produce incomplete results. Default Kubernetes roles participate when present in the input, such as a complete cluster snapshot.

CRD extraction validates catalog-relevant fields. Subresources are unioned across served versions because RBAC rules match API groups and resources across API versions. Kubernetes performs the full deployment validation. Admission controllers, other authorizers, identity-provider membership and final application behavior require their respective runtime data.

Risk findings cover Secret reads, RBAC bind/escalate and impersonation permissions. They identify new authorization ranges for review. Whether a multi-step privilege escalation can execute depends on additional permissions and runtime conditions.

## Formats and exit codes

Diff JSON follows `report.schema.json`. Consumers should use `range` as the complete request set and `representativeRequest` as one member. Compact mode preserves the union of requests, decisions, risks and evidence, including source locations. Change IDs derive from subject, target type, change type and final range. Compaction and correctness fixes can change IDs and range counts. Text and Markdown are intended for people and may gain formatting improvements.

| Code | Meaning |
| --- | --- |
| 0 | Completed and satisfied the selected CI condition, or a definite query result |
| 1 | Selected added, removed or high-risk condition was triggered |
| 2 | Invalid command, input or report output |
| 3 | Incomplete diff or unknown query |
| 4 | Snapshot connection, collection or persistence failed |

Incomplete analysis takes precedence over risk exit code 1. A query with `not_granted` returns code 0 and records the authorization result in its output. `--summary` and `--compact` use the same failure rules. An empty baseline requires `--allow-empty-before`. Removing all policies requires `--allow-empty-after`, including snapshot candidates.

Catalogs, identities and candidate operations retain schema 1.0. New persisted snapshots use schema 1.1 and SHA-256 digests for the complete `rbac.json` and `catalog.json` files. This detects changes to RBAC, captured identities, namespaces and resource knowledge. Metadata retains the RBAC object digest used by schema 1.0 candidate operations. Snapshot checksums provide integrity checks relative to their metadata, and the directory must come from a trusted source.

Legacy snapshot schema 1.0 remains readable with its original RBAC-only integrity check. Its catalog and identity metadata rely on the caller's trusted input. Snapshot collection compares two observations of policy and identity objects and records list resource versions. Discovery is collected separately. A snapshot describes an observation interval, so callers should capture a new baseline when cluster state changes.

The candidate proposes compatibility for documented commands and schema 1.0 inputs through v1.x. A future incompatible structured format receives a new schema version and a documented migration. Correctness fixes can revise conclusions for affected inputs, with regression cases and release notes identifying the change.

## Release acceptance

The automated release workflow runs unit tests and vet on both supported operating systems, the three-version Kubernetes matrix, native tests of both extracted archives and a comparison of platform output fingerprints. It verifies checksums, version strings, schema 1.0 reports, four scenarios, exit codes, Unicode paths, CRLF/BOM, read-only input and packaged documentation.

The two pinned controller fixtures cover an added permission in ExternalDNS and a removed permission in kube-state-metrics. Their source hashes, licenses, hypothetical candidate changes and expected reports are included. Candidate testing by outside users is tracked separately from these automated checks.

Before declaring v1.0 stable, review the latest candidate results and resolve known incorrect authorization conclusions, misleading complete reports and supported-platform installation failures. Record remaining documented boundaries in the stable notes. The RC milestone is complete when its promised checks and packages are reproducible.
