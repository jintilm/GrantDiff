package diff

import (
	"encoding/json"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/jintilm/GrantDiff/internal/input"
	"github.com/jintilm/GrantDiff/internal/model"
)

func TestCompactPreservesCoverageEvidenceAndRisk(t *testing.T) {
	before, after, identities := fixtures(t)
	// Load a policy with overlapping names, namespaces, URL prefixes and risks.
	data := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata: {name: wide}
rules:
  - apiGroups: [""]
    resources: [configmaps, secrets]
    verbs: [get, list, watch]
  - nonResourceURLs: [/healthz, /healthz/*, /healthz/live, /healthz/live/*]
    verbs: [get]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata: {name: wide}
roleRef: {apiGroup: rbac.authorization.k8s.io, kind: ClusterRole, name: wide}
subjects: [{kind: ServiceAccount, namespace: finance, name: report-bot}]
`
	wide, err := input.Parse([]input.File{{Name: "wide.yaml", Data: []byte(data)}}, "test", input.Options{})
	if err != nil {
		t.Fatal(err)
	}
	for key, object := range wide.Objects {
		after.Objects[key] = object
	}
	before.Catalog.Resources = append(before.Catalog.Resources, model.Resource{Name: "secrets", Namespaced: true})
	after.Catalog = before.Catalog
	raw := Compare(before, after, identities, 100000)
	encodedBefore, _ := json.Marshal(raw)
	compact := Compact(raw)
	if len(compact.Changes) >= len(raw.Changes) {
		t.Fatalf("compaction did not reduce %d ranges", len(raw.Changes))
	}
	if compact.Scope != raw.Scope || compact.Compared != raw.Compared || compact.Identities != raw.Identities || model.Hash(compact.Diagnostics) != model.Hash(raw.Diagnostics) {
		t.Fatal("compaction changed review metadata")
	}
	for _, verb := range []string{"get", "list", "watch", "patch", "future"} {
		for _, resource := range []string{"configmaps", "secrets", "pods", "future"} {
			for _, namespace := range []string{"", "finance", "other", "future"} {
				for _, name := range []string{"", "billing-config", "other", "future"} {
					request := model.Request{Verb: verb, Resource: resource, Namespace: namespace, Name: name}
					assertSameCoverage(t, raw, compact, request)
				}
			}
		}
		for _, url := range []string{"/", "/healthz", "/healthz/", "/healthz/ready", "/healthz/live", "/healthz/live/", "/healthz/live/ready", "/private"} {
			assertSameCoverage(t, raw, compact, model.Request{Verb: verb, URL: url})
		}
	}
	for _, change := range compact.Changes {
		if !regionContains(change.Range, change.Request) {
			t.Fatalf("representative request escaped its range: %+v", change)
		}
		if change.ID != model.Hash([]any{change.TargetType, change.Subject, change.Type, change.Range})[:20] {
			t.Fatal("merged change has an incorrect ID")
		}
	}
	encodedAfter, _ := json.Marshal(raw)
	if string(encodedBefore) != string(encodedAfter) {
		t.Fatal("compaction mutated its input")
	}
	if model.Hash(compact) != model.Hash(Compact(compact)) {
		t.Fatal("compaction is not idempotent")
	}
	random := rand.New(rand.NewPCG(20261120, 6))
	random.Shuffle(len(raw.Changes), func(i, j int) { raw.Changes[i], raw.Changes[j] = raw.Changes[j], raw.Changes[i] })
	if model.Hash(compact) != model.Hash(Compact(raw)) {
		t.Fatal("compaction depends on input order")
	}
}

func TestCompactPreservesHolesAndDistinctSemantics(t *testing.T) {
	value := "special"
	base := Change{TargetType: "identity_access", Subject: "alice", Type: "added", Before: "not_granted", After: "granted",
		Range:         Region{"name": {Except: []string{value, "restricted"}}},
		AfterEvidence: []model.Evidence{{Binding: "access", Role: "reader", RuleSource: model.Source{File: "policy.yaml", Line: 3}}},
	}
	exact := base
	exact.Range = Region{"name": {Value: &value}}
	variants := map[string]func(*Change){
		"subject":  func(c *Change) { c.Subject = "bob" },
		"target":   func(c *Change) { c.TargetType = "group_grant" },
		"type":     func(c *Change) { c.Type = "removed" },
		"decision": func(c *Change) { c.After = "unknown" },
		"risk":     func(c *Change) { c.Risk = []Risk{{ID: "secret_read", Severity: "high"}} },
		"evidence location": func(c *Change) {
			c.AfterEvidence = slices.Clone(c.AfterEvidence)
			c.AfterEvidence[0].RuleSource.Line = 4
		},
	}
	for name, mutate := range variants {
		t.Run(name, func(t *testing.T) {
			other := exact
			mutate(&other)
			result := Compact(Report{Changes: []Change{base, other}})
			if len(result.Changes) != 2 {
				t.Fatal("distinct semantics were merged")
			}
		})
	}
	result := Compact(Report{Changes: []Change{base, exact}})
	if len(result.Changes) != 1 || contains(result.Changes[0].Range["name"], "restricted") || !contains(result.Changes[0].Range["name"], value) {
		t.Fatal("merging added access through an excluded hole")
	}
	otherValue := "second"
	other := exact
	other.Range = Region{"name": {Value: &otherValue}}
	if len(Compact(Report{Changes: []Change{exact, other}}).Changes) != 2 {
		t.Fatal("two exact values cannot become an unrestricted selector")
	}
}

func TestCompactURLExclusionsAndIncompleteReport(t *testing.T) {
	prefix, nested, exact := "/healthz/", "/healthz/private/", "/healthz/private/check"
	root := Change{Subject: "alice", Type: "indeterminate", Range: Region{"url": {Prefix: &prefix, Except: []string{exact}, ExceptPrefixes: []string{nested}}}}
	child := root
	child.Range = Region{"url": {Value: &exact}}
	report := Report{Scope: Scope{Calculation: "incomplete"}, Compared: 2, Diagnostics: []model.Diagnostic{{Code: "budget_exceeded"}}, Changes: []Change{root, child}}
	result := Compact(report)
	if len(result.Changes) != 2 || result.Scope != report.Scope || result.Compared != report.Compared || model.Hash(result.Diagnostics) != model.Hash(report.Diagnostics) {
		t.Fatal("prefix exclusion or incomplete metadata changed")
	}
}

func regionContains(region Region, request model.Request) bool {
	if _, url := region["url"]; url != (request.URL != "") {
		return false
	}
	values := map[string]string{"verb": request.Verb, "apiGroup": request.APIGroup, "resource": request.Resource, "subresource": request.Subresource, "namespace": request.Namespace, "name": request.Name, "url": request.URL}
	for dimension, selector := range region {
		if !contains(selector, values[dimension]) {
			return false
		}
	}
	return true
}

func assertSameCoverage(t *testing.T, raw, compact Report, request model.Request) {
	t.Helper()
	matched := func(report Report) []string {
		results := []string{}
		for _, change := range report.Changes {
			if regionContains(change.Range, request) {
				change.ID, change.Range, change.Request = "", nil, model.Request{}
				results = append(results, model.Hash(change))
			}
		}
		slices.Sort(results)
		return results
	}
	if !slices.Equal(matched(raw), matched(compact)) {
		t.Fatalf("coverage or semantics changed for %+v", request)
	}
}
