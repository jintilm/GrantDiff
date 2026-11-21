"use strict";

(() => {
  const words = {
    en: {
      review: "Permission review",
      local: "Local report",
      open: "Open JSON report",
      welcomeTitle: "Follow every permission change.",
      welcomeText:
        "Open a GrantDiff JSON report to explore affected identities, request ranges and authorization paths.",
      privacy: "Files are read in this browser. The viewer works offline.",
      reportLabel: "RBAC CHANGE REPORT",
      title: "Permission changes",
      changes: "Changes",
      complete: "Analysis complete",
      incomplete: "Incomplete review",
      incompleteTitle: "This review is incomplete",
      incompleteText:
        "Some conclusions depend on missing information or an unfinished calculation. Review the input scope and diagnostics before making a decision.",
      scopeTitle: "Input scope and diagnostics",
      beforeSource: "Before input",
      afterSource: "After input",
      beforeDigest: "Before policy digest",
      afterDigest: "After policy digest",
      policies: "Policy completeness",
      identities: "Identity completeness",
      calculation: "Calculation",
      source: "Input scope",
      declared_complete: "Declared complete",
      identity: "Identity or group",
      allIdentities: "All identities and groups",
      risk: "Risk",
      allRisks: "All risks",
      highOnly: "High-risk ranges",
      high: "High risk",
      namespaceFilter: "Namespace in range",
      namespacePlaceholder: "e.g. finance",
      clusterScope: "Empty namespace requests",
      reset: "Reset filters",
      rangeNote:
        "Counts describe request ranges. Each range can cover many objects.",
      searchLabel: "Search changes",
      searchPlaceholder: "Search identity, resource, verb or path…",
      previous: "Previous",
      next: "Next",
      all: "All changes",
      added: "Added",
      removed: "Removed",
      retained_via_other_path: "Retained access",
      path_changed: "Path changed",
      indeterminate: "Indeterminate",
      before: "Before",
      after: "After",
      granted: "Granted",
      not_granted: "Not granted",
      unknown: "Unknown",
      range: "Full request range",
      paths: "Authorization paths",
      risks: "Risk findings",
      example: "Representative request",
      exampleNote: "One request inside this range",
      noEvidence: "No matching authorization path recorded.",
      subject: "Subject match",
      binding: "Binding",
      role: "Role",
      aggregation: "Aggregation",
      rule: "Rule ID",
      bindingSource: "Binding source",
      ruleSource: "Rule source",
      document: "document",
      line: "line",
      allValues: "All values",
      except: "excluding",
      prefix: "Starts with",
      exceptPrefixes: "excluding prefixes",
      emptyValue: '"" (empty)',
      core: '"" (core API)',
      apiGroup: "API group",
      verb: "Verb",
      resource: "Resource",
      subresource: "Subresource",
      namespace: "Namespace",
      name: "Object name",
      url: "URL",
      noMatches: "No matching changes",
      noMatchesText: "Adjust the filters to explore other ranges.",
      noChanges: "No permission changes",
      noChangesText: "The supplied report contains no changed request ranges.",
      unknownEmpty: "No changes recorded yet",
      unknownEmptyText:
        "The analysis is incomplete. Inspect the diagnostics for unresolved inputs or limits.",
      selectChange: "Select a change",
      selectText: "Explore its full request range and authorization evidence.",
      loaded: "Loaded",
      invalid: "Could not open this report",
      unsupported: "Select a GrantDiff schema 1.0 diff JSON report.",
      tooLarge: "The report exceeds the 128 MiB viewer limit.",
      reading: "Reading report…",
      footer: "Kubernetes RBAC · Evidence for the supplied inputs",
      noDiagnostics: "No input diagnostics recorded.",
      identity_access: "Identity access",
      group_grant: "Group grant",
      morePaths: "Show more paths",
      addedExplanation:
        "This range gains a matching RBAC authorization path in the candidate state.",
      removedExplanation:
        "The baseline granted this range. The candidate has no matching grant for it in the supplied inputs.",
      retainedExplanation:
        "An authorization path was removed, and another recorded path continues to grant this range.",
      pathExplanation:
        "Access remains granted. The rule or authorization evidence changed.",
      unknownExplanation:
        "The report cannot determine the change for this range. Inspect both the available paths and the diagnostics.",
      meta: (identities, classes) =>
        `${identities.toLocaleString()} identities reviewed · ${classes.toLocaleString()} request classes compared`,
      matches: (count, total) =>
        `${count.toLocaleString()} of ${total.toLocaleString()} ranges`,
      page: (page, total) => `${page} / ${total}`,
      pathCount: (count) => `${count} recorded paths`,
    },
    zh: {
      review: "权限变更评审",
      local: "本地报告",
      open: "打开 JSON 报告",
      welcomeTitle: "看清每一项权限变化。",
      welcomeText:
        "打开 GrantDiff JSON 报告，查看受影响的身份、请求范围和授权路径。",
      privacy: "文件在当前浏览器中读取，查看器支持离线使用。",
      reportLabel: "KUBERNETES 权限评审",
      title: "权限变更",
      changes: "变更类型",
      complete: "计算已完成",
      incomplete: "评审不完整",
      incompleteTitle: "这份评审尚不完整",
      incompleteText:
        "部分结论依赖缺失的信息或尚未完成的计算。请结合输入范围与诊断信息进行判断。",
      scopeTitle: "输入范围与诊断",
      beforeSource: "变更前输入",
      afterSource: "变更后输入",
      beforeDigest: "变更前策略摘要",
      afterDigest: "变更后策略摘要",
      policies: "策略完整性",
      identities: "身份完整性",
      calculation: "计算状态",
      source: "输入范围",
      declared_complete: "已声明完整",
      identity: "身份或组",
      allIdentities: "全部身份和组",
      risk: "风险",
      allRisks: "全部风险级别",
      highOnly: "高风险范围",
      high: "高风险",
      namespaceFilter: "范围内的命名空间",
      namespacePlaceholder: "例如 finance",
      clusterScope: "空命名空间请求",
      reset: "重置筛选",
      rangeNote: "数量表示请求范围，一个范围可以覆盖多个对象。",
      searchLabel: "搜索变更",
      searchPlaceholder: "搜索身份、资源、动词或路径…",
      previous: "上一页",
      next: "下一页",
      all: "全部变更",
      added: "新增权限",
      removed: "撤销权限",
      retained_via_other_path: "权限仍保留",
      path_changed: "路径变化",
      indeterminate: "无法确定",
      before: "变更前",
      after: "变更后",
      granted: "已授予",
      not_granted: "未授予",
      unknown: "未知",
      range: "完整请求范围",
      paths: "授权路径",
      risks: "风险提示",
      example: "代表请求",
      exampleNote: "该范围内的一个请求示例",
      noEvidence: "报告中没有记录匹配的授权路径。",
      subject: "匹配主体",
      binding: "权限绑定",
      role: "角色",
      aggregation: "聚合路径",
      rule: "规则 ID",
      bindingSource: "绑定来源",
      ruleSource: "规则来源",
      document: "文档",
      line: "行",
      allValues: "全部取值",
      except: "排除",
      prefix: "以前缀开头",
      exceptPrefixes: "排除前缀",
      emptyValue: '""（空值）',
      core: '""（核心 API）',
      apiGroup: "API 组",
      verb: "动词",
      resource: "资源",
      subresource: "子资源",
      namespace: "命名空间",
      name: "对象名",
      url: "URL",
      noMatches: "没有匹配的变更",
      noMatchesText: "调整筛选条件以查看其他范围。",
      noChanges: "没有权限变更",
      noChangesText: "当前报告中没有发生变化的请求范围。",
      unknownEmpty: "尚未记录变更",
      unknownEmptyText: "分析尚不完整，请查看诊断中的缺失信息或计算限制。",
      selectChange: "选择一项变更",
      selectText: "查看完整请求范围和授权证据。",
      loaded: "已加载",
      invalid: "无法打开这份报告",
      unsupported: "请选择 GrantDiff Schema 1.0 的差异 JSON 报告。",
      tooLarge: "报告超过查看器的 128 MiB 大小上限。",
      reading: "正在读取报告…",
      footer: "Kubernetes RBAC · 基于所提供输入的授权证据",
      noDiagnostics: "报告中没有输入诊断。",
      identity_access: "身份权限",
      group_grant: "组授权",
      morePaths: "显示更多路径",
      addedExplanation: "候选状态中增加了匹配此范围的 RBAC 授权路径。",
      removedExplanation:
        "基线授予此范围的权限。根据所提供的输入，候选状态中已没有匹配的授权。",
      retainedExplanation:
        "一条授权路径被移除，另一条已记录的路径仍然授予此范围的权限。",
      pathExplanation: "此范围仍被授予权限，对应的规则或授权证据发生了变化。",
      unknownExplanation:
        "报告无法确定此范围的变化，请结合现有路径与诊断信息判断。",
      meta: (identities, classes) =>
        `已评审 ${identities.toLocaleString()} 个身份 · 已比较 ${classes.toLocaleString()} 类请求`,
      matches: (count, total) =>
        `${count.toLocaleString()} / ${total.toLocaleString()} 个范围`,
      page: (page, total) => `${page} / ${total}`,
      pathCount: (count) => `${count} 条授权路径`,
    },
  };
  const types = [
    "all",
    "added",
    "removed",
    "retained_via_other_path",
    "path_changed",
    "indeterminate",
  ];
  const dimensions = [
    "verb",
    "apiGroup",
    "resource",
    "subresource",
    "namespace",
    "name",
    "url",
  ];
  const pageSize = 40;
  const maxBytes = 128 * 1024 * 1024;
  const schema = JSON.parse(
    document.getElementById("report-schema").textContent,
  );
  const $ = (id) => document.getElementById(id);
  let language = navigator.language.toLowerCase().startsWith("zh")
    ? "zh"
    : "en";
  let report = null;
  let label = "";
  let filtered = [];
  let searchIndex = [];
  let selected = -1;
  let page = 0;
  let activeType = "all";
  let loadSequence = 0;
  const t = (key) => Object.hasOwn(words[language], key) ? words[language][key] : key;

  function node(tag, text, className) {
    const element = document.createElement(tag);
    if (text !== undefined) element.textContent = text;
    if (className) element.className = className;
    return element;
  }

  // This validator implements the keywords used by the embedded report schema.
  // Unsupported keywords fail closed, so changes to the contract need review.
  const supported = new Set([
    "$schema",
    "$defs",
    "$ref",
    "title",
    "type",
    "additionalProperties",
    "required",
    "properties",
    "const",
    "enum",
    "items",
    "uniqueItems",
    "pattern",
    "minimum",
    "not",
  ]);
  function validate(value, rule, path = "$", depth = 0) {
    if (depth > 64) throw new Error(`${path} exceeds maximum depth`);
    for (const keyword of Object.keys(rule)) {
      if (!supported.has(keyword))
        throw new Error(`Unsupported schema keyword ${keyword}`);
    }
    if (rule.$ref) {
      if (!rule.$ref.startsWith("#/$defs/"))
        throw new Error("Unsupported schema reference");
      return validate(value, schema.$defs[rule.$ref.slice(8)], path, depth + 1);
    }
    const kind = Array.isArray(value)
      ? "array"
      : value === null
        ? "null"
        : typeof value;
    const validType =
      rule.type === "integer"
        ? Number.isSafeInteger(value)
        : kind === rule.type;
    if (rule.type && !validType)
      throw new Error(`${path} must be ${rule.type}`);
    if (Object.hasOwn(rule, "const") && value !== rule.const)
      throw new Error(`${path} has an unsupported value`);
    if (rule.enum && !rule.enum.includes(value))
      throw new Error(`${path} has an unsupported value`);
    if (rule.minimum !== undefined && value < rule.minimum)
      throw new Error(`${path} is below its minimum`);
    if (rule.pattern && !new RegExp(rule.pattern).test(value))
      throw new Error(`${path} has an invalid format`);
    if (kind === "object") {
      for (const key of rule.required || []) {
        if (!Object.hasOwn(value, key))
          throw new Error(`${path}.${key} is required`);
      }
      if (rule.not?.required?.every((key) => Object.hasOwn(value, key)))
        throw new Error(`${path} combines incompatible fields`);
      for (const [key, child] of Object.entries(value)) {
        if (Object.hasOwn(rule.properties || {}, key)) {
          validate(child, rule.properties[key], `${path}.${key}`, depth + 1);
        } else if (rule.additionalProperties === false) {
          throw new Error(`${path}.${key} is not supported`);
        } else if (typeof rule.additionalProperties === "object") {
          validate(
            child,
            rule.additionalProperties,
            `${path}.${key}`,
            depth + 1,
          );
        }
      }
    }
    if (kind === "array") {
      if (
        rule.uniqueItems &&
        new Set(value.map((item) => JSON.stringify(item))).size !== value.length
      )
        throw new Error(`${path} has duplicate entries`);
      value.forEach((item, index) =>
        validate(item, rule.items || {}, `${path}[${index}]`, depth + 1),
      );
    }
  }

  function validateReport(value) {
    validate(value, schema);
    for (const change of value.changes) {
      const keys = Object.keys(change.range);
      const expected = Object.hasOwn(change.range, "url")
        ? ["verb", "url"]
        : dimensions.filter((dimension) => dimension !== "url");
      if (
        keys.length !== expected.length ||
        expected.some((key) => !keys.includes(key))
      )
        throw new Error("A change has an incomplete request range");
      for (const [dimension, selector] of Object.entries(change.range)) {
        if (
          dimension !== "url" &&
          (selector.prefix !== undefined ||
            selector.exceptPrefixes !== undefined)
        )
          throw new Error("Prefix selectors require a URL range");
        if (
          selector.value !== undefined &&
          (selector.except !== undefined ||
            selector.exceptPrefixes !== undefined)
        )
          throw new Error("Exact selectors cannot contain exclusions");
      }
    }
  }

  function matchesSelector(selector, value) {
    if (!selector) return false;
    if (selector.value !== undefined) return selector.value === value;
    if (selector.except?.includes(value)) return false;
    if (selector.prefix !== undefined && !value.startsWith(selector.prefix))
      return false;
    return !selector.exceptPrefixes?.some((prefix) => value.startsWith(prefix));
  }

  function selectorText(selector, dimension) {
    if (selector.value !== undefined)
      return selector.value === ""
        ? t(dimension === "apiGroup" ? "core" : "emptyValue")
        : selector.value;
    let text =
      selector.prefix !== undefined
        ? `${t("prefix")} ${JSON.stringify(selector.prefix)}`
        : t("allValues");
    if (selector.except?.length)
      text += ` · ${t("except")} ${selector.except.map((value) => JSON.stringify(value)).join(", ")}`;
    if (selector.exceptPrefixes?.length)
      text += ` · ${t("exceptPrefixes")} ${selector.exceptPrefixes.map((value) => JSON.stringify(value)).join(", ")}`;
    return text;
  }

  function rangeTitle(change) {
    const range = change.range;
    const resource = range.url || range.resource;
    const base = selectorText(resource, range.url ? "url" : "resource");
    const sub = range.subresource;
    return `${selectorText(range.verb, "verb")} ${base}${sub && sub.value !== "" ? ` / ${selectorText(sub, "subresource")}` : ""}`;
  }

  function incomplete() {
    return (
      report.scope.calculation !== "complete" ||
      report.scope.identityCompleteness !== "complete" ||
      report.scope.policyCompleteness === "incomplete"
    );
  }

  function localize() {
    document.documentElement.lang = language === "zh" ? "zh-CN" : "en";
    document.title = `GrantDiff · ${t("review")}`;
    document.querySelectorAll("[data-t]").forEach((element) => {
      element.textContent = t(element.dataset.t);
    });
    document.querySelectorAll("[data-placeholder]").forEach((element) => {
      element.placeholder = t(element.dataset.placeholder);
    });
    $("language").textContent = language === "zh" ? "English" : "中文";
    $("language").setAttribute(
      "aria-label",
      language === "zh" ? "Switch to English" : "切换为中文",
    );
    if (report) renderReport();
  }

  function renderReport() {
    $("report-title").textContent = t("title");
    $("report-meta").textContent =
      `${label ? `${label} · ` : ""}${t("meta")(report.identities, report.comparedClasses)}`;
    $("calculation").textContent = t(incomplete() ? "incomplete" : "complete");
    $("calculation").className = `status${incomplete() ? " incomplete" : ""}`;
    $("incomplete").hidden = !incomplete();
    $("incomplete-text").textContent = t("incompleteText");
    renderScope();
    renderTypes();
    const currentIdentity = $("identity").value;
    $("identity").replaceChildren(new Option(t("allIdentities"), ""));
    const identities = [
      ...new Set(
        report.changes.map(
          (change) => JSON.stringify([change.targetType, change.subject]),
        ),
      ),
    ].sort();
    identities.forEach((value) => {
      const [kind, subject] = JSON.parse(value);
      $("identity").add(new Option(`${subject} · ${t(kind)}`, value));
    });
    $("identity").value = identities.includes(currentIdentity)
      ? currentIdentity
      : "";
    applyFilters(false);
  }

  function renderScope() {
    const grid = node("dl", undefined, "scope-grid");
    const values = [
      ["beforeSource", report.scope.beforeSource],
      ["afterSource", report.scope.afterSource],
      ["policies", t(report.scope.policyCompleteness)],
      ["identities", t(report.scope.identityCompleteness)],
      ["calculation", t(report.scope.calculation)],
      ["source", report.scope.source],
      ["beforeDigest", report.scope.beforeDigest],
      ["afterDigest", report.scope.afterDigest],
    ];
    values.forEach(([key, value]) => {
      const group = node("div");
      group.append(node("dt", t(key)), node("dd", value));
      grid.append(group);
    });
    $("scope-content").replaceChildren(grid);
    if (!report.diagnostics.length)
      $("scope-content").append(node("p", t("noDiagnostics"), "diagnostic"));
    report.diagnostics.forEach((diagnostic) => {
      const item = node("div", undefined, "diagnostic");
      item.append(
        node("strong", diagnostic.code),
        node("p", diagnostic.message),
      );
      if (diagnostic.object) item.append(node("p", diagnostic.object));
      if (diagnostic.source)
        item.append(node("p", sourceText(diagnostic.source), "source"));
      $("scope-content").append(item);
    });
  }

  function renderTypes() {
    const counts = { all: report.changes.length };
    for (const change of report.changes)
      counts[change.type] = (counts[change.type] || 0) + 1;
    $("type-filters").replaceChildren();
    types.forEach((type) => {
      const button = node(
        "button",
        undefined,
        `type-button${activeType === type ? " selected" : ""}`,
      );
      button.type = "button";
      button.dataset.type = type;
      button.setAttribute("aria-pressed", String(activeType === type));
      const dot = node("span", undefined, `type-dot ${type}`);
      dot.setAttribute("aria-hidden", "true");
      button.append(
        dot,
        node("span", t(type)),
        node("span", (counts[type] || 0).toLocaleString(), "type-count"),
      );
      button.addEventListener("click", () => {
        activeType = type;
        renderTypes();
        applyFilters();
      });
      $("type-filters").append(button);
    });
  }

  function applyFilters(resetPage = true) {
    if (!report) return;
    if (resetPage) page = 0;
    const query = $("search").value.trim().toLowerCase();
    const identity = $("identity").value;
    const namespace = $("namespace").value.trim();
    const clusterScope = $("cluster-scope").checked;
    $("namespace").disabled = clusterScope;
    filtered = [];
    report.changes.forEach((change, index) => {
      if (activeType !== "all" && change.type !== activeType) return;
      if (
        identity &&
        JSON.stringify([change.targetType, change.subject]) !== identity
      )
        return;
      if (
        $("risk").value === "high" &&
        !change.risk.some((risk) => risk.severity === "high")
      )
        return;
      if (
        (namespace || clusterScope) &&
        !matchesSelector(change.range.namespace, clusterScope ? "" : namespace)
      )
        return;
      if (query && !searchIndex[index].includes(query)) return;
      filtered.push(index);
    });
    page = Math.min(
      page,
      Math.max(0, Math.ceil(filtered.length / pageSize) - 1),
    );
    const visible = filtered.slice(page * pageSize, (page + 1) * pageSize);
    if (!visible.includes(selected)) selected = visible[0] ?? -1;
    renderRows();
    renderDetail();
  }

  function renderRows() {
    $("match-count").textContent = t("matches")(
      filtered.length,
      report.changes.length,
    );
    $("rows").replaceChildren();
    if (!filtered.length) {
      const empty = node("div", undefined, "empty");
      const noReportChanges = !report.changes.length;
      const title = noReportChanges
        ? incomplete()
          ? "unknownEmpty"
          : "noChanges"
        : "noMatches";
      const text = noReportChanges
        ? incomplete()
          ? "unknownEmptyText"
          : "noChangesText"
        : "noMatchesText";
      empty.append(node("h2", t(title)), node("p", t(text)));
      $("rows").append(empty);
    }
    filtered.slice(page * pageSize, (page + 1) * pageSize).forEach((index) => {
      const change = report.changes[index];
      const button = node(
        "button",
        undefined,
        `change-row${selected === index ? " selected" : ""}`,
      );
      button.type = "button";
      button.setAttribute("aria-pressed", String(selected === index));
      const top = node("div", undefined, "row-top");
      top.append(node("span", t(change.type), `change-tag ${change.type}`));
      if (change.risk.some((risk) => risk.severity === "high"))
        top.append(node("span", t("high"), "risk-badge"));
      button.append(
        top,
        node("p", rangeTitle(change), "row-resource"),
        node("p", change.subject, "row-subject"),
      );
      if (change.range.namespace)
        button.append(
          node(
            "p",
            `${t("namespace")} · ${selectorText(change.range.namespace, "namespace")}`,
            "row-scope",
          ),
        );
      button.addEventListener("click", () => {
        selected = index;
        $("rows")
          .querySelectorAll(".change-row")
          .forEach((row) => {
            row.classList.remove("selected");
            row.setAttribute("aria-pressed", "false");
          });
        button.classList.add("selected");
        button.setAttribute("aria-pressed", "true");
        renderDetail();
        if (window.innerWidth < 851)
          $("detail").scrollIntoView({ block: "start" });
      });
      $("rows").append(button);
    });
    $("page-count").textContent = t("page")(
      filtered.length ? page + 1 : 0,
      Math.ceil(filtered.length / pageSize),
    );
    $("previous").disabled = page === 0;
    $("next").disabled = (page + 1) * pageSize >= filtered.length;
  }

  function sourceText(source) {
    return `${source.file} · ${t("document")} ${source.document}${source.line ? ` · ${t("line")} ${source.line}` : ""}`;
  }

  function renderPaths(evidence, side) {
    const column = node("div", undefined, "path-side");
    column.append(
      node("h3", `${t(side)} · ${t("pathCount")(evidence.length)}`),
    );
    if (!evidence.length) column.append(node("p", t("noEvidence"), "muted"));
    let rendered = 0;
    const more = node("button", t("morePaths"), "quiet");
    more.type = "button";
    function appendPaths() {
      const end = Math.min(rendered + 20, evidence.length);
      for (; rendered < end; rendered++) {
        const item = evidence[rendered];
        const detail = node("details", undefined, "path");
        detail.open = rendered === 0;
        detail.append(node("summary", item.binding));
        const body = node("div", undefined, "path-body");
        const steps = [
          ["subject", item.subject],
          ["binding", item.binding],
          ...(item.aggregationPath || []).map((value) => [
            "aggregation",
            value,
          ]),
          ["role", item.role],
        ];
        steps.forEach(([key, value]) => {
          const step = node("div", undefined, "path-node");
          step.append(node("span", t(key)), node("div", value));
          body.append(step);
        });
        body.append(
          node(
            "p",
            `${t("bindingSource")} · ${sourceText(item.bindingSource)}`,
            "source",
          ),
          node(
            "p",
            `${t("ruleSource")} · ${sourceText(item.ruleSource)}`,
            "source",
          ),
          node("p", `${t("rule")} ${item.ruleId}`, "rule-id"),
        );
        detail.append(body);
        column.insertBefore(detail, more);
      }
      more.hidden = rendered >= evidence.length;
    }
    column.append(more);
    more.addEventListener("click", appendPaths);
    appendPaths();
    return column;
  }

  function renderDetail() {
    const target = $("detail");
    target.replaceChildren();
    if (selected < 0) {
      const empty = node("div", undefined, "empty");
      empty.append(node("h2", t("selectChange")), node("p", t("selectText")));
      target.append(empty);
      return;
    }
    const change = report.changes[selected];
    const heading = node("div", undefined, "detail-heading");
    heading.append(
      node("span", t(change.type), `change-tag ${change.type}`),
      node("span", t(change.targetType), "muted"),
    );
    target.append(
      heading,
      node("h2", rangeTitle(change)),
      node("p", change.subject, "subject-heading"),
    );
    const decisions = node("div", undefined, "decision-grid");
    ["before", "after"].forEach((side) => {
      const card = node("div", undefined, `decision ${change[side]}`);
      card.append(
        node("span", t(side), "label"),
        node("strong", t(change[side])),
      );
      decisions.append(card);
    });
    const explanations = {
      added: "addedExplanation",
      removed: "removedExplanation",
      retained_via_other_path: "retainedExplanation",
      path_changed: "pathExplanation",
      indeterminate: "unknownExplanation",
    };
    target.append(
      decisions,
      node("p", t(explanations[change.type]), "explanation"),
    );
    if (change.risk.length) {
      const riskSection = node("section", undefined, "detail-section");
      riskSection.append(node("h3", t("risks")));
      change.risk.forEach((risk) => {
        const item = node("div", undefined, "risk-item");
        item.append(node("strong", risk.id), node("p", risk.message));
        riskSection.append(item);
      });
      target.append(riskSection);
    }
    const rangeSection = node("section", undefined, "detail-section");
    rangeSection.append(node("h3", t("range")));
    const table = node("table", undefined, "range-table");
    dimensions.forEach((dimension) => {
      if (!change.range[dimension]) return;
      const row = node("tr");
      const header = node("th", t(dimension));
      header.scope = "row";
      row.append(
        header,
        node("td", selectorText(change.range[dimension], dimension)),
      );
      table.append(row);
    });
    rangeSection.append(table);
    const example = node("details", undefined, "example");
    example.append(
      node("summary", `${t("example")} · ${t("exampleNote")}`),
      node("pre", JSON.stringify(change.representativeRequest, null, 2)),
    );
    rangeSection.append(example);
    const pathsSection = node("section", undefined, "detail-section");
    pathsSection.append(node("h3", t("paths")));
    const paths = node("div", undefined, "paths");
    paths.append(
      renderPaths(change.beforeEvidence, "before"),
      renderPaths(change.afterEvidence, "after"),
    );
    pathsSection.append(paths);
    target.append(rangeSection, pathsSection);
  }

  function acceptReport(value, name) {
    validateReport(value);
    report = value;
    label = name;
    selected = -1;
    searchIndex = report.changes.map((change) =>
      [
        change.subject,
        change.targetType,
        ...Object.values(change.range).map((selector) =>
          JSON.stringify(selector),
        ),
        ...change.beforeEvidence
          .concat(change.afterEvidence)
          .map(
            (item) =>
              `${item.subject} ${item.binding} ${item.role} ${item.ruleSource.file} ${(item.aggregationPath || []).join(" ")}`,
          ),
        ...change.risk.map((risk) => `${risk.id} ${risk.message}`),
      ]
        .join(" ")
        .toLowerCase(),
    );
    $("welcome").hidden = true;
    $("workspace").hidden = false;
    $("error").hidden = true;
    $("scope-panel").open = incomplete();
    resetFilters();
  }

  function resetFilters() {
    activeType = "all";
    page = 0;
    ["identity", "namespace", "search"].forEach((id) => {
      $(id).value = "";
    });
    $("risk").value = "all";
    $("cluster-scope").checked = false;
    if (report) renderReport();
  }

  function showError(error) {
    $("error").textContent =
      `${t("invalid")} · ${t("unsupported")} ${error.message}`;
    $("error").hidden = false;
  }

  $("open-report").addEventListener("click", () => $("file").click());
  $("welcome-open").addEventListener("click", () => $("file").click());
  $("file").addEventListener("change", async (event) => {
    const file = event.target.files[0];
    if (!file) return;
    const sequence = ++loadSequence;
    $("open-report").textContent = t("reading");
    try {
      if (file.size > maxBytes) throw new Error(t("tooLarge"));
      const text = await file.text();
      if (sequence !== loadSequence) return;
      acceptReport(JSON.parse(text.replace(/^\uFEFF/, "")), file.name);
    } catch (error) {
      if (sequence === loadSequence) showError(error);
    } finally {
      if (sequence === loadSequence) {
        $("open-report").textContent = t("open");
        $("file").value = "";
      }
    }
  });
  $("language").addEventListener("click", () => {
    language = language === "zh" ? "en" : "zh";
    localize();
  });
  ["identity", "risk", "cluster-scope"].forEach((id) =>
    $(id).addEventListener("change", () => applyFilters()),
  );
  let searchTimer;
  ["search", "namespace"].forEach((id) =>
    $(id).addEventListener("input", () => {
      clearTimeout(searchTimer);
      searchTimer = setTimeout(() => applyFilters(), 120);
    }),
  );
  $("reset").addEventListener("click", resetFilters);
  $("previous").addEventListener("click", () => {
    if (page > 0) {
      page--;
      selected = filtered[page * pageSize];
      renderRows();
      renderDetail();
      $("rows").scrollTop = 0;
    }
  });
  $("next").addEventListener("click", () => {
    if ((page + 1) * pageSize < filtered.length) {
      page++;
      selected = filtered[page * pageSize];
      renderRows();
      renderDetail();
      $("rows").scrollTop = 0;
    }
  });
  localize();
  try {
    const initial = JSON.parse($("initial-report").textContent);
    if (initial) acceptReport(initial, "");
  } catch (error) {
    showError(error);
  }
})();
