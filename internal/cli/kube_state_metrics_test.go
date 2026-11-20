package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/jintilm/GrantDiff/internal/diff"
)

func TestPinnedKubeStateMetricsReview(t *testing.T) {
	root := "../../examples/kube-state-metrics"
	args := []string{"diff", "--before", root + "/before", "--after", root + "/after", "--catalog", root + "/catalog.json", "--compact", "--fail-on", "removed", "--format", "json"}
	var stdout, stderr bytes.Buffer
	if code := Run(args, &stdout, &stderr); code != 1 {
		t.Fatalf("expected removed-access exit 1, got %d: %s", code, stderr.String())
	}
	var review diff.Report
	if err := json.Unmarshal(stdout.Bytes(), &review); err != nil {
		t.Fatal(err)
	}
	removed, paths := []string{}, 0
	for _, change := range review.Changes {
		if change.Subject != "system:serviceaccount:kube-system:kube-state-metrics" || len(change.BeforeEvidence) != 1 {
			t.Fatalf("unexpected subject or evidence: %+v", change)
		}
		if change.BeforeEvidence[0].Binding != "ClusterRoleBinding//kube-state-metrics" || change.BeforeEvidence[0].Role != "ClusterRole//kube-state-metrics" {
			t.Fatal("wrong upstream authorization path")
		}
		switch change.Type {
		case "removed":
			if change.Request.APIGroup != "" || change.Request.Resource != "secrets" || change.After != "not_granted" {
				t.Fatalf("unexpected revocation: %+v", change)
			}
			removed = append(removed, change.Request.Verb)
		case "path_changed":
			if change.Before != "granted" || change.After != "granted" || len(change.AfterEvidence) != 1 {
				t.Fatal("remaining grant lost its evidence")
			}
			paths++
		default:
			t.Fatalf("unexpected change: %+v", change)
		}
	}
	slices.Sort(removed)
	if review.Scope.Calculation != "complete" || !slices.Equal(removed, []string{"list", "watch"}) || paths != 24 {
		t.Fatalf("unexpected report: removed=%v paths=%d scope=%+v", removed, paths, review.Scope)
	}
	stdout.Reset()
	if code := Run(append(args, "--format", "markdown"), &stdout, &stderr); code != 1 {
		t.Fatal(stderr.String())
	}
	expected, err := os.ReadFile(root + "/expected-compact.md")
	if err != nil || !bytes.Equal(bytes.ReplaceAll(expected, []byte("\r\n"), []byte("\n")), stdout.Bytes()) {
		t.Fatalf("expected report differs: %v", err)
	}
	data, err := os.ReadFile(root + "/provenance.json")
	if err != nil {
		t.Fatal(err)
	}
	var provenance struct {
		SourceFiles []struct{ Local, SHA256 string } `json:"sourceFiles"`
	}
	if err := json.Unmarshal(data, &provenance); err != nil {
		t.Fatal(err)
	}
	if len(provenance.SourceFiles) != 4 {
		t.Fatal("missing pinned source files")
	}
	for _, file := range provenance.SourceFiles {
		content, err := os.ReadFile(root + "/" + file.Local)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n")))
		if hex.EncodeToString(digest[:]) != file.SHA256 {
			t.Fatalf("pinned source differs: %s", file.Local)
		}
	}
}
