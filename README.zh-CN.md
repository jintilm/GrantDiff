# GrantDiff

在 Windows 和 Linux 上评审 Kubernetes RBAC 变更，查看权限增加、权限撤销和权限仍然保留的具体原因。

项目提供目录比较、Git 提交比较、单项权限查询、CRD 资源目录生成、集群快照和基于快照的候选变更。输出支持文本、Markdown 和 JSON。当前版本为 `0.5.0-preview`。

## 运行演示

开发环境使用 Go 1.26 或更新版本。

源码仓库为 [jintilm/GrantDiff](https://github.com/jintilm/GrantDiff)。克隆后进入项目目录。

```text
git clone git@github.com:jintilm/GrantDiff.git
cd GrantDiff
```

发行包包含程序、样例和文档。解压后，Linux 使用 `./grantdiff`，Windows 使用 `./grantdiff.exe`，后接下面命令中的 `diff` 参数即可运行样例。

```text
go run ./cmd/grantdiff diff --before examples/retained/before --after examples/retained/after --identities examples/retained/identities.yaml --catalog examples/retained/catalog.yaml
```

演示移除了 `report-bot` 的直接绑定。账号仍通过 `system:serviceaccounts:finance` 组获得读取权限，报告会显示 `retained_via_other_path`，并列出剩余路径。

Windows 构建命令如下。

```text
go build -o bin/grantdiff.exe ./cmd/grantdiff
```

Linux 构建命令如下。

```text
go build -o bin/grantdiff ./cmd/grantdiff
```

离线评审依赖程序和输入文件。Git 模式需要 Git。采集快照需要可访问的 Kubernetes API，集群对照测试需要 Docker 和 kind。

## 输入与结论

提供变更前后的完整 RBAC 清单，以及身份和资源目录。Git 模式读取两个提交中目标目录的完整内容，未修改文件中的授权也会参与计算。

普通用户的组成员关系由 identities 文件提供。ServiceAccount 的系统组由工具推导。资源目录描述 API group、资源规范名称和命名空间作用域，样例位于 `examples/retained/catalog.yaml`。自定义资源可加入目录文件。

资源目录中的每个资源必须显式填写 `namespaced: true` 或 `namespaced: false`，缺失或 null 会返回输入错误。程序支持 `RoleList`、`RoleBindingList` 等类型化列表，条目省略 kind 和 apiVersion 时从外层列表补全，冲突的类型信息会报错。

报告说明给定输入下的原生 RBAC 授权。缺失角色、未知资源作用域、用户组信息不完整或比较预算耗尽时，报告标记不完整。授权路径中能够确定的证据继续展示。

JSON 中 `representativeRequest` 是范围内的一个示例请求，完整范围由 `range` 表达。`value` 表示精确值，`except` 表示排除指定值后的剩余范围，URL 使用 `prefix` 和 `exceptPrefixes`。当前版本逐项输出请求等价类，通配范围很大时报告可能较长。

文本和 Markdown 报告先汇总新增、移除、保留、路径变化、不确定范围及高风险范围，再按身份列出数量。Markdown 中的详细授权证据可以展开。`--summary` 输出摘要和诊断，CI 的失败条件仍按完整分析结果判断。摘要数量表示请求范围，一个范围可能覆盖多个对象名称。

输入中包含 v1 CRD 时，程序自动提取 API group、资源名称、作用域和已服务版本的 status、scale 子资源。也可以生成独立目录，并与已有目录合并，作用域冲突会报错。

```text
go run ./cmd/grantdiff catalog --input examples/crd/after
go run ./cmd/grantdiff diff --before examples/crd/before --after examples/crd/after --summary --format markdown
go run ./cmd/grantdiff diff --before examples/external-dns/before --after examples/external-dns/after --catalog examples/external-dns/catalog.yaml --summary --fail-on high
```

ExternalDNS 案例固定了上游版本和文件哈希，候选清单中的 Secret 读取规则是本项目编写的假设变更。预期报告与来源均随仓库保存。[ExternalDNS 案例](examples/external-dns/README.md) · [CRD 案例](examples/crd/README.md)

| 退出码 | 含义 |
| --- | --- |
| 0 | 评审完成，符合所选通过条件 |
| 1 | 触发 `--fail-on added`、`--fail-on removed` 或 `--fail-on high` |
| 2 | 参数、输入或输出错误 |
| 3 | 评审不完整或查询结论未知 |
| 4 | 集群快照操作失败 |

## 验证和打包

```text
go test ./...
go vet ./...
go run ./scripts/integration --kind kind --result .work/verification.json
go run ./scripts/benchmark --mode files
go run ./scripts/release --version 0.5.0-preview
```

集成测试程序创建自己的 kind 集群，完成授权对照后清理该集群。可以用 `--image` 选择固定摘要的节点镜像。发布程序在 `dist/` 下生成 Windows 和 Linux 软件包及校验和。

组合测试通过八个固定种子生成角色、绑定和组成员关系，分别核对权限判定和差异覆盖。设置环境变量 `GRANTDIFF_TEST_SEED` 可重跑一个种子。失败时，完整清单、身份、请求和报告保存到 `.work/failures/`，CI 会上传这些文件。

本地输入限制为 10,000 个文件、64 MiB 总大小和 64 层 YAML 嵌套，YAML 别名需要预先展开。请求比较默认上限为 1,000,000 类，通过 `--max-classes` 调整。首批对象或全部删除的比较分别使用 `--allow-empty-before` 和 `--allow-empty-after`。

完整命令、快照格式和解释见 [英文使用说明](README.md)。实际验证结果见 [验证记录](docs/validation.md)，设计目标见 [中文设计文档](docs/design.zh-CN.md)。
