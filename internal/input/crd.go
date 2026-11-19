package input

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/jintilm/GrantDiff/internal/model"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/yaml"
)

type customResourceDefinition struct {
	APIVersion string         `json:"apiVersion"`
	Metadata   model.Metadata `json:"metadata"`
	Spec       struct {
		Group string `json:"group"`
		Scope string `json:"scope"`
		Names struct {
			Plural string `json:"plural"`
		} `json:"names"`
		Versions []struct {
			Name         string                     `json:"name"`
			Served       *bool                      `json:"served"`
			Subresources map[string]json.RawMessage `json:"subresources"`
		} `json:"versions"`
	} `json:"spec"`
}

func addCRD(state *model.State, data []byte, source model.Source) error {
	var crd customResourceDefinition
	if err := yaml.Unmarshal(data, &crd); err != nil {
		return fmt.Errorf("%s:%d: %w", source.File, source.Line, err)
	}
	if crd.APIVersion != "apiextensions.k8s.io/v1" {
		return fmt.Errorf("%s:%d: unsupported CRD apiVersion %q", source.File, source.Line, crd.APIVersion)
	}
	resource := model.Resource{APIGroup: crd.Spec.Group, Name: crd.Spec.Names.Plural, Namespaced: crd.Spec.Scope == "Namespaced"}
	if resource.APIGroup == "" || len(validation.IsDNS1123Subdomain(resource.APIGroup)) != 0 || len(validation.IsDNS1035Label(resource.Name)) != 0 {
		return fmt.Errorf("%s:%d: CRD requires a valid spec.group and spec.names.plural", source.File, source.Line)
	}
	if crd.Metadata.Name != resource.Name+"."+resource.APIGroup || crd.Metadata.Namespace != "" {
		return fmt.Errorf("%s:%d: CRD metadata must name %s.%s at cluster scope", source.File, source.Line, resource.Name, resource.APIGroup)
	}
	if crd.Spec.Scope != "Namespaced" && crd.Spec.Scope != "Cluster" {
		return fmt.Errorf("CRD %s requires Namespaced or Cluster scope", crd.Metadata.Name)
	}
	if _, exists := state.Catalog.Find(resource.APIGroup, resource.Name); exists {
		return fmt.Errorf("duplicate CRD resource %s/%s", resource.APIGroup, resource.Name)
	}
	served := false
	versions := map[string]bool{}
	for _, version := range crd.Spec.Versions {
		if version.Name == "" || len(validation.IsDNS1035Label(version.Name)) != 0 || versions[version.Name] || version.Served == nil {
			return fmt.Errorf("CRD %s requires unique version names and explicit served booleans", crd.Metadata.Name)
		}
		versions[version.Name] = true
		for subresource, definition := range version.Subresources {
			if !slices.Contains([]string{"status", "scale"}, subresource) {
				return fmt.Errorf("CRD %s has unsupported subresource %q", crd.Metadata.Name, subresource)
			}
			definition = bytes.TrimSpace(definition)
			if bytes.Equal(definition, []byte("null")) {
				continue
			}
			if len(definition) == 0 || definition[0] != '{' {
				return fmt.Errorf("CRD %s subresource %s must be an object", crd.Metadata.Name, subresource)
			}
			if *version.Served {
				resource.Subresources = append(resource.Subresources, subresource)
			}
		}
		served = served || *version.Served
	}
	if !served {
		return fmt.Errorf("CRD %s requires at least one served version", crd.Metadata.Name)
	}
	catalog := model.Catalog{SchemaVersion: "1.0", Resources: []model.Resource{resource}}
	merged, err := model.MergeCatalogs(state.Catalog, catalog)
	if err != nil {
		return err
	}
	state.Catalog = merged
	return nil
}
