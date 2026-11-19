package diff

import (
	"slices"
	"testing"

	rbac "k8s.io/api/rbac/v1"

	"github.com/jintilm/GrantDiff/internal/input"
	"github.com/jintilm/GrantDiff/internal/model"
)

func TestWildcardIndeterminateReportIsIncomplete(t *testing.T) {
	before, after, identities := fixtures(t)
	before.Objects["Role/finance/report-reader"].Rules = []rbac.PolicyRule{{APIGroups: []string{""}, Resources: []string{"*"}, Verbs: []string{"get"}}}
	after.Objects["Role/finance/report-reader"].Rules = nil
	result := Compare(before, after, identities, 10000)
	indeterminate := false
	for _, change := range result.Changes {
		indeterminate = indeterminate || change.Type == "indeterminate"
	}
	if indeterminate && result.Scope.Calculation == "complete" {
		t.Fatal("indeterminate wildcard changes were marked complete")
	}
}

func TestWildcardGroupsRespectCatalogScope(t *testing.T) {
	before, after, identities := fixtures(t)
	before.Objects["Role/finance/report-reader"].Rules = nil
	after.Objects["Role/finance/report-reader"].Rules = []rbac.PolicyRule{{APIGroups: []string{"*"}, Resources: []string{"*"}, Verbs: []string{"get"}}}
	catalog := model.Catalog{SchemaVersion: "1.0", Resources: []model.Resource{{APIGroup: "example.test", Name: "widgets", Namespaced: false}}}
	before.Catalog, after.Catalog = catalog, catalog
	result := Compare(before, after, identities, 100000)
	for _, change := range result.Changes {
		if change.Type == "added" && contains(change.Range["apiGroup"], "example.test") && contains(change.Range["resource"], "widgets") {
			t.Fatal("namespaced binding appears to grant cluster-scoped catalog resource through an API-group complement")
		}
	}
}

func fixtures(t *testing.T) (*model.State, *model.State, []model.Identity) {
	t.Helper()
	before, err := input.Load("../../examples/retained/before", input.Options{})
	if err != nil {
		t.Fatal(err)
	}
	after, err := input.Load("../../examples/retained/after", input.Options{})
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := input.LoadCatalog("../../examples/retained/catalog.yaml")
	if err != nil {
		t.Fatal(err)
	}
	before.Catalog, after.Catalog = catalog, catalog
	identities, err := input.LoadIdentities("../../examples/retained/identities.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return before, after, identities
}

func TestRetainedPermission(t *testing.T) {
	before, after, identities := fixtures(t)
	report := Compare(before, after, identities, 100000)
	if report.Scope.Calculation != "complete" || len(report.Changes) != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	change := report.Changes[0]
	if change.Type != "retained_via_other_path" || change.Before != "granted" || change.After != "granted" || len(change.AfterEvidence) != 1 {
		t.Fatalf("unexpected change: %+v", change)
	}
}

func TestWildcardNameReduction(t *testing.T) {
	before, after, identities := fixtures(t)
	delete(before.Objects, "RoleBinding/finance/direct-read")
	before.Objects["Role/finance/report-reader"].Rules[0].ResourceNames = nil
	report := Compare(before, after, identities, 100000)
	found := false
	for _, change := range report.Changes {
		if change.Type == "removed" && slices.Contains(change.Range["name"].Except, "billing-config") {
			found = true
		}
		if change.Type == "removed" && change.Request.Name == "billing-config" {
			t.Fatal("allowed name was marked removed")
		}
	}
	if !found {
		t.Fatalf("lost complement of named permission: %+v", report.Changes)
	}
}

func TestRuleOrderingDoesNotChangePermissions(t *testing.T) {
	_, after, identities := fixtures(t)
	before, err := input.Load("../../examples/retained/after", input.Options{})
	if err != nil {
		t.Fatal(err)
	}
	before.Catalog = after.Catalog
	for _, state := range []*model.State{before, after} {
		state.Objects["Role/finance/report-reader"].Rules[0].Verbs = []string{"get", "list"}
	}
	after.Objects["Role/finance/report-reader"].Rules[0].Verbs = []string{"list", "get"}
	report := Compare(before, after, identities, 1000)
	if len(report.Changes) != 0 {
		t.Fatal("order-only change reported permissions")
	}
}

func TestBudgetIsIncomplete(t *testing.T) {
	before, after, identities := fixtures(t)
	report := Compare(before, after, identities, 1)
	if report.Scope.Calculation != "incomplete" || report.Compared != 1 {
		t.Fatalf("budget incorrectly reported complete: %+v", report)
	}
}

func TestUnknownResourceScopeCannotClaimRevocation(t *testing.T) {
	before, after, identities := fixtures(t)
	before.Catalog, after.Catalog = model.Catalog{}, model.Catalog{}
	delete(after.Objects, "RoleBinding/finance/group-read")
	result := Compare(before, after, identities, 10000)
	if result.Scope.Calculation != "incomplete" {
		t.Fatal("missing catalog was marked complete")
	}
	for _, change := range result.Changes {
		if change.Type != "indeterminate" {
			t.Fatalf("unknown scope claimed %s", change.Type)
		}
	}
}

func TestURLPartition(t *testing.T) {
	patterns := []string{"*", "/", "/health*", "/healthz", "/healthz/*", "/x*", "/xx*"}
	for _, item := range urlCells(patterns) {
		if item.selector.Value != nil {
			continue
		}
		for _, excluded := range item.selector.Except {
			if item.witness == excluded {
				t.Fatal("prefix witness is an excluded exact URL")
			}
		}
		for _, prefix := range item.selector.ExceptPrefixes {
			if len(item.witness) >= len(prefix) && item.witness[:len(prefix)] == prefix {
				t.Fatalf("witness %q lies in excluded prefix %q", item.witness, prefix)
			}
		}
	}
}
