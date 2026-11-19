package input

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCatalogRequiresExplicitResourceScope(t *testing.T) {
	for _, scope := range []string{"", "    namespaced: null\n", "    namespaced: 'false'\n"} {
		t.Run(scope, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "catalog.yaml")
			data := "schemaVersion: '1.0'\nresources:\n  - apiGroup: ''\n    name: configmaps\n" + scope
			if err := os.WriteFile(file, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadCatalog(file); err == nil {
				t.Fatal("catalog with missing or invalid resource scope was accepted")
			}
		})
	}
}

func TestCatalogPreservesExplicitScopes(t *testing.T) {
	file := filepath.Join(t.TempDir(), "catalog.yaml")
	data := "schemaVersion: '1.0'\nresources:\n  - apiGroup: ''\n    name: configmaps\n    namespaced: true\n  - apiGroup: ''\n    name: nodes\n    namespaced: false\n"
	if err := os.WriteFile(file, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	catalog, err := LoadCatalog(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Resources) != 2 || !catalog.Resources[0].Namespaced || catalog.Resources[1].Namespaced {
		t.Fatalf("resource scope changed: %+v", catalog)
	}
}
