# kube-state-metrics permission narrowing

The fixture pins the standard ClusterRole, ClusterRoleBinding and ServiceAccount from kube-state-metrics v2.14.0, commit `e763ea2aa37e7fa507775a8506387aaa95ff17d1`. The copied files and Apache-2.0 license have SHA-256 hashes in `provenance.json`.

The locally authored candidate removes `secrets` from the first ClusterRole rule. This hypothetical change revokes Secret `list` and `watch` for `system:serviceaccount:kube-system:kube-state-metrics`. The catalog supplies scopes for the 37 resources used by this configuration. Its Kubernetes authorization behavior can be reviewed independently of whether this historical application version is deployed.

Run from the repository root or the extracted release directory with the installed `grantdiff` executable.

```text
grantdiff diff --before examples/kube-state-metrics/before --after examples/kube-state-metrics/after --catalog examples/kube-state-metrics/catalog.json --compact --format markdown --fail-on removed
```

The compact report contains two removed ranges and 24 path changes. The path changes preserve granted access and record the changed rule identifier for the other resources in that same rule. The process exits with code 1 because Secret access was removed. `expected-compact.md` contains the complete expected output. Package verification and CLI regressions check the subject, resource, verbs, remaining grants and evidence.

[Pinned upstream configuration](https://github.com/kubernetes/kube-state-metrics/tree/e763ea2aa37e7fa507775a8506387aaa95ff17d1/examples/standard)
