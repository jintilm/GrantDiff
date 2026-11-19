package diff

import (
	"slices"
	"strings"

	"github.com/jintilm/GrantDiff/internal/evaluate"
	"github.com/jintilm/GrantDiff/internal/model"
)

type cell struct {
	witness  string
	selector Selector
}

// Cells partition a string dimension into all named values and its complement.
func cells(values []string, wildcard bool) []cell {
	values = model.Unique(values)
	result := []cell{}
	for _, value := range values {
		value := value
		result = append(result, cell{value, Selector{Value: &value}})
	}
	if wildcard {
		witness := "grantdiff-other"
		for slices.Contains(values, witness) {
			witness += "-x"
		}
		result = append(result, cell{witness, Selector{Except: values}})
	}
	return result
}
func splitPatterns(values []string) ([]string, bool) {
	result := []string{}
	wildcard := false
	for _, value := range values {
		if value == "*" {
			wildcard = true
		} else {
			result = append(result, value)
		}
	}
	return result, wildcard
}

func walkRequests(grants []evaluate.Grant, before, after *model.State, visit func(model.Request, Region) bool) bool {
	groups, verbs, namespaces, names := []string{}, []string{}, []string{""}, []string{""}
	urls := []string{}
	for _, grant := range grants {
		groups = append(groups, grant.Rule.APIGroups...)
		verbs = append(verbs, grant.Rule.Verbs...)
		namespaces = append(namespaces, grant.Namespace)
		names = append(names, grant.Rule.ResourceNames...)
		urls = append(urls, grant.Rule.NonResourceURLs...)
	}
	groups, allGroups := splitPatterns(groups)
	verbs, allVerbs := splitPatterns(verbs)
	if allGroups {
		groups = append(groups, "", "rbac.authorization.k8s.io", "authentication.k8s.io")
		for _, state := range []*model.State{before, after} {
			for _, resource := range state.Catalog.Resources {
				groups = append(groups, resource.APIGroup)
			}
		}
	}
	if allVerbs {
		verbs = append(verbs, "get", "list", "watch", "create", "update", "patch", "delete", "deletecollection", "bind", "escalate", "impersonate")
	}
	for _, group := range cells(groups, allGroups) {
		resources := []string{}
		wildResource := false
		for _, grant := range grants {
			if !slices.Contains(grant.Rule.APIGroups, "*") && !slices.Contains(grant.Rule.APIGroups, group.witness) {
				continue
			}
			for _, resource := range grant.Rule.Resources {
				base, _, _ := strings.Cut(resource, "/")
				if base == "*" {
					wildResource = true
				} else {
					resources = append(resources, base)
				}
			}
		}
		if wildResource {
			for _, state := range []*model.State{before, after} {
				for _, resource := range state.Catalog.Resources {
					if resource.APIGroup == group.witness {
						resources = append(resources, resource.Name)
					}
				}
			}
			if group.witness == "" {
				resources = append(resources, "secrets", "users", "groups", "serviceaccounts")
			}
			if group.witness == "rbac.authorization.k8s.io" {
				resources = append(resources, "roles", "clusterroles")
			}
		}
		for _, resource := range cells(resources, wildResource) {
			subs := []string{""}
			allSubs := false
			for _, grant := range grants {
				if !slices.Contains(grant.Rule.APIGroups, "*") && !slices.Contains(grant.Rule.APIGroups, group.witness) {
					continue
				}
				for _, pattern := range grant.Rule.Resources {
					if pattern == "*" {
						allSubs = true
						continue
					}
					base, sub, hasSub := strings.Cut(pattern, "/")
					if hasSub && (base == resource.witness || base == "*") {
						subs = append(subs, sub)
					}
				}
			}
			if allSubs {
				for _, state := range []*model.State{before, after} {
					if known, ok := state.Catalog.Find(group.witness, resource.witness); ok {
						subs = append(subs, known.Subresources...)
					}
				}
			}
			for _, sub := range cells(subs, allSubs) {
				for _, verb := range cells(verbs, allVerbs) {
					for _, namespace := range cells(namespaces, true) {
						knownResource, known := after.Catalog.Find(group.witness, resource.witness)
						if !known {
							knownResource, known = before.Catalog.Find(group.witness, resource.witness)
						}
						if known && !knownResource.Namespaced && namespace.witness != "" {
							continue
						}
						if known && knownResource.Namespaced && namespace.witness == "" && !slices.Contains([]string{"list", "watch"}, verb.witness) {
							continue
						}
						for _, name := range cells(names, true) {
							if sub.witness == "" && slices.Contains([]string{"create", "deletecollection"}, verb.witness) && name.witness != "" {
								continue
							}
							if slices.Contains([]string{"get", "update", "patch", "delete"}, verb.witness) && name.witness == "" {
								continue
							}
							request := model.Request{Verb: verb.witness, APIGroup: group.witness, Resource: resource.witness, Subresource: sub.witness, Namespace: namespace.witness, Name: name.witness}
							region := Region{"verb": verb.selector, "apiGroup": group.selector, "resource": resource.selector, "subresource": sub.selector, "namespace": namespace.selector, "name": name.selector}
							if !visit(request, region) {
								return false
							}
						}
					}
				}
			}
		}
	}
	for _, url := range urlCells(urls) {
		for _, verb := range cells(verbs, allVerbs) {
			if !visit(model.Request{Verb: verb.witness, URL: url.witness}, Region{"verb": verb.selector, "url": url.selector}) {
				return false
			}
		}
	}
	return true
}

// A prefix cell excludes each nested prefix and exact URL, so each rule is
// constant throughout the cell and comparing its witness is sufficient.
func urlCells(patterns []string) []cell {
	exacts, prefixes := []string{}, []string{}
	for _, pattern := range model.Unique(patterns) {
		if strings.HasSuffix(pattern, "*") {
			prefixes = append(prefixes, strings.TrimRight(pattern, "*"))
		} else {
			exacts = append(exacts, pattern)
		}
	}
	result := cells(exacts, false)
	for _, prefix := range model.Unique(prefixes) {
		excluded := []string{}
		for _, nested := range prefixes {
			if nested != prefix && strings.HasPrefix(nested, prefix) {
				excluded = append(excluded, nested)
			}
		}
		root := prefix
		if root == "" {
			root = "/"
		}
		coveredRoot := false
		for _, nested := range excluded {
			coveredRoot = coveredRoot || strings.HasPrefix(root, nested)
		}
		if coveredRoot {
			continue
		}
		witness := root
		if slices.Contains(exacts, witness) {
			// A finite set of stricter prefixes cannot exhaust all Unicode next
			// characters. Pick a branch outside every child prefix.
			for next := 'a'; ; next++ {
				candidate := root + string(next) + "grantdiff"
				covered := slices.Contains(exacts, candidate)
				for _, nested := range excluded {
					covered = covered || strings.HasPrefix(candidate, nested)
				}
				if !covered {
					witness = candidate
					break
				}
			}
		}
		prefix := prefix
		result = append(result, cell{witness, Selector{Prefix: &prefix, Except: exacts, ExceptPrefixes: excluded}})
	}
	return result
}
