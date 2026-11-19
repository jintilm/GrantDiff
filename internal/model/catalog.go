package model

import (
	"fmt"
	"slices"
	"strings"
)

// MergeCatalogs combines discovery, explicit and CRD-derived resource knowledge.
// Conflicting scopes cannot be resolved by choosing whichever input came last.
func MergeCatalogs(catalogs ...Catalog) (Catalog, error) {
	result := Catalog{SchemaVersion: "1.0", Resources: []Resource{}}
	resources := map[string]Resource{}
	for _, catalog := range catalogs {
		if catalog.KubernetesVersion != "" {
			if result.KubernetesVersion != "" && result.KubernetesVersion != catalog.KubernetesVersion {
				return result, fmt.Errorf("catalog Kubernetes versions conflict: %s and %s", result.KubernetesVersion, catalog.KubernetesVersion)
			}
			result.KubernetesVersion = catalog.KubernetesVersion
		}
		for _, resource := range catalog.Resources {
			key := resource.APIGroup + "/" + resource.Name
			if previous, ok := resources[key]; ok {
				if previous.Namespaced != resource.Namespaced {
					return result, fmt.Errorf("resource scope conflicts for %s", key)
				}
				resource.Subresources = append(slices.Clone(previous.Subresources), resource.Subresources...)
			}
			resource.Subresources = Unique(resource.Subresources)
			resources[key] = resource
		}
	}
	for _, resource := range resources {
		result.Resources = append(result.Resources, resource)
	}
	slices.SortFunc(result.Resources, func(a, b Resource) int { return strings.Compare(a.APIGroup+"/"+a.Name, b.APIGroup+"/"+b.Name) })
	return result, nil
}
