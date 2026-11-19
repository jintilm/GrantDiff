package input

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListAndMultiDocumentInputsAgree(t *testing.T) {
	expected, err := Load("../../examples/retained/before", Options{})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(map[string]any{"apiVersion": "v1", "kind": "List", "items": expected.SortedObjects()}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	actual, err := Parse([]File{{Name: "export.json", Data: data}}, "test", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if actual.Digest() != expected.Digest() {
		t.Fatal("List decoding changed policy objects")
	}
	for _, object := range actual.Objects {
		if object.Source.File != "export.json" || object.Source.Line <= 1 {
			t.Fatalf("List item location was lost: %+v", object.Source)
		}
	}
}

func TestTypedListItemsInheritTypeMetadata(t *testing.T) {
	data := `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleList
items:
  - metadata: {name: reader, namespace: finance}
    rules: [{apiGroups: [""], resources: [configmaps], verbs: [get]}]
`
	state, err := Parse([]File{{Name: "roles.yaml", Data: []byte(data)}}, "test", Options{})
	if err != nil {
		t.Fatal(err)
	}
	role := state.Objects["Role/finance/reader"]
	if role == nil || role.APIVersion != "rbac.authorization.k8s.io/v1" {
		t.Fatalf("typed List lost type metadata: %+v", state)
	}
	if role.Source.Line != 4 {
		t.Fatalf("item location changed: %+v", role.Source)
	}
}

func TestTypedListsRejectConflictingTypeMetadata(t *testing.T) {
	data := `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleList
items:
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRole
    metadata: {name: reader}
    rules: [{apiGroups: [""], resources: [nodes], verbs: [get]}]
`
	if _, err := Parse([]File{{Name: "roles.yaml", Data: []byte(data)}}, "test", Options{}); err == nil {
		t.Fatal("RoleList accepted a ClusterRole item")
	}
}

func TestInvalidAndDuplicateInputs(t *testing.T) {
	valid := `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata: {name: test, namespace: finance}
rules: [{apiGroups: [""], resources: [pods], verbs: [get]}]
`
	tests := map[string]string{
		"duplicate object":   valid + "---\n" + valid,
		"duplicate key":      strings.Replace(valid, "kind: Role", "kind: Role\nkind: Role", 1),
		"unknown rule field": strings.Replace(valid, "verbs: [get]", "verb: [get]", 1),
		"missing namespace":  strings.Replace(valid, ", namespace: finance", "", 1),
		"alias":              strings.Replace(valid, "rules: [", "rules: &r [", 1) + "other: *r\n",
		"unknown version":    strings.Replace(valid, "rbac.authorization.k8s.io/v1", "rbac.authorization.k8s.io/v8", 1),
	}
	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]File{{"test.yaml", []byte(data)}}, "test", Options{}); err == nil {
				t.Fatal("accepted invalid input")
			}
		})
	}
}

func TestUnicodePathCRLFAndBOM(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "权限 示例.yaml")
	data := "\ufeffapiVersion: rbac.authorization.k8s.io/v1\r\nkind: Role\r\nmetadata: {name: test, namespace: finance}\r\nrules: []\r\n"
	if err := os.WriteFile(file, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	state, err := Load(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if state.Objects["Role/finance/test"].Source.File != "权限 示例.yaml" {
		t.Fatal("relative unicode source lost")
	}
}

func TestPathEscape(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.yaml")
	if err := os.WriteFile(outside, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape.yaml")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Load(root, Options{}); err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("escaped input root: %v", err)
	}
}
