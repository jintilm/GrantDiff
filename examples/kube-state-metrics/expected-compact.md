**GrantDiff — complete**

Reviewed 1 identities · 1 affected identities or groups · 281 compared request classes

Added 0 · Removed 2 · Retained via other path 0 · Path changed 24 · Indeterminate 0 · High-risk ranges 0

Counts describe changed request ranges, including ranges that cover many object names.

Scope supplied-manifests · Policies declared\_complete · Identity groups complete

| Identity or group | Result type | Added | Removed | Retained | Path changed | Unknown | High-risk |
| --- | --- | --- | --- | --- | --- | --- | --- |
| system:serviceaccount:kube-system:kube-state-metrics | identity_access | 0 | 2 | 0 | 24 | 0 | 0 |


<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · 00da061db4c422253a32</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace=all resource="endpoints" subresource="" verb="list"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · 034f6438895e7628b925</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace=all resource="services" subresource="" verb="list"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · 0508dbe0477b97be7ef4</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace="" resource="namespaces" subresource="" verb="watch"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · 0e0be142e1c4d556e171</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace=all resource="serviceaccounts" subresource="" verb="watch"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · 162a8ee6f02e347acf79</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace=all resource="pods" subresource="" verb="watch"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · 1e23a55934308e05dbb9</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace=all resource="persistentvolumeclaims" subresource="" verb="watch"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · 2674deb0d158dd683f7f</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace=all resource="resourcequotas" subresource="" verb="list"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>removed · system:serviceaccount:kube-system:kube-state-metrics · 4776f8fb0b34d98ed7bf</summary>

    removed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → not_granted
      apiGroup="" name=all namespace=all resource="secrets" subresource="" verb="watch"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · 5b6aeda9bd8c0b468f9a</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace=all resource="limitranges" subresource="" verb="watch"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · 6514b7c8d1d0a8673cd6</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace=all resource="replicationcontrollers" subresource="" verb="watch"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · 655b947647cc93ae3b4f</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace=all resource="serviceaccounts" subresource="" verb="list"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · 65bcf3660c7e544c75c3</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace="" resource="nodes" subresource="" verb="watch"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · 69523d5a32e279e999fc</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace=all resource="persistentvolumeclaims" subresource="" verb="list"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · 76d30cf843585f2b2869</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace="" resource="persistentvolumes" subresource="" verb="watch"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · 8e1ae4e86b9133edc118</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace=all resource="pods" subresource="" verb="list"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · a3aace720a868533d435</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace="" resource="nodes" subresource="" verb="list"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · b8f6c9790328a9c7adfe</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace=all resource="limitranges" subresource="" verb="list"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · b91e5b1f9fea86ce10a9</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace="" resource="namespaces" subresource="" verb="list"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · c8cb554cb8453f08413e</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace=all resource="replicationcontrollers" subresource="" verb="list"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · d9ac40db02cebc48760e</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace=all resource="resourcequotas" subresource="" verb="watch"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>removed · system:serviceaccount:kube-system:kube-state-metrics · df546fd01da16985c120</summary>

    removed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → not_granted
      apiGroup="" name=all namespace=all resource="secrets" subresource="" verb="list"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · dfc00bc571d154d87bec</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace=all resource="configmaps" subresource="" verb="list"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · e40aa603c18803dff102</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace=all resource="configmaps" subresource="" verb="watch"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · e74a80a20a7f4809c7fc</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace=all resource="endpoints" subresource="" verb="watch"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · f7a356261777e96e05f9</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace=all resource="services" subresource="" verb="watch"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>

<details>
<summary>path_changed · system:serviceaccount:kube-system:kube-state-metrics · fa66b491578a5510f64d</summary>

    path_changed  system:serviceaccount:kube-system:kube-state-metrics  identity_access
      granted → granted
      apiGroup="" name=all namespace="" resource="persistentvolumes" subresource="" verb="list"
      Before  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)
      After  ServiceAccount/kube-system/kube-state-metrics → ClusterRoleBinding//kube-state-metrics → ClusterRole//kube-state-metrics (cluster-role.yaml:10)


</details>
