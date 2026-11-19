package model

import "testing"

func TestCatalogMergeRejectsScopeConflict(t *testing.T) {
	base := Catalog{Resources: []Resource{{APIGroup: "example.test", Name: "widgets", Namespaced: true}}}
	other := Catalog{Resources: []Resource{{APIGroup: "example.test", Name: "widgets", Namespaced: false}}}
	if _, err := MergeCatalogs(base, other); err == nil {
		t.Fatal("catalog silently overwrote scope")
	}
}

func TestCatalogMergeUnionsSubresourcesWithoutMutation(t *testing.T) {
	base := Catalog{Resources: []Resource{{APIGroup: "example.test", Name: "widgets", Namespaced: true, Subresources: []string{"status"}}}}
	other := Catalog{Resources: []Resource{{APIGroup: "example.test", Name: "widgets", Namespaced: true, Subresources: []string{"scale", "status"}}}}
	merged, err := MergeCatalogs(base, other)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Resources) != 1 || len(merged.Resources[0].Subresources) != 2 || len(base.Resources[0].Subresources) != 1 {
		t.Fatal("catalog merge lost information or mutated its input")
	}
}
