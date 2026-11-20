package input

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestRelativeInputFromWindowsShortPath(t *testing.T) {
	directory := t.TempDir()
	manifests := filepath.Join(directory, "manifest directory")
	if err := os.Mkdir(manifests, 0700); err != nil {
		t.Fatal(err)
	}
	data := []byte("apiVersion: rbac.authorization.k8s.io/v1\nkind: Role\nmetadata: {name: reader, namespace: finance}\nrules: []\n")
	if err := os.WriteFile(filepath.Join(manifests, "policy.yaml"), data, 0600); err != nil {
		t.Fatal(err)
	}
	path, err := syscall.UTF16PtrFromString(directory)
	if err != nil {
		t.Fatal(err)
	}
	buffer := make([]uint16, 32768)
	size, err := syscall.GetShortPathName(path, &buffer[0], uint32(len(buffer)))
	if err != nil || size >= uint32(len(buffer)) {
		t.Fatalf("short path lookup failed: %v", err)
	}
	short := syscall.UTF16ToString(buffer[:size])
	if strings.EqualFold(short, directory) {
		t.Skip("filesystem does not provide a distinct short path")
	}
	t.Chdir(short)
	state, err := Load("manifest directory", Options{})
	if err != nil {
		t.Fatal(err)
	}
	object := state.Objects["Role/finance/reader"]
	if object == nil || object.Source.File != "policy.yaml" {
		t.Fatal("short working directory changed policy or source path")
	}
}
