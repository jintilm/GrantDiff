package input

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jintilm/GrantDiff/internal/model"
	"sigs.k8s.io/yaml"
)

func Config(path string, value any) error {
	data, err := ReadLimited(path)
	if err != nil {
		return err
	}
	if err := yaml.UnmarshalStrict(data, value); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func LoadIdentities(path string) ([]model.Identity, error) {
	var config model.Identities
	if err := Config(path, &config); err != nil {
		return nil, err
	}
	if config.SchemaVersion != "1.0" {
		return nil, fmt.Errorf("unsupported identities schemaVersion %q", config.SchemaVersion)
	}
	ids, keys := map[string]bool{}, map[string]bool{}
	for _, identity := range config.Identities {
		if err := identity.Validate(); err != nil {
			return nil, err
		}
		if ids[identity.ID] || keys[identity.Key()] {
			return nil, fmt.Errorf("duplicate identity %q", identity.ID)
		}
		ids[identity.ID], keys[identity.Key()] = true, true
	}
	if len(config.Identities) == 0 {
		return nil, fmt.Errorf("identities file is empty")
	}
	return config.Identities, nil
}

func LoadCatalog(path string) (model.Catalog, error) {
	// Keep scope optional while decoding so omission cannot become false.
	var document struct {
		SchemaVersion     string `json:"schemaVersion"`
		KubernetesVersion string `json:"kubernetesVersion,omitempty"`
		Resources         []struct {
			APIGroup     string   `json:"apiGroup"`
			Name         string   `json:"name"`
			Namespaced   *bool    `json:"namespaced"`
			Subresources []string `json:"subresources,omitempty"`
		} `json:"resources"`
	}
	var catalog model.Catalog
	if err := Config(path, &document); err != nil {
		return catalog, err
	}
	if document.SchemaVersion != "1.0" {
		return catalog, fmt.Errorf("unsupported catalog schemaVersion")
	}
	catalog.SchemaVersion = document.SchemaVersion
	catalog.KubernetesVersion = document.KubernetesVersion
	seen := map[string]bool{}
	for _, resource := range document.Resources {
		key := resource.APIGroup + "/" + resource.Name
		if resource.Name == "" || seen[key] || strings.ContainsAny(resource.Name, "/*") {
			return catalog, fmt.Errorf("invalid or duplicate resource %q", key)
		}
		if resource.Namespaced == nil {
			return catalog, fmt.Errorf("resource %q requires an explicit namespaced boolean", key)
		}
		catalog.Resources = append(catalog.Resources, model.Resource{
			APIGroup: resource.APIGroup, Name: resource.Name,
			Namespaced: *resource.Namespaced, Subresources: resource.Subresources,
		})
		seen[key] = true
	}
	return catalog, nil
}

func InferIdentities(states ...*model.State) []model.Identity {
	identities := map[string]model.Identity{}
	for _, state := range states {
		for _, identity := range state.Identities {
			identities[identity.Key()] = identity
		}
		for _, object := range state.SortedObjects() {
			for _, subject := range object.Subjects {
				identity := model.Identity{Kind: subject.Kind, Name: subject.Name, Namespace: subject.Namespace, Authenticated: subject.Kind == "User" && subject.Name != "system:anonymous", GroupsComplete: subject.Kind != "User"}
				identity.ID = identity.Username()
				if identity.Kind == "Group" {
					identity.ID = "group:" + identity.Name
				}
				identities[identity.Key()] = identity
			}
		}
	}
	result := make([]model.Identity, 0, len(identities))
	for _, identity := range identities {
		result = append(result, identity)
	}
	slices.SortFunc(result, func(a, b model.Identity) int { return strings.Compare(a.Key(), b.Key()) })
	return result
}
