package input

import (
	"strings"
	"testing"
)

const versionedCRD = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata: {name: widgets.example.test}
spec:
  group: example.test
  scope: Namespaced
  names: {plural: widgets, kind: Widget}
  versions:
    - name: v1
      served: true
      storage: true
      subresources: {status: {}}
    - name: v2
      served: true
      storage: false
      subresources: {scale: {specReplicasPath: .spec.replicas, statusReplicasPath: .status.replicas}}
    - name: v0
      served: false
      storage: false
`

func TestCRDCatalogIncludesServedSubresources(t *testing.T) {
	state, err := Parse([]File{{Name: "crd.yaml", Data: []byte(versionedCRD)}}, "test", Options{AllowEmpty: true})
	if err != nil {
		t.Fatal(err)
	}
	resource, exists := state.Catalog.Find("example.test", "widgets")
	if !exists || !resource.Namespaced || strings.Join(resource.Subresources, ",") != "scale,status" {
		t.Fatalf("unexpected resource catalog: %+v", state.Catalog)
	}
	data := strings.Replace(versionedCRD, "name: v2\n      served: true", "name: v2\n      served: false", 1)
	state, err = Parse([]File{{Name: "crd.yaml", Data: []byte(data)}}, "test", Options{AllowEmpty: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(state.Catalog.Resources[0].Subresources, ",") != "status" {
		t.Fatal("unserved version leaked a subresource")
	}
}

func TestMalformedCRDsAreRejected(t *testing.T) {
	tests := map[string]string{
		"scope":                 strings.Replace(versionedCRD, "scope: Namespaced", "scope: Guess", 1),
		"missing scope":         strings.Replace(versionedCRD, "  scope: Namespaced\n", "", 1),
		"missing served":        strings.Replace(versionedCRD, "      served: true\n", "", 1),
		"unknown subresource":   strings.Replace(versionedCRD, "status: {}", "log: {}", 1),
		"no served versions":    strings.ReplaceAll(versionedCRD, "served: true", "served: false"),
		"duplicate CRD":         versionedCRD + "---\n" + versionedCRD,
		"unsupported version":   strings.Replace(versionedCRD, "apiextensions.k8s.io/v1", "apiextensions.k8s.io/v1beta1", 1),
		"incorrect object name": strings.Replace(versionedCRD, "name: widgets.example.test", "name: widgets.other.test", 1),
	}
	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]File{{Name: "crd.yaml", Data: []byte(data)}}, "test", Options{AllowEmpty: true}); err == nil {
				t.Fatal("invalid CRD accepted")
			}
		})
	}
}
