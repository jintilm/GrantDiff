package input

import (
	"encoding/json"
	"fmt"
	"github.com/jintilm/GrantDiff/internal/model"
)

type ChangeSet struct {
	SchemaVersion  string      `json:"schemaVersion"`
	BaselineDigest string      `json:"baselineDigest"`
	Changes        []Operation `json:"changes"`
}
type Operation struct {
	Action       string          `json:"action"`
	Key          string          `json:"key"`
	BeforeDigest string          `json:"beforeDigest,omitempty"`
	Object       json.RawMessage `json:"object,omitempty"`
}

func ApplyChanges(before *model.State, path string, opts Options) (*model.State, error) {
	var changes ChangeSet
	if err := Config(path, &changes); err != nil {
		return nil, err
	}
	if changes.SchemaVersion != "1.0" {
		return nil, fmt.Errorf("unsupported changes schema")
	}
	if changes.BaselineDigest == "" || changes.BaselineDigest != before.Digest() {
		return nil, fmt.Errorf("baseline digest is missing or does not match")
	}
	after := model.NewState()
	after.Catalog = before.Catalog
	after.Identities = before.Identities
	after.Namespaces = before.Namespaces
	after.Origin = "candidate:" + path
	for key, object := range before.Objects {
		after.Objects[key] = object
	}
	seen := map[string]bool{}
	for index, change := range changes.Changes {
		if seen[change.Key] {
			return nil, fmt.Errorf("duplicate operation for %s", change.Key)
		}
		seen[change.Key] = true
		previous, exists := before.Objects[change.Key]
		if change.BeforeDigest != "" && (!exists || previous.Digest() != change.BeforeDigest) {
			return nil, fmt.Errorf("object digest mismatch: %s", change.Key)
		}
		switch change.Action {
		case "delete":
			if !exists || len(change.Object) > 0 {
				return nil, fmt.Errorf("invalid delete of %s", change.Key)
			}
			delete(after.Objects, change.Key)
		case "upsert":
			parsed, err := Parse([]File{{Name: fmt.Sprintf("changes/%d.json", index), Data: change.Object}}, path, opts)
			if err != nil {
				return nil, err
			}
			object, ok := parsed.Objects[change.Key]
			if !ok || len(parsed.Objects) != 1 {
				return nil, fmt.Errorf("operation key must match exactly one object: %s", change.Key)
			}
			after.Objects[change.Key] = object
		default:
			return nil, fmt.Errorf("unsupported operation %q", change.Action)
		}
	}
	return after, nil
}
