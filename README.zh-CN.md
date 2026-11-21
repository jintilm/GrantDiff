# GrantDiff

在 Windows 和 Linux 上评审 Kubernetes RBAC 变更，查看权限增加、权限撤销和权限仍然保留的具体原因。

项目提供目录比较、Git 提交比较、单项权限查询、CRD 资源目录生成、集群快照和基于快照的候选变更。输出支持文本、Markdown 和 JSON。当前版本为交互查看器预览版 `1.1.0-preview.1`。

本版在候选版的 CLI 基础上增加离线交互报告查看器，提供中英文切换、范围筛选、前后结论和授权路径展开。支持约定和版本边界已写入兼容性文档。[兼容性约定](docs/compatibility.md) · [查看器说明](docs/viewer.md)

## 运行演示

开发环境使用 Go 1.26 或更新版本。

源码仓库为 [jintilm/GrantDiff](https://github.com/jintilm/GrantDiff)。克隆后进入项目目录。

```text
git clone git@github.com:jintilm/GrantDiff.git
cd GrantDiff
```

发行包包含程序、样例和文档。解压后，Linux 使用 `./grantdiff`，Windows 使用 `./grantdiff.exe`，后接下面命令中的 `diff` 参数即可运行样例。

下载页提供 Windows/Linux x86-64 软件包和 `SHA256SUMS`。Windows 可用 `Get-FileHash -Algorithm SHA256`，Linux 可用 `sha256sum` 核对下载文件。[版本下载](https://github.com/jintilm/GrantDiff/releases)

```text
go run ./cmd/grantdiff diff --before examples/retained/before --after examples/retained/after --identities examples/retained/identities.yaml --catalog examples/retained/catalog.yaml
```

演示移除了 `report-bot` 的直接绑定。账号仍通过 `system:serviceaccounts:finance` 组获得读取权限，报告会显示 `retained_via_other_path`，并列出剩余路径。

生成 JSON 后可以直接制作离线交互报告。

```text
go run ./cmd/grantdiff diff --before examples/retained/before --after examples/retained/after --identities examples/retained/identities.yaml --catalog examples/retained/catalog.yaml --format json --output report.json
go run ./cmd/grantdiff view --input report.json --output review.html
```

用浏览器打开 `review.html`，可以按身份、变更类型、风险和命名空间筛选，再查看前后授权路径及文件位置。安装包内附带可以直接打开的 `viewer.html`，也可用 `view --output viewer.html` 生成空查看器。页面支持离线运行，嵌入报告的 HTML 包含原报告的全部详情。JSON 上限为 128 MiB，报告不完整时会持续显示提示，默认语言跟随浏览器并可手动切换。

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

JSON 中 `representativeRequest` 是范围内的一个示例请求，完整范围由 `range` 表达。`value` 表示精确值，`except` 表示排除指定值后的剩余范围，URL 使用 `prefix` 和 `exceptPrefixes`。默认逐项输出请求等价类，通配范围很大时报告可能较长。

`--compact` 合并身份、变化类型、前后结论、风险及完整证据都一致的兼容范围，证据比较包括文件位置。每次合并在一个维度中补回一个被排除的精确值，其他维度及 URL 前缀限制继续约束结果。摘要数量和变化 ID 对应合并后的范围，`comparedClasses` 记录实际比较的等价类数量。该模式沿用 JSON Schema 1.0，支持与 `--summary` 一起使用，未完成诊断和 CI 退出规则继续生效。

文本和 Markdown 报告先汇总新增、移除、保留、路径变化、不确定范围及高风险范围，再按身份列出数量。Markdown 中的详细授权证据可以展开。`--summary` 输出摘要和诊断，CI 的失败条件仍按完整分析结果判断。摘要数量表示请求范围，一个范围可能覆盖多个对象名称。

输入中包含 v1 CRD 时，程序自动提取 API group、资源名称、作用域和已服务版本的 status、scale 子资源。也可以生成独立目录，并与已有目录合并，作用域冲突会报错。

```text
go run ./cmd/grantdiff catalog --input examples/crd/after
go run ./cmd/grantdiff diff --before examples/crd/before --after examples/crd/after --summary --format markdown
go run ./cmd/grantdiff diff --before examples/external-dns/before --after examples/external-dns/after --catalog examples/external-dns/catalog.yaml --summary --fail-on high
go run ./cmd/grantdiff diff --before examples/external-dns/before --after examples/external-dns/after --catalog examples/external-dns/catalog.yaml --compact --format markdown
go run ./cmd/grantdiff diff --before examples/kube-state-metrics/before --after examples/kube-state-metrics/after --catalog examples/kube-state-metrics/catalog.json --compact --fail-on removed
```

ExternalDNS 案例固定了上游版本和文件哈希，候选清单中的 Secret 读取规则是本项目编写的假设变更。预期报告与来源均随仓库保存。[ExternalDNS 案例](examples/external-dns/README.md) · [CRD 案例](examples/crd/README.md)

kube-state-metrics 案例固定 v2.14.0 的上游权限配置，演示撤销 Secret 的 list/watch 授权。其余受同一规则保护的资源仍被授予权限，报告会记录规则证据的变化。[kube-state-metrics 案例](examples/kube-state-metrics/README.md)

新快照采用 Schema 1.1，校验完整的 RBAC/身份清单和资源目录文件。旧版 Schema 1.0 继续按原有 RBAC 摘要规则读取，其身份和目录依赖输入来源的可信性。候选变更删除全部 RBAC 对象时，需要显式使用 `--allow-empty-after`。

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
go run ./scripts/benchmark --mode files --scenario wildcard --compact
go run ./scripts/release --version 1.1.0-preview.1
python -m pip install -r scripts/packagecheck/requirements.txt
python scripts/packagecheck/verify.py --version 1.1.0-preview.1 --result .work/package-native.json
```

集成测试程序创建自己的 kind 集群，完成授权对照后清理该集群。可以用 `--image` 选择固定摘要的节点镜像。发布程序在 `dist/` 下生成 Windows 和 Linux 软件包及校验和。

CI 会构建同一组安装包，在 Windows 和 Ubuntu 分别解压运行，再比较输出指纹。自动检查覆盖四组案例、JSON Schema、退出码、中文和空格路径、BOM/CRLF、只读输入及文档链接。带版本标签的发布流程在这些检查通过后上传安装包。

浏览器交互检查使用解压后的可执行文件。安装 `scripts/packagecheck/browser-requirements.txt` 并执行 `python -m playwright install --with-deps chromium` 后，可在安装包验证命令中加入 `--browser`。CI 在两个平台验证离线交互，并保留桌面和窄屏截图。

组合测试通过八个固定种子生成角色、绑定和组成员关系，分别核对权限判定、差异覆盖，以及合并前后每个采样请求的变化类型、风险和证据。设置环境变量 `GRANTDIFF_TEST_SEED` 可重跑一个种子。失败时，完整清单、身份、请求和报告保存到 `.work/failures/`，CI 会上传这些文件。

性能程序提供 `sparse` 与 `wildcard` 两个固定场景，文件模式分别记录加载、比较、范围合并和输出耗时，以及 JSON 体积。可用 `--max-classes` 测量预算耗尽行为。

本地输入限制为 10,000 个文件、64 MiB 总大小和 64 层 YAML 嵌套，YAML 别名需要预先展开。请求比较默认上限为 1,000,000 类，通过 `--max-classes` 调整。首批对象或全部删除的比较分别使用 `--allow-empty-before` 和 `--allow-empty-after`。

完整命令、快照格式和解释见 [英文使用说明](README.md)。实际验证结果见 [验证记录](docs/validation.md)，设计目标见 [中文设计文档](docs/design.zh-CN.md)。
