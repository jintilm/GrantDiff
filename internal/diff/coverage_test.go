package diff

import (
	"slices"
	"strings"
	"testing"

	"github.com/jintilm/GrantDiff/internal/evaluate"
	"github.com/jintilm/GrantDiff/internal/model"
	rbac "k8s.io/api/rbac/v1"
)

func contains(selector Selector, value string) bool {
	if selector.Value != nil {
		return value == *selector.Value
	}
	if slices.Contains(selector.Except, value) {
		return false
	}
	if selector.Prefix != nil && !strings.HasPrefix(value, *selector.Prefix) {
		return false
	}
	for _, prefix := range selector.ExceptPrefixes {
		if strings.HasPrefix(value, prefix) {
			return false
		}
	}
	return true
}

func TestPartitionPreservesRuleTruth(t *testing.T) {
	rules := []rbac.PolicyRule{
		{APIGroups: []string{"*"}, Resources: []string{"*"}, Verbs: []string{"*"}},
		{APIGroups: []string{""}, Resources: []string{"pods", "*/log"}, Verbs: []string{"get"}, ResourceNames: []string{"app"}},
		{APIGroups: []string{"apps"}, Resources: []string{"deployments/scale"}, Verbs: []string{"patch"}},
	}
	grants := []evaluate.Grant{}
	for _, rule := range rules {
		grants = append(grants, evaluate.Grant{Rule: rule})
	}
	state := model.NewState()
	state.Catalog = model.Catalog{Resources: []model.Resource{{Name: "pods", Namespaced: true, Subresources: []string{"log"}}, {APIGroup: "apps", Name: "deployments", Namespaced: true, Subresources: []string{"scale"}}}}
	type sample struct {
		request model.Request
		region  Region
	}
	partitions := []sample{}
	walkRequests(grants, state, state, func(request model.Request, region Region) bool {
		partitions = append(partitions, sample{request, region})
		return true
	})
	for _, group := range []string{"", "apps", "future.example"} {
		for _, resource := range []string{"pods", "deployments", "future"} {
			for _, sub := range []string{"", "log", "scale", "future"} {
				for _, name := range []string{"app", "other"} {
					for _, verb := range []string{"get", "patch", "list"} {
						request := model.Request{Verb: verb, APIGroup: group, Resource: resource, Subresource: sub, Namespace: "finance", Name: name}
						values := map[string]string{"verb": verb, "apiGroup": group, "resource": resource, "subresource": sub, "namespace": "finance", "name": name}
						matches := 0
						for _, partition := range partitions {
							in := true
							for dimension, value := range values {
								in = in && contains(partition.region[dimension], value)
							}
							if !in {
								continue
							}
							matches++
							for _, rule := range rules {
								if evaluate.Allows(rule, request) != evaluate.Allows(rule, partition.request) {
									t.Fatalf("rule truth changes within region: request=%+v witness=%+v", request, partition.request)
								}
							}
						}
						if matches != 1 {
							t.Fatalf("request maps to %d regions: %+v", matches, request)
						}
					}
				}
			}
		}
	}
}
