package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jintilm/GrantDiff/internal/diff"
)

func TestPinnedExternalDNSReview(t *testing.T) {
	root := "../../examples/external-dns"
	args := []string{"diff", "--before", root + "/before", "--after", root + "/after", "--catalog", root + "/catalog.yaml", "--format", "json", "--fail-on", "high"}
	var stdout, stderr bytes.Buffer
	if code := Run(args, &stdout, &stderr); code != 1 {
		t.Fatalf("expected high-risk exit 1, got %d: %s", code, stderr.String())
	}
	var result diff.Report
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Scope.Calculation != "complete" || len(result.Changes) == 0 {
		t.Fatalf("fixture incomplete: %+v", result)
	}
	verbs := map[string]bool{}
	for _, change := range result.Changes {
		if change.Type != "added" || change.Subject != "system:serviceaccount:default:external-dns" || change.Request.Resource != "secrets" || change.Request.APIGroup != "" || !slices.Contains([]string{"get", "list", "watch"}, change.Request.Verb) {
			t.Fatalf("unexpected permission change: %+v", change)
		}
		if len(change.Risk) != 1 || change.Risk[0].ID != "secret_read" {
			t.Fatal("missing Secret read warning")
		}
		for _, evidence := range change.AfterEvidence {
			if evidence.Binding != "ClusterRoleBinding//external-dns-viewer" || evidence.Role != "ClusterRole//external-dns" {
				t.Fatalf("unexpected evidence: %+v", evidence)
			}
		}
		verbs[change.Request.Verb] = true
	}
	if len(verbs) != 3 {
		t.Fatal("Secret read verbs were lost")
	}
	args = []string{"diff", "--before", root + "/before", "--after", root + "/after", "--catalog", root + "/catalog.yaml", "--format", "markdown"}
	stdout.Reset()
	if code := Run(args, &stdout, &stderr); code != 0 {
		t.Fatal(stderr.String())
	}
	expected, err := os.ReadFile(root + "/expected.md")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.ReplaceAll(expected, []byte("\r\n"), []byte("\n")), stdout.Bytes()) {
		t.Fatal("ExternalDNS Markdown differs from the reviewed expected report")
	}
}

func TestExternalDNSProvenance(t *testing.T) {
	root := "../../examples/external-dns"
	data, err := os.ReadFile(root + "/provenance.json")
	if err != nil {
		t.Fatal(err)
	}
	var provenance struct {
		SourceFiles []struct{ Path, SHA256 string } `json:"sourceFiles"`
	}
	if err := json.Unmarshal(data, &provenance); err != nil {
		t.Fatal(err)
	}
	for _, file := range provenance.SourceFiles {
		local := root + "/before/" + filepath.Base(file.Path)
		if file.Path == "LICENSE.md" {
			local = root + "/UPSTREAM-LICENSE.md"
		}
		content, err := os.ReadFile(local)
		if err != nil {
			t.Fatal(err)
		}
		content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
		digest := sha256.Sum256(content)
		if hex.EncodeToString(digest[:]) != file.SHA256 {
			t.Fatalf("upstream file %s differs from the pinned source", file.Path)
		}
	}
}

func TestCompactExternalDNSReviewAndExitCodes(t *testing.T) {
	root := "../../examples/external-dns"
	args := []string{"diff", "--before", root + "/before", "--after", root + "/after", "--catalog", root + "/catalog.yaml", "--format", "json", "--fail-on", "high", "--compact"}
	var stdout, stderr bytes.Buffer
	if code := Run(args, &stdout, &stderr); code != 1 {
		t.Fatalf("compacted high-risk review returned %d: %s", code, stderr.String())
	}
	var result diff.Report
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Changes) != 3 || result.Scope.Calculation != "complete" {
		t.Fatalf("unexpected compacted report: %+v", result)
	}
	verbs := []string{}
	for _, change := range result.Changes {
		if change.Type != "added" || change.Request.Resource != "secrets" || len(change.Risk) != 1 || change.Risk[0].ID != "secret_read" || len(change.AfterEvidence) != 1 {
			t.Fatalf("compact report lost meaning: %+v", change)
		}
		verbs = append(verbs, change.Request.Verb)
	}
	slices.Sort(verbs)
	if !slices.Equal(verbs, []string{"get", "list", "watch"}) {
		t.Fatal("compact report lost Secret verbs")
	}
	stdout.Reset()
	if code := Run(append(args, "--format", "markdown"), &stdout, &stderr); code != 1 {
		t.Fatalf("compact Markdown returned %d", code)
	}
	expected, err := os.ReadFile(root + "/expected-compact.md")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.ReplaceAll(expected, []byte("\r\n"), []byte("\n")), stdout.Bytes()) {
		t.Fatal("compact Markdown differs from the reviewed expected report")
	}
	stdout.Reset()
	if code := Run(append(args, "--format", "markdown", "--summary"), &stdout, &stderr); code != 1 || !bytes.Contains(stdout.Bytes(), []byte("High-risk ranges 3")) || bytes.Contains(stdout.Bytes(), []byte("<details>")) {
		t.Fatalf("compact summary changed CI failure: %d %s", code, stdout.String())
	}
	stdout.Reset()
	if code := Run(append(args, "--max-classes", "1"), &stdout, &stderr); code != 3 || !bytes.Contains(stdout.Bytes(), []byte("budget_exceeded")) {
		t.Fatalf("compact budget failure returned %d: %s", code, stdout.String())
	}
}
