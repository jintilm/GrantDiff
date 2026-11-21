package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestOfflineViewerThroughCLI(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "权限 报告.json")
	output := filepath.Join(directory, "交互 报告.html")
	var stdout, stderr bytes.Buffer
	args := []string{"diff", "--before", "../../examples/external-dns/before", "--after", "../../examples/external-dns/after", "--catalog", "../../examples/external-dns/catalog.yaml", "--compact", "--format", "json", "--max-classes", "1", "--output", input}
	if code := Run(args, &stdout, &stderr); code != 3 {
		t.Fatalf("expected incomplete source report, got %d: %s", code, stderr.String())
	}
	for _, args := range [][]string{{"view", "--output", output}, {"view", "--input", input, "--output", output}} {
		if code := Run(args, &stdout, &stderr); code != 0 {
			t.Fatalf("viewer returned %d: %s", code, stderr.String())
		}
	}
	page, err := os.ReadFile(output)
	if err != nil || !bytes.Contains(page, []byte("budget_exceeded")) || !bytes.Contains(page, []byte("Content-Security-Policy")) {
		t.Fatalf("viewer lost incomplete report or CSP: %v", err)
	}
	for _, args := range [][]string{{"view"}, {"view", "--output", input}, {"view", "--input", input, "--output", output + ".txt"}} {
		if code := Run(args, &stdout, &stderr); code != 2 {
			t.Fatalf("invalid viewer arguments returned %d", code)
		}
	}
	if err := os.WriteFile(input, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if code := Run([]string{"view", "--input", input, "--output", output}, &stdout, &stderr); code != 2 {
		t.Fatal("wrong report accepted")
	}
	current, _ := os.ReadFile(output)
	if !bytes.Equal(page, current) {
		t.Fatal("invalid report replaced existing output")
	}
	if err := os.WriteFile(output, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if code := Run([]string{"view", "--input", output, "--output", output}, &stdout, &stderr); code != 2 {
		t.Fatal("viewer overwrote its input")
	}
}
