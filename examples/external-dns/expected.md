**GrantDiff — complete**

Reviewed 1 identities · 1 affected identities or groups · 59 compared request classes

Added 9 · Removed 0 · Retained via other path 0 · Path changed 0 · Indeterminate 0 · High-risk ranges 9

Counts describe changed request ranges, including ranges that cover many object names.

Scope supplied-manifests · Policies declared\_complete · Identity groups complete

| Identity or group | Result type | Added | Removed | Retained | Path changed | Unknown | High-risk |
| --- | --- | --- | --- | --- | --- | --- | --- |
| system:serviceaccount:default:external-dns | identity_access | 9 | 0 | 0 | 0 | 0 | 9 |

    Added access for system:serviceaccount:default:external-dns through ClusterRoleBinding//external-dns-viewer → ClusterRole//external-dns

<details>
<summary>added · system:serviceaccount:default:external-dns · 089d0d8b697c97591db2</summary>

    added  system:serviceaccount:default:external-dns  identity_access
      not_granted → granted
      apiGroup="" name="" namespace="" resource="secrets" subresource="" verb="list"
      After  ServiceAccount/default/external-dns → ClusterRoleBinding//external-dns-viewer → ClusterRole//external-dns (external-dns-clusterrole.yaml:18)
      high secret_read — new access to Secret data in the reported scope


</details>

<details>
<summary>added · system:serviceaccount:default:external-dns · 19750b920a4356afef99</summary>

    added  system:serviceaccount:default:external-dns  identity_access
      not_granted → granted
      apiGroup="" name="" namespace="" resource="secrets" subresource="" verb="watch"
      After  ServiceAccount/default/external-dns → ClusterRoleBinding//external-dns-viewer → ClusterRole//external-dns (external-dns-clusterrole.yaml:18)
      high secret_read — new access to Secret data in the reported scope


</details>

<details>
<summary>added · system:serviceaccount:default:external-dns · 29d07da345136607faf0</summary>

    added  system:serviceaccount:default:external-dns  identity_access
      not_granted → granted
      apiGroup="" name="" namespace=all except [""] resource="secrets" subresource="" verb="list"
      After  ServiceAccount/default/external-dns → ClusterRoleBinding//external-dns-viewer → ClusterRole//external-dns (external-dns-clusterrole.yaml:18)
      high secret_read — new access to Secret data in the reported scope


</details>

<details>
<summary>added · system:serviceaccount:default:external-dns · 3143a875a873424ba2e9</summary>

    added  system:serviceaccount:default:external-dns  identity_access
      not_granted → granted
      apiGroup="" name=all except [""] namespace=all except [""] resource="secrets" subresource="" verb="get"
      After  ServiceAccount/default/external-dns → ClusterRoleBinding//external-dns-viewer → ClusterRole//external-dns (external-dns-clusterrole.yaml:18)
      high secret_read — new access to Secret data in the reported scope


</details>

<details>
<summary>added · system:serviceaccount:default:external-dns · 705c688c73628651e7b5</summary>

    added  system:serviceaccount:default:external-dns  identity_access
      not_granted → granted
      apiGroup="" name="" namespace=all except [""] resource="secrets" subresource="" verb="watch"
      After  ServiceAccount/default/external-dns → ClusterRoleBinding//external-dns-viewer → ClusterRole//external-dns (external-dns-clusterrole.yaml:18)
      high secret_read — new access to Secret data in the reported scope


</details>

<details>
<summary>added · system:serviceaccount:default:external-dns · 9bce485921e80ea09890</summary>

    added  system:serviceaccount:default:external-dns  identity_access
      not_granted → granted
      apiGroup="" name=all except [""] namespace=all except [""] resource="secrets" subresource="" verb="list"
      After  ServiceAccount/default/external-dns → ClusterRoleBinding//external-dns-viewer → ClusterRole//external-dns (external-dns-clusterrole.yaml:18)
      high secret_read — new access to Secret data in the reported scope


</details>

<details>
<summary>added · system:serviceaccount:default:external-dns · 9eb0a60f7b9f7e4c0116</summary>

    added  system:serviceaccount:default:external-dns  identity_access
      not_granted → granted
      apiGroup="" name=all except [""] namespace="" resource="secrets" subresource="" verb="watch"
      After  ServiceAccount/default/external-dns → ClusterRoleBinding//external-dns-viewer → ClusterRole//external-dns (external-dns-clusterrole.yaml:18)
      high secret_read — new access to Secret data in the reported scope


</details>

<details>
<summary>added · system:serviceaccount:default:external-dns · b37e325cc07295cf8a0d</summary>

    added  system:serviceaccount:default:external-dns  identity_access
      not_granted → granted
      apiGroup="" name=all except [""] namespace=all except [""] resource="secrets" subresource="" verb="watch"
      After  ServiceAccount/default/external-dns → ClusterRoleBinding//external-dns-viewer → ClusterRole//external-dns (external-dns-clusterrole.yaml:18)
      high secret_read — new access to Secret data in the reported scope


</details>

<details>
<summary>added · system:serviceaccount:default:external-dns · d5beddba4071b800f0ac</summary>

    added  system:serviceaccount:default:external-dns  identity_access
      not_granted → granted
      apiGroup="" name=all except [""] namespace="" resource="secrets" subresource="" verb="list"
      After  ServiceAccount/default/external-dns → ClusterRoleBinding//external-dns-viewer → ClusterRole//external-dns (external-dns-clusterrole.yaml:18)
      high secret_read — new access to Secret data in the reported scope


</details>
