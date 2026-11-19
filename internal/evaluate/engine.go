package evaluate

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jintilm/GrantDiff/internal/model"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type resolvedRule struct {
	Rule   rbac.PolicyRule
	Role   string
	Path   []string
	Source model.Source
}
type resolvedRole struct {
	Rules       []resolvedRule
	Diagnostics []model.Diagnostic
}
type Engine struct {
	State    *model.State
	bindings map[string][]*model.Object
	roles    map[string]resolvedRole
	visiting map[string]bool
}
type Grant struct {
	Rule      rbac.PolicyRule
	Namespace string
	Evidence  model.Evidence
}
type unresolved struct {
	Namespace  string
	Diagnostic model.Diagnostic
}
type Principal struct {
	Identity model.Identity
	Grants   []Grant
	missing  []unresolved
}

func New(state *model.State) *Engine {
	e := &Engine{State: state, bindings: map[string][]*model.Object{}, roles: map[string]resolvedRole{}, visiting: map[string]bool{}}
	for _, object := range state.SortedObjects() {
		if object.Kind == "RoleBinding" || object.Kind == "ClusterRoleBinding" {
			for _, subject := range object.Subjects {
				key := "user:" + subject.Name
				switch subject.Kind {
				case "Group":
					key = "group:" + subject.Name
				case "ServiceAccount":
					key = "user:system:serviceaccount:" + subject.Namespace + ":" + subject.Name
				}
				e.bindings[key] = append(e.bindings[key], object)
			}
		}
	}
	return e
}

func (e *Engine) resolve(key string) resolvedRole {
	if role, ok := e.roles[key]; ok {
		return role
	}
	object, ok := e.State.Objects[key]
	if !ok {
		return resolvedRole{Diagnostics: []model.Diagnostic{{Code: "missing_role", Message: "referenced role is absent", Object: key}}}
	}
	if e.visiting[key] {
		return resolvedRole{Diagnostics: []model.Diagnostic{{Code: "aggregation_cycle", Message: "aggregation dependency cycle at " + key, Object: key}}}
	}
	e.visiting[key] = true
	defer delete(e.visiting, key)
	result := resolvedRole{}
	if object.AggregationRule == nil {
		for index, rule := range object.Rules {
			source := object.Source
			if index < len(object.RuleSources) {
				source = object.RuleSources[index]
			}
			result.Rules = append(result.Rules, resolvedRule{Rule: rule, Role: key, Source: source})
		}
	} else {
		selected := map[string]bool{}
		for _, selector := range object.AggregationRule.ClusterRoleSelectors {
			matcher, err := metav1.LabelSelectorAsSelector(&selector)
			if err != nil {
				result.Diagnostics = append(result.Diagnostics, model.Diagnostic{Code: "invalid_selector", Message: err.Error(), Object: key})
				continue
			}
			for _, candidate := range e.State.SortedObjects() {
				candidateKey := candidate.Key()
				if candidate.Kind != "ClusterRole" || candidateKey == key || selected[candidateKey] || !matcher.Matches(labels.Set(candidate.Metadata.Labels)) {
					continue
				}
				selected[candidateKey] = true
				child := e.resolve(candidateKey)
				result.Diagnostics = append(result.Diagnostics, child.Diagnostics...)
				for _, rule := range child.Rules {
					rule.Path = append([]string{key}, rule.Path...)
					result.Rules = append(result.Rules, rule)
				}
			}
		}
	}
	// An uncertain aggregate cannot supply authoritative rules of its own.
	if len(result.Diagnostics) > 0 {
		result.Rules = nil
	}
	e.roles[key] = result
	return result
}

func (e *Engine) Prepare(identity model.Identity) *Principal {
	p := &Principal{Identity: identity}
	groups := identity.EffectiveGroups()
	keys := []string{}
	if identity.Kind != "Group" {
		keys = append(keys, "user:"+identity.Username())
	}
	for _, group := range groups {
		keys = append(keys, "group:"+group)
	}
	candidates := map[string]*model.Object{}
	for _, key := range keys {
		for _, binding := range e.bindings[key] {
			candidates[binding.Key()] = binding
		}
	}
	bindingKeys := []string{}
	for key := range candidates {
		bindingKeys = append(bindingKeys, key)
	}
	slices.Sort(bindingKeys)
	for _, key := range bindingKeys {
		binding := candidates[key]
		matched := []string{}
		for _, subject := range binding.Subjects {
			switch subject.Kind {
			case "User":
				if identity.Kind != "Group" && subject.Name == identity.Username() {
					matched = append(matched, "User/"+subject.Name)
				}
			case "ServiceAccount":
				name := "system:serviceaccount:" + subject.Namespace + ":" + subject.Name
				if identity.Kind != "Group" && name == identity.Username() {
					matched = append(matched, "ServiceAccount/"+subject.Namespace+"/"+subject.Name)
				}
			case "Group":
				if slices.Contains(groups, subject.Name) {
					matched = append(matched, "Group/"+subject.Name)
				}
			}
		}
		if len(matched) == 0 {
			continue
		}
		namespace := ""
		if binding.RoleRef.Kind == "Role" {
			namespace = binding.Metadata.Namespace
		}
		role := e.resolve(model.Key(binding.RoleRef.Kind, namespace, binding.RoleRef.Name))
		for _, diagnostic := range role.Diagnostics {
			p.missing = append(p.missing, unresolved{binding.Metadata.Namespace, diagnostic})
		}
		for _, subject := range model.Unique(matched) {
			for _, rule := range role.Rules {
				evidence := model.Evidence{Subject: subject, Binding: binding.Key(), Role: rule.Role, RuleID: model.RuleID(rule.Rule), AggregationPath: rule.Path, BindingSource: binding.Source, RuleSource: rule.Source}
				p.Grants = append(p.Grants, Grant{Rule: rule.Rule, Namespace: binding.Metadata.Namespace, Evidence: evidence})
			}
		}
	}
	slices.SortFunc(p.Grants, func(a, b Grant) int { return strings.Compare(a.Evidence.Key(), b.Evidence.Key()) })
	return p
}

