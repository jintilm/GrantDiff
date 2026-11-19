package diff

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jintilm/GrantDiff/internal/evaluate"
	"github.com/jintilm/GrantDiff/internal/model"
)

type Selector struct {
	Value          *string  `json:"value,omitempty"`
	Except         []string `json:"except,omitempty"`
	Prefix         *string  `json:"prefix,omitempty"`
	ExceptPrefixes []string `json:"exceptPrefixes,omitempty"`
}
type Region map[string]Selector
type Change struct {
	ID             string           `json:"id"`
	TargetType     string           `json:"targetType"`
	Type           string           `json:"type"`
	Subject        string           `json:"subject"`
	Request        model.Request    `json:"representativeRequest"`
	Range          Region           `json:"range"`
	Before         string           `json:"before"`
	After          string           `json:"after"`
	BeforeEvidence []model.Evidence `json:"beforeEvidence"`
	AfterEvidence  []model.Evidence `json:"afterEvidence"`
	Risk           []Risk           `json:"risk"`
}
type Risk struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}
type Scope struct {
	Source               string `json:"source"`
	BeforeSource         string `json:"beforeSource"`
	AfterSource          string `json:"afterSource"`
	BeforeDigest         string `json:"beforeDigest"`
	AfterDigest          string `json:"afterDigest"`
	PolicyCompleteness   string `json:"policyCompleteness"`
	IdentityCompleteness string `json:"identityCompleteness"`
	Calculation          string `json:"calculation"`
}
type Report struct {
	SchemaVersion      string             `json:"schemaVersion"`
	AuthorizationModel string             `json:"authorizationModel"`
	Scope              Scope              `json:"scope"`
	Changes            []Change           `json:"changes"`
	Diagnostics        []model.Diagnostic `json:"diagnostics"`
	Compared           int                `json:"comparedClasses"`
	Identities         int                `json:"identities"`
}

