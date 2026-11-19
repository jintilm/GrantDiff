package input

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitReadsCompleteTreesWithoutCheckout(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git unavailable")
	}
	root := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=RBAC Test", "-c", "user.email=test@example.invalid"}, args...)...)
		data, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, data)
		}
		return strings.TrimSpace(string(data))
	}
	run("init")
	directory := filepath.Join(root, "policies")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	role := `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata: {name: reader, namespace: finance}
rules: [{apiGroups: [""], resources: [pods], verbs: [get]}]
`
	binding := `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata: {name: access, namespace: finance}
subjects: [{kind: ServiceAccount, name: bot, namespace: finance}]
roleRef: {kind: Role, name: reader, apiGroup: rbac.authorization.k8s.io}
`
	if err := os.WriteFile(filepath.Join(directory, "role.yaml"), []byte(role), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "binding.yaml"), []byte(binding), 0600); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "base")
	base := run("rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(directory, "binding.yaml"), []byte(strings.Replace(binding, "name: bot", "name: another", 1)), 0600); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "change")
	head := run("rev-parse", "HEAD")
	dirty := []byte("local uncommitted content\n")
	if err := os.WriteFile(filepath.Join(directory, "role.yaml"), dirty, 0600); err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{base, head} {
		state, err := Git(root, ref, "policies", Options{})
		if err != nil {
			t.Fatal(err)
		}
		if len(state.Objects) != 2 || state.Objects["Role/finance/reader"] == nil {
			t.Fatal("unchanged role missing from Git state")
		}
	}
	actual, err := os.ReadFile(filepath.Join(directory, "role.yaml"))
	if err != nil || string(actual) != string(dirty) {
		t.Fatal("Git review modified working tree")
	}
}
