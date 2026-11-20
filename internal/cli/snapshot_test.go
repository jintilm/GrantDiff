package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jintilm/GrantDiff/internal/cluster"
	"github.com/jintilm/GrantDiff/internal/input"
)

func TestSnapshotIntegrityThroughCLI(t *testing.T) {
	data, err := os.ReadFile("../../examples/retained/after/rbac.yaml")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, []byte("\n---\napiVersion: v1\nkind: ServiceAccount\nmetadata: {name: report-bot, namespace: finance}\n")...)
	state, err := input.Parse([]input.File{{Name: "rbac.json", Data: data}}, "fixture", input.Options{})
	if err != nil {
		t.Fatal(err)
	}
	state.Catalog, err = input.LoadCatalog("../../examples/retained/catalog.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, file, old, replacement string
		legacy                       bool
		want                         int
	}{
		{name: "new snapshot", want: 0},
		{name: "legacy snapshot", legacy: true, want: 0},
		{name: "catalog scope changed", file: "catalog.json", old: `"namespaced": true`, replacement: `"namespaced": false`, want: 2},
		{name: "identity changed", file: "rbac.json", old: "name: report-bot", replacement: "name: other-bot", want: 2},
		{name: "RBAC changed", file: "rbac.json", old: "verbs: [get]", replacement: "verbs: [list]", want: 2},
		{name: "missing file digest", file: "metadata.json", old: `"rbac.json"`, replacement: `"missing.json"`, want: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := filepath.Join(t.TempDir(), "snapshot")
			metadata := cluster.Snapshot{SchemaVersion: "1.0", Digest: state.Digest(), Consistency: "two-equal-observations"}
			if err := cluster.Save(directory, state, metadata, data); err != nil {
				t.Fatal(err)
			}
			if test.legacy {
				encoded, _ := json.Marshal(metadata)
				if err := os.WriteFile(filepath.Join(directory, "metadata.json"), encoded, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if test.file != "" {
				path := filepath.Join(directory, test.file)
				content, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				changed := bytes.Replace(content, []byte(test.old), []byte(test.replacement), 1)
				if bytes.Equal(content, changed) {
					t.Fatal("test mutation did not match")
				}
				if err := os.WriteFile(path, changed, 0600); err != nil {
					t.Fatal(err)
				}
			}
			for _, args := range [][]string{
				{"query", "--input", directory, "--identity", "system:serviceaccount:finance:report-bot", "--verb", "get", "--resource", "configmaps", "--namespace", "finance", "--name", "billing-config", "--format", "json"},
				{"diff", "--before", directory, "--after", directory, "--format", "json"},
			} {
				var stdout, stderr bytes.Buffer
				if code := Run(args, &stdout, &stderr); code != test.want {
					t.Fatalf("%s returned %d, want %d: %s", args[0], code, test.want, stderr.String())
				}
				if test.want == 0 && args[0] == "query" && !bytes.Contains(stdout.Bytes(), []byte(`"result": "granted"`)) {
					t.Fatal("snapshot query lost authorization")
				}
			}
		})
	}
}