func Compare(before, after *model.State, identities []model.Identity, budget int) Report {
	report := Report{SchemaVersion: "1.0", AuthorizationModel: "kubernetes-rbac", Scope: Scope{Source: "supplied-manifests", BeforeDigest: before.Digest(), AfterDigest: after.Digest(), PolicyCompleteness: "declared_complete", IdentityCompleteness: "complete", Calculation: "complete"}, Changes: []Change{}, Diagnostics: []model.Diagnostic{}, Identities: len(identities)}
	report.Scope.BeforeSource = strings.ReplaceAll(before.Origin, "\\", "/")
	report.Scope.AfterSource = strings.ReplaceAll(after.Origin, "\\", "/")
	if strings.HasPrefix(before.Origin, "git:") {
		report.Scope.Source = "git"
	}
	if strings.HasPrefix(before.Origin, "snapshot:") || before.Origin == "cluster-snapshot" {
		report.Scope.Source = "snapshot"
	}
	left, right := evaluate.New(before), evaluate.New(after)
	report.Diagnostics = append(report.Diagnostics, before.Diagnostics...)
	report.Diagnostics = append(report.Diagnostics, after.Diagnostics...)
	for _, resource := range before.Catalog.Resources {
		if candidate, ok := after.Catalog.Find(resource.APIGroup, resource.Name); ok && candidate.Namespaced != resource.Namespaced {
			report.Scope.Calculation = "incomplete"
			report.Diagnostics = append(report.Diagnostics, model.Diagnostic{Code: "catalog_scope_changed", Message: "before and after disagree on the scope of " + resource.APIGroup + "/" + resource.Name})
		}
	}
	if report.Scope.Calculation == "incomplete" {
		return report
	}
	unknownResources := map[string]bool{}
	markUnknownResource := func(group, resource string) {
		report.Scope.Calculation = "incomplete"
		key := group + "/" + resource
		if !unknownResources[key] {
			report.Diagnostics = append(report.Diagnostics, model.Diagnostic{Code: "unknown_resource_scope", Message: "supply catalog scope for " + key})
			unknownResources[key] = true
		}
	}
	for _, identity := range identities {
		if !identity.Complete() {
			report.Scope.IdentityCompleteness = "incomplete"
		}
		old, new := left.Prepare(identity), right.Prepare(identity)
		for _, principal := range []*evaluate.Principal{old, new} {
			for _, diagnostic := range principal.Diagnostics() {
				report.Diagnostics = append(report.Diagnostics, diagnostic)
				report.Scope.Calculation = "incomplete"
				if diagnostic.Code != "incomplete_groups" {
					report.Scope.PolicyCompleteness = "incomplete"
				}
			}
		}
		if old.Signature() == new.Signature() {
			continue
		}
		grants := append(slices.Clone(old.Grants), new.Grants...)
		for _, grant := range grants {
			for _, group := range grant.Rule.APIGroups {
				for _, resourcePath := range grant.Rule.Resources {
					resource, _, _ := strings.Cut(resourcePath, "/")
					if group == "*" || resource == "*" {
						continue
					}
					_, oldKnown := before.Catalog.Find(group, resource)
					_, newKnown := after.Catalog.Find(group, resource)
					if !oldKnown && !newKnown {
						markUnknownResource(group, resource)
					}
				}
			}
		}
		finished := walkRequests(grants, before, after, func(request model.Request, region Region) bool {
			if report.Compared >= budget {
				return false
			}
			report.Compared++
			previous, current := old.Decide(request), new.Decide(request)
			if request.URL == "" && region["apiGroup"].Value != nil && region["resource"].Value != nil {
				_, oldKnown := before.Catalog.Find(request.APIGroup, request.Resource)
				_, newKnown := after.Catalog.Find(request.APIGroup, request.Resource)
				if !oldKnown && !newKnown {
					if previous.Result != "granted" && current.Result != "granted" {
						return true
					}
					markUnknownResource(request.APIGroup, request.Resource)
					previous.Result, current.Result = "unknown", "unknown"
				}
			}
			typeName := changeType(previous, current)
			if typeName == "" {
				return true
			}
			if typeName == "indeterminate" {
				report.Scope.Calculation = "incomplete"
			}
			change := Change{TargetType: identity.TargetType(), Type: typeName, Subject: identity.Username(), Request: request, Range: region, Before: previous.Result, After: current.Result, BeforeEvidence: previous.Evidence, AfterEvidence: current.Evidence, Risk: []Risk{}}
			change.ID = model.Hash([]any{change.TargetType, change.Subject, change.Type, region})[:20]
			if change.Type == "added" {
				change.Risk = risks(request)
			}
			report.Changes = append(report.Changes, change)
			return true
		})
		if !finished {
			report.Scope.Calculation = "incomplete"
			report.Diagnostics = append(report.Diagnostics, model.Diagnostic{Code: "budget_exceeded", Message: fmt.Sprintf("comparison stopped after %d classes; increase --max-classes", budget)})
			break
		}
	}
	for key, old := range before.Objects {
		if current, ok := after.Objects[key]; ok && strings.HasSuffix(old.Kind, "Binding") && old.RoleRef != current.RoleRef {
			report.Diagnostics = append(report.Diagnostics, model.Diagnostic{Code: "binding_recreation_required", Message: "roleRef is immutable; applying this target state requires recreating the binding", Object: key})
		}
	}
	slices.SortFunc(report.Changes, func(a, b Change) int { return strings.Compare(a.Subject+"/"+a.ID, b.Subject+"/"+b.ID) })
	slices.SortFunc(report.Diagnostics, func(a, b model.Diagnostic) int { return strings.Compare(model.Hash(a), model.Hash(b)) })
	report.Diagnostics = slices.CompactFunc(report.Diagnostics, func(a, b model.Diagnostic) bool { return model.Hash(a) == model.Hash(b) })
	return report
}

func evidenceKeys(evidence []model.Evidence) []string {
	keys := []string{}
	for _, item := range evidence {
		keys = append(keys, item.Key())
	}
	return model.Unique(keys)
}
func changeType(before, after model.Decision) string {
	if before.Result == "unknown" || after.Result == "unknown" {
		return "indeterminate"
	}
	if before.Result != "granted" && after.Result == "granted" {
		return "added"
	}
	if before.Result == "granted" && after.Result != "granted" {
		return "removed"
	}
	if before.Result != "granted" {
		return ""
	}
	old, new := evidenceKeys(before.Evidence), evidenceKeys(after.Evidence)
	if slices.Equal(old, new) {
		return ""
	}
	removed, retained := false, false
	for _, key := range old {
		if slices.Contains(new, key) {
			retained = true
		} else {
			removed = true
		}
	}
	if removed && retained {
		return "retained_via_other_path"
	}
	return "path_changed"
}

func risks(request model.Request) []Risk {
	result := []Risk{}
	if request.APIGroup == "" && request.Resource == "secrets" && slices.Contains([]string{"get", "list", "watch"}, request.Verb) {
		result = append(result, Risk{"secret_read", "high", "new access to Secret data in the reported scope"})
	}
	if request.APIGroup == "rbac.authorization.k8s.io" && slices.Contains([]string{"roles", "clusterroles"}, request.Resource) && slices.Contains([]string{"bind", "escalate"}, request.Verb) {
		result = append(result, Risk{"rbac_delegation", "high", "new authorization-management permission; evaluate accompanying write and object restrictions"})
	}
	if request.Verb == "impersonate" {
		result = append(result, Risk{"impersonation", "high", "new impersonation permission; evaluate the identities and companion groups allowed"})
	}
	return result
}
