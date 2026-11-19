# CRD catalog and review

The fixture defines namespaced `widgets.example.test` with a served `v1` version and a `status` subresource. The candidate adds `patch` permission for that subresource.

Run from the repository root. The diff and query commands infer the resource catalog from the CRD present in each input state.

```text
go run ./cmd/grantdiff diff --before examples/crd/before --after examples/crd/after --format markdown
go run ./cmd/grantdiff catalog --input examples/crd/after --output widgets-catalog.json
go run ./cmd/grantdiff query --input examples/crd/after --identity system:serviceaccount:apps:widget-bot --api-group example.test --resource widgets --subresource status --verb patch --namespace apps --name demo
```

The report must contain one added request range for `widgets/status`. Catalog generation can merge a core-resource base file through `--catalog`. Conflicting scopes produce an input error. The generated catalog unions `status` and `scale` subresources across served versions because Kubernetes RBAC rules match API groups and resources across API versions.
