package evaluate

import (
	"strings"
	"testing"

	"github.com/jintilm/GrantDiff/internal/input"
	"github.com/jintilm/GrantDiff/internal/model"
	rbac "k8s.io/api/rbac/v1"
)

func TestRuleMatching(t *testing.T) {
	tests := []struct {
		name    string
		rule    rbac.PolicyRule
		request model.Request
		want    bool
	}{
		{"core", rbac.PolicyRule{Verbs: []string{"get"}, APIGroups: []string{""}, Resources: []string{"pods"}}, model.Request{Verb: "get", Resource: "pods"}, true},
		{"subresource separate", rbac.PolicyRule{Verbs: []string{"get"}, APIGroups: []string{""}, Resources: []string{"pods"}}, model.Request{Verb: "get", Resource: "pods", Subresource: "log"}, false},
		{"subresource wildcard", rbac.PolicyRule{Verbs: []string{"get"}, APIGroups: []string{""}, Resources: []string{"*/log"}}, model.Request{Verb: "get", Resource: "pods", Subresource: "log"}, true},
		{"star name literal", rbac.PolicyRule{Verbs: []string{"get"}, APIGroups: []string{""}, Resources: []string{"pods"}, ResourceNames: []string{"*"}}, model.Request{Verb: "get", Resource: "pods", Name: "app"}, false},
		{"name missing", rbac.PolicyRule{Verbs: []string{"list"}, APIGroups: []string{""}, Resources: []string{"pods"}, ResourceNames: []string{"app"}}, model.Request{Verb: "list", Resource: "pods"}, false},
		{"url prefix", rbac.PolicyRule{Verbs: []string{"get"}, NonResourceURLs: []string{"/healthz/*"}}, model.Request{Verb: "get", URL: "/healthz/live"}, true},
		{"url parent", rbac.PolicyRule{Verbs: []string{"get"}, NonResourceURLs: []string{"/healthz/*"}}, model.Request{Verb: "get", URL: "/healthz"}, false},
		{"group different", rbac.PolicyRule{Verbs: []string{"get"}, APIGroups: []string{"apps"}, Resources: []string{"deployments"}}, model.Request{Verb: "get", Resource: "deployments"}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Allows(test.rule, test.request); got != test.want {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestInvalidQueryShapes(t *testing.T) {
	catalog := model.Catalog{Resources: []model.Resource{{Name: "pods", Namespaced: true, Subresources: []string{"log", "exec"}}}}
	tests := []model.Request{
		{Verb: "get", Resource: "pods", Namespace: "finance"},
		{Verb: "get", Resource: "pods", Subresource: "invented", Namespace: "finance", Name: "app"},
		{Verb: "get", URL: "/healthz", APIGroup: "apps"},
		{Verb: "create", Resource: "pods", Subresource: "exec", Namespace: "finance"},
	}
	for _, request := range tests {
		if err := CheckRequest(request, catalog); err == nil {
			t.Errorf("invalid request was accepted: %+v", request)
		}
	}
}

func TestRetainedAndNamespace(t *testing.T) {
	state, err := input.Load("../../examples/retained/after", input.Options{})
	if err != nil {
		t.Fatal(err)
	}
	principal := New(state).Prepare(model.Identity{Kind: "ServiceAccount", Name: "report-bot", Namespace: "finance"})
	request := model.Request{Verb: "get", Resource: "configmaps", Name: "billing-config", Namespace: "finance"}
	decision := principal.Decide(request)
	if decision.Result != "granted" || len(decision.Evidence) != 1 || !strings.Contains(decision.Evidence[0].Subject, "Group/") {
		t.Fatalf("unexpected decision: %+v", decision)
	}
	request.Namespace = "other"
	if principal.Decide(request).Result != "not_granted" {
		t.Fatal("namespace grant leaked")
	}
}

func TestMissingRoleAndIncompleteGroups(t *testing.T) {
	state, err := input.Load("../../examples/retained/after", input.Options{})
	if err != nil {
		t.Fatal(err)
	}
	delete(state.Objects, "Role/finance/report-reader")
	decision := New(state).Prepare(model.Identity{Kind: "ServiceAccount", Name: "report-bot", Namespace: "finance"}).Decide(model.Request{Verb: "get", Resource: "configmaps", Namespace: "finance"})
	if decision.Result != "unknown" {
		t.Fatalf("missing role became %s", decision.Result)
	}
	decision = New(model.NewState()).Prepare(model.Identity{Kind: "User", Name: "alice"}).Decide(model.Request{Verb: "get", Resource: "pods"})
	if decision.Result != "unknown" {
		t.Fatal("incomplete groups reported a negative conclusion")
	}
}

func TestAggregation(t *testing.T) {
	data := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata: {name: reader}
aggregationRule:
  clusterRoleSelectors:
    - matchLabels: {review: included}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata: {name: leaf, labels: {review: included}}
rules:
  - apiGroups: [""]
    resources: [nodes]
    verbs: [get]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata: {name: binding}
subjects: [{kind: User, name: alice}]
roleRef: {kind: ClusterRole, name: reader, apiGroup: rbac.authorization.k8s.io}
`
	state, err := input.Parse([]input.File{{Name: "test.yaml", Data: []byte(data)}}, "test", input.Options{})
	if err != nil {
		t.Fatal(err)
	}
	identity := model.Identity{Kind: "User", Name: "alice", GroupsComplete: true}
	decision := New(state).Prepare(identity).Decide(model.Request{Verb: "get", Resource: "nodes"})
	if decision.Result != "granted" || len(decision.Evidence[0].AggregationPath) != 1 {
		t.Fatalf("unexpected aggregate: %+v", decision)
	}
	state.Objects["ClusterRole//leaf"].AggregationRule = state.Objects["ClusterRole//reader"].AggregationRule.DeepCopy()
	state.Objects["ClusterRole//reader"].Metadata.Labels = map[string]string{"review": "included"}
	decision = New(state).Prepare(identity).Decide(model.Request{Verb: "get", Resource: "nodes"})
	if decision.Result != "unknown" {
		t.Fatalf("cycle returned %s", decision.Result)
	}
}