func (p *Principal) Diagnostics() []model.Diagnostic {
	diagnostics := []model.Diagnostic{}
	for _, missing := range p.missing {
		diagnostics = append(diagnostics, missing.Diagnostic)
	}
	if !p.Identity.Complete() {
		diagnostics = append(diagnostics, model.Diagnostic{Code: "incomplete_groups", Message: "group membership is incomplete for " + p.Identity.Username()})
	}
	return diagnostics
}
func (p *Principal) Signature() string {
	keys := []string{}
	for _, grant := range p.Grants {
		keys = append(keys, grant.Namespace+"|"+grant.Evidence.Key())
	}
	return model.Hash(struct {
		Keys        []string
		Diagnostics []model.Diagnostic
	}{keys, p.Diagnostics()})
}

func (p *Principal) Decide(request model.Request) model.Decision {
	decision := model.Decision{Result: "not_granted", Evidence: []model.Evidence{}, Diagnostics: []model.Diagnostic{}}
	seen := map[string]bool{}
	for _, grant := range p.Grants {
		if grant.Namespace != "" && (request.URL != "" || grant.Namespace != request.Namespace) {
			continue
		}
		if Allows(grant.Rule, request) && !seen[grant.Evidence.Key()] {
			decision.Evidence = append(decision.Evidence, grant.Evidence)
			seen[grant.Evidence.Key()] = true
		}
	}
	for _, missing := range p.missing {
		if missing.Namespace == "" || (request.URL == "" && missing.Namespace == request.Namespace) {
			decision.Diagnostics = append(decision.Diagnostics, missing.Diagnostic)
		}
	}
	if !p.Identity.Complete() {
		decision.Diagnostics = append(decision.Diagnostics, model.Diagnostic{Code: "incomplete_groups", Message: "identity may have additional group grants"})
	}
	if len(decision.Evidence) > 0 {
		decision.Result = "granted"
	} else if len(decision.Diagnostics) > 0 {
		decision.Result = "unknown"
	}
	return decision
}

func Allows(rule rbac.PolicyRule, request model.Request) bool {
	if !matches(rule.Verbs, request.Verb) {
		return false
	}
	if request.URL != "" {
		for _, pattern := range rule.NonResourceURLs {
			if pattern == request.URL || (strings.HasSuffix(pattern, "*") && strings.HasPrefix(request.URL, strings.TrimRight(pattern, "*"))) {
				return true
			}
		}
		return false
	}
	if !matches(rule.APIGroups, request.APIGroup) {
		return false
	}
	resourceMatches := matches(rule.Resources, request.ResourcePath())
	if request.Subresource != "" && slices.Contains(rule.Resources, "*/"+request.Subresource) {
		resourceMatches = true
	}
	return resourceMatches && (len(rule.ResourceNames) == 0 || slices.Contains(rule.ResourceNames, request.Name))
}
func matches(patterns []string, value string) bool {
	return slices.Contains(patterns, "*") || slices.Contains(patterns, value)
}

func CheckRequest(request model.Request, catalog model.Catalog) error {
	if request.Verb == "" {
		return fmt.Errorf("verb is required")
	}
	if request.URL != "" {
		if !strings.HasPrefix(request.URL, "/") || request.APIGroup != "" || request.Resource != "" || request.Namespace != "" || request.Name != "" || request.Subresource != "" {
			return fmt.Errorf("URL requests require an absolute URL path and no resource attributes")
		}
		return nil
	}
	if request.Resource == "" {
		return fmt.Errorf("resource or URL is required")
	}
	resource, known := catalog.Find(request.APIGroup, request.Resource)
	if !known {
		return fmt.Errorf("resource %s/%s is absent from catalog", request.APIGroup, request.Resource)
	}
	if !resource.Namespaced && request.Namespace != "" {
		return fmt.Errorf("cluster-scoped resource %q cannot have namespace", request.Resource)
	}
	if resource.Namespaced && request.Namespace == "" && !slices.Contains([]string{"list", "watch"}, request.Verb) {
		return fmt.Errorf("resource %q requires namespace for %s", request.Resource, request.Verb)
	}
	if request.Subresource != "" {
		if !slices.Contains(resource.Subresources, request.Subresource) {
			return fmt.Errorf("subresource %s/%s is absent from catalog", request.Resource, request.Subresource)
		}
		if request.Name == "" {
			return fmt.Errorf("subresource requests require the parent object's name")
		}
		if resource.Namespaced && request.Namespace == "" {
			return fmt.Errorf("namespaced subresource requests require namespace")
		}
	}
	if slices.Contains([]string{"get", "update", "patch", "delete"}, request.Verb) && request.Name == "" {
		return fmt.Errorf("%s requests require an object name", request.Verb)
	}
	if request.Subresource == "" && slices.Contains([]string{"create", "deletecollection"}, request.Verb) && request.Name != "" {
		return fmt.Errorf("%s on a top-level resource has no authorization name", request.Verb)
	}
	return nil
}
