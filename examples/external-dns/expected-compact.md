**GrantDiff — complete**

Reviewed 1 identities · 1 affected identities or groups · 59 compared request classes

Added 3 · Removed 0 · Retained via other path 0 · Path changed 0 · Indeterminate 0 · High-risk ranges 3

Counts describe changed request ranges, including ranges that cover many object names.

Scope supplied-manifests · Policies declared\_complete · Identity groups complete

| Identity or group | Result type | Added | Removed | Retained | Path changed | Unknown | High-risk |
| --- | --- | --- | --- | --- | --- | --- | --- |
| system:serviceaccount:default:external-dns | identity_access | 3 | 0 | 0 | 0 | 0 | 3 |

    Added access for system:serviceaccount:default:external-dns through ClusterRoleBinding//external-dns-viewer → ClusterRole//external-dns

<details>
<summary>added · system:serviceaccount:default:external-dns · 3143a875a873424ba2e9</summary>

    added  system:serviceaccount:default:external-dns  identity_access
      not_granted → granted
      apiGroup="" name=all except [""] namespace=all except [""] resource="secrets" subresource="" verb="get"
      After  ServiceAccount/default/external-dns → ClusterRoleBinding//external-dns-viewer → ClusterRole//external-dns (external-dns-clusterrole.yaml:18)
      high secret_read — new access to Secret data in the reported scope


</details>

<details>
<summary>added · system:serviceaccount:default:external-dns · f81b87bcc921c777f3f0</summary>

    added  system:serviceaccount:default:external-dns  identity_access
      not_granted → granted
      apiGroup="" name=all namespace=all resource="secrets" subresource="" verb="watch"
      After  ServiceAccount/default/external-dns → ClusterRoleBinding//external-dns-viewer → ClusterRole//external-dns (external-dns-clusterrole.yaml:18)
      high secret_read — new access to Secret data in the reported scope


</details>

<details>
<summary>added · system:serviceaccount:default:external-dns · fc75f549d1175372e31b</summary>

    added  system:serviceaccount:default:external-dns  identity_access
      not_granted → granted
      apiGroup="" name=all namespace=all resource="secrets" subresource="" verb="list"
      After  ServiceAccount/default/external-dns → ClusterRoleBinding//external-dns-viewer → ClusterRole//external-dns (external-dns-clusterrole.yaml:18)
      high secret_read — new access to Secret data in the reported scope


</details>
