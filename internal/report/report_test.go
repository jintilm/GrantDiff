package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jintilm/GrantDiff/internal/diff"
	"github.com/jintilm/GrantDiff/internal/model"
)

func TestReadableSummaryKeepsAllChangesAndRiskCounts(t *testing.T) {
	r := diff.Report{Scope: diff.Scope{Calculation: "incomplete"}, Identities: 3, Compared: 10,
		Changes: []diff.Change{
			{ID: "a", Subject: "alice", TargetType: "identity_access", Type: "added", AfterEvidence: []model.Evidence{{Binding: "RoleBinding/apps/access", Role: "Role/apps/reader"}}, Risk: []diff.Risk{{ID: "secret_read", Severity: "high"}}},
			{ID: "b", Subject: "alice", TargetType: "identity_access", Type: "removed"},
			{ID: "c", Subject: "reviewers", TargetType: "group_grant", Type: "indeterminate"},
		}, Diagnostics: []model.Diagnostic{{Code: "missing_role", Message: "reader is missing"}},
	}
	full, err := Render(r, "markdown")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(full, []byte("<details>")) != 3 || !bytes.Contains(full, []byte("High-risk ranges 1")) || !bytes.Contains(full, []byte("Indeterminate 1")) {
		t.Fatalf("summary dropped changes: %s", full)
	}
	summary, err := RenderSummary(r, "markdown")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(summary, []byte("<details>")) || !bytes.Contains(summary, []byte("missing_role")) || !bytes.Contains(summary, []byte("RoleBinding/apps/access")) {
		t.Fatalf("summary omitted diagnostic or attribution: %s", summary)
	}
	encoded, err := Render(r, "json")
	if err != nil {
		t.Fatal(err)
	}
	var decoded diff.Report
	if err := json.Unmarshal(encoded, &decoded); err != nil || len(decoded.Changes) != 3 {
		t.Fatal("JSON changes lost")
	}
}

func TestMarkdownEscapesUntrustedNames(t *testing.T) {
	r := diff.Report{Scope: diff.Scope{Calculation: "complete"}, Changes: []diff.Change{{ID: "a", Subject: "<img src=x>|[click](https://example.test)\nsecond", Type: "added", TargetType: "identity_access"}}}
	data, err := Render(r, "markdown")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, "<summary>added · <img") || strings.Contains(text, "| <img") || strings.Contains(text, "\nsecond") {
		t.Fatalf("unescaped name altered Markdown structure: %s", text)
	}
	if !strings.Contains(text, "&lt;img") || !strings.Contains(text, "\\|") || !strings.Contains(text, "\\[click\\]") {
		t.Fatal("summary name was not escaped")
	}
}
