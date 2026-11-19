package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jintilm/GrantDiff/internal/diff"
)

func TestNormalCLI(t *testing.T) {
	args := []string{"diff", "--before", "../../examples/retained/before", "--after", "../../examples/retained/after", "--identities", "../../examples/retained/identities.yaml", "--catalog", "../../examples/retained/catalog.yaml", "--format", "json"}
	var stdout, stderr bytes.Buffer
	if code := Run(args, &stdout, &stderr); code != 0 {
		t.Fatalf("code %d: %s", code, stderr.String())
	}
	var result diff.Report
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Changes) != 1 || result.Changes[0].Type != "retained_via_other_path" {
		t.Fatalf("unexpected result: %s", stdout.String())
	}
	path := filepath.Join(t.TempDir(), "报告.json")
	for range 2 {
		if code := Run(append(args, "--output", path), &stdout, &stderr); code != 0 {
			t.Fatalf("output overwrite failed: %s", stderr.String())
		}
	}
	if _, err := os.ReadFile(path); err != nil {
		t.Fatal(err)
	}
}

func TestExitCodes(t *testing.T) {
	tests := []struct {
		args []string
		want int
	}{
		{[]string{"--help"}, 0},
		{[]string{"version"}, 0},
		{[]string{"diff", "--format", "xml"}, 2},
		{[]string{"diff", "--before", "missing"}, 2},
		{[]string{"diff", "--before", "../../examples/retained/before", "--after", "../../examples/retained/after", "--identities", "../../examples/retained/identities.yaml", "--max-classes", "1"}, 3},
		{[]string{"query", "--input", "../../examples/retained/after", "--identities", "../../examples/retained/identities.yaml", "--identity", "report-bot", "--verb", "get", "--resource", "unknown", "--namespace", "finance"}, 3},
	}
	for _, test := range tests {
		var stdout, stderr bytes.Buffer
		if code := Run(test.args, &stdout, &stderr); code != test.want {
			t.Fatalf("%v returned %d, want %d: %s", test.args, code, test.want, stderr.String())
		}
	}
}

func TestWildcardUnknownCannotPassCI(t *testing.T) {
	directory := t.TempDir()
	data, err := os.ReadFile("../../examples/retained/before/rbac.yaml")
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.Replace(data, []byte("resources: [configmaps]"), []byte("resources: ['*']"), 1)
	file := filepath.Join(directory, "wildcard.yaml")
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"diff", "--before", file, "--after", "../../examples/retained/after", "--identities", "../../examples/retained/identities.yaml", "--catalog", "../../examples/retained/catalog.yaml", "--format", "json"}
	var stdout, stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	var result diff.Report
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	for _, change := range result.Changes {
		if change.Type == "indeterminate" && code != 3 {
			t.Fatalf("indeterminate report returned %d, want 3", code)
		}
	}
}

func TestOmittedCatalogScopeCannotHideChanges(t *testing.T) {
	directory := t.TempDir()
	catalog := filepath.Join(directory, "catalog.yaml")
	if err := os.WriteFile(catalog, []byte("schemaVersion: '1.0'\nresources:\n  - apiGroup: ''\n    name: configmaps\n"), 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"diff", "--before", "../../examples/retained/before", "--after", "../../examples/retained/after", "--identities", "../../examples/retained/identities.yaml", "--catalog", catalog, "--format", "json"}
	var stdout, stderr bytes.Buffer
	if code := Run(args, &stdout, &stderr); code != 2 {
		t.Fatalf("invalid catalog returned %d; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
}

func TestQueryAcceptsTypedRBACLists(t *testing.T) {
	file := filepath.Join(t.TempDir(), "export.yaml")
	data := `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleList
items:
  - metadata: {name: reader, namespace: finance}
    rules: [{apiGroups: [""], resources: [configmaps], resourceNames: [billing-config], verbs: [get]}]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBindingList
items:
  - metadata: {name: access, namespace: finance}
    subjects: [{kind: ServiceAccount, name: report-bot, namespace: finance}]
    roleRef: {apiGroup: rbac.authorization.k8s.io, kind: Role, name: reader}
`
	if err := os.WriteFile(file, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"query", "--input", file, "--identities", "../../examples/retained/identities.yaml", "--catalog", "../../examples/retained/catalog.yaml", "--identity", "report-bot", "--verb", "get", "--resource", "configmaps", "--name", "billing-config", "--namespace", "finance", "--format", "json"}
	var stdout, stderr bytes.Buffer
	if code := Run(args, &stdout, &stderr); code != 0 {
		t.Fatalf("query returned %d: %s", code, stderr.String())
	}
	var result struct {
		Decision struct {
			Result string `json:"result"`
		} `json:"decision"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Decision.Result != "granted" {
		t.Fatalf("typed export lost authorization: %s", stdout.String())
	}
}
