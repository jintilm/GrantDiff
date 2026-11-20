# ExternalDNS permission review

This fixture uses the ClusterRole and ClusterRoleBinding from ExternalDNS v0.15.1, pinned to commit `c55b1a1127df024b95f927869a1bcd662ec4337a`. The `before/` files reproduce the upstream bytes. `provenance.json` records their source URLs and SHA-256 hashes. The upstream Apache-2.0 license is included as `UPSTREAM-LICENSE.md`.

The `after/` directory adds a locally authored rule granting Secret `get`, `list` and `watch` to the controller's existing ClusterRole. This hypothetical change demonstrates review behavior. The catalog explicitly records the resource scopes used by the pinned example, including its legacy `extensions/ingresses` rule.

Run from the GrantDiff repository root.

```text
go run ./cmd/grantdiff diff --before examples/external-dns/before --after examples/external-dns/after --catalog examples/external-dns/catalog.yaml --format markdown
go run ./cmd/grantdiff diff --before examples/external-dns/before --after examples/external-dns/after --catalog examples/external-dns/catalog.yaml --summary --fail-on high
go run ./cmd/grantdiff diff --before examples/external-dns/before --after examples/external-dns/after --catalog examples/external-dns/catalog.yaml --compact --format markdown
```

The result must identify added Secret reads for `system:serviceaccount:default:external-dns`. Each added range must cite `ClusterRoleBinding//external-dns-viewer` and `ClusterRole//external-dns`. `--fail-on high` makes the executable return exit code 1. `go run` wraps a nonzero executable exit in its own error message.

The human-readable expected report is stored in `expected.md`. Tests compare the generated Markdown and verify the change types, target identity, resource, verbs and risk rule. They also verify the copied upstream file hashes.

The compact report in `expected-compact.md` contains three high-risk ranges, one per Secret read verb. The default report has nine ranges. Compaction joins compatible name and namespace ranges with identical evidence. Both reports cover the same requests and trigger `--fail-on high`.

[Upstream role](https://github.com/kubernetes-sigs/external-dns/blob/c55b1a1127df024b95f927869a1bcd662ec4337a/kustomize/external-dns-clusterrole.yaml) · [Upstream binding](https://github.com/kubernetes-sigs/external-dns/blob/c55b1a1127df024b95f927869a1bcd662ec4337a/kustomize/external-dns-clusterrolebinding.yaml)
