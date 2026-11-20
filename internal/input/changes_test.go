package input

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCandidateDeletionKeepsOtherObjects(t *testing.T) {
	before, err := Load("../../examples/retained/before", Options{})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "changes.json")
	changes := ChangeSet{SchemaVersion: "1.0", BaselineDigest: before.Digest(), Changes: []Operation{{Action: "delete", Key: "RoleBinding/finance/direct-read"}}}
	data, err := json.Marshal(changes)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	after, err := ApplyChanges(before, path, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Objects) != 2 || after.Objects["RoleBinding/finance/group-read"] == nil || len(before.Objects) != 3 {
		t.Fatal("candidate deletion changed unrelated objects or baseline")
	}
	changes.BaselineDigest = "stale"
	data, _ = json.Marshal(changes)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyChanges(before, path, Options{}); err == nil {
		t.Fatal("stale baseline was accepted")
	}
}

func TestCandidateDeletingAllObjectsRequiresExplicitEmptyState(t *testing.T) {
	before, err := Load("../../examples/retained/before", Options{})
	if err != nil {
		t.Fatal(err)
	}
	changes := ChangeSet{SchemaVersion: "1.0", BaselineDigest: before.Digest()}
	for _, object := range before.SortedObjects() {
		changes.Changes = append(changes.Changes, Operation{Action: "delete", Key: object.Key()})
	}
	data, err := json.Marshal(changes)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "delete-all.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyChanges(before, path, Options{}); err == nil {
		t.Fatal("empty candidate accepted without explicit flag")
	}
	after, err := ApplyChanges(before, path, Options{AllowEmpty: true})
	if err != nil || len(after.Objects) != 0 || len(before.Objects) != 3 {
		t.Fatalf("explicit deletion failed or mutated baseline: %v", err)
	}
}
