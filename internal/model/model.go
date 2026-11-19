package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	rbac "k8s.io/api/rbac/v1"
)

type Source struct {
	File     string `json:"file"`
	Document int    `json:"document"`
	Line     int    `json:"line,omitempty"`
}

type Diagnostic struct {
	Code    string  `json:"code"`
	Message string  `json:"message"`
	Object  string  `json:"object,omitempty"`
	Source  *Source `json:"source,omitempty"`
}

type Object struct {
	APIVersion      string                `json:"apiVersion"`
	Kind            string                `json:"kind"`
	Metadata        Metadata              `json:"metadata"`
	Rules           []rbac.PolicyRule     `json:"rules,omitempty"`
	Subjects        []rbac.Subject        `json:"subjects,omitempty"`
	RoleRef         rbac.RoleRef          `json:"roleRef,omitempty"`
	AggregationRule *rbac.AggregationRule `json:"aggregationRule,omitempty"`
	Source          Source                `json:"-"`
	RuleSources     []Source              `json:"-"`
}

type Metadata struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

func Key(kind, namespace, name string) string { return kind + "/" + namespace + "/" + name }
func (o Object) Key() string                  { return Key(o.Kind, o.Metadata.Namespace, o.Metadata.Name) }
func (o Object) Digest() string               { return Hash(o) }
func Hash(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

type Resource struct {
	APIGroup     string   `json:"apiGroup"`
	Name         string   `json:"name"`
	Namespaced   bool     `json:"namespaced"`
	Subresources []string `json:"subresources,omitempty"`
}
type Catalog struct {
	SchemaVersion     string     `json:"schemaVersion"`
	KubernetesVersion string     `json:"kubernetesVersion,omitempty"`
	Resources         []Resource `json:"resources"`
}

func (c Catalog) Find(group, name string) (Resource, bool) {
	for _, resource := range c.Resources {
		if resource.APIGroup == group && resource.Name == name {
			return resource, true
		}
	}
	return Resource{}, false
}

type Identity struct {
	ID             string   `json:"id"`
	Kind           string   `json:"kind"`
	Name           string   `json:"name"`
	Namespace      string   `json:"namespace,omitempty"`
	Authenticated  bool     `json:"authenticated,omitempty"`
	Groups         []string `json:"groups,omitempty"`
	GroupsComplete bool     `json:"groupsComplete"`
}

func (i Identity) Username() string {
	if i.Kind == "ServiceAccount" {
		return "system:serviceaccount:" + i.Namespace + ":" + i.Name
	}
	return i.Name
}
func (i Identity) TargetType() string {
	if i.Kind == "Group" {
		return "group_grant"
	}
	return "identity_access"
}
func (i Identity) Key() string { return i.TargetType() + "/" + i.Username() }
func (i Identity) EffectiveGroups() []string {
	groups := slices.Clone(i.Groups)
	switch i.Kind {
	case "ServiceAccount":
		groups = append(groups, "system:authenticated", "system:serviceaccounts", "system:serviceaccounts:"+i.Namespace)
	case "Group":
		groups = append(groups, i.Name)
	case "User":
		if i.Authenticated {
			groups = append(groups, "system:authenticated")
		}
		if i.Name == "system:anonymous" {
			groups = append(groups, "system:unauthenticated")
		}
	}
	return Unique(groups)
}
func (i Identity) Complete() bool { return i.Kind != "User" || i.GroupsComplete }
func (i Identity) Validate() error {
	if i.ID == "" || i.Name == "" {
		return fmt.Errorf("identity id and name are required")
	}
	switch i.Kind {
	case "ServiceAccount":
		if i.Namespace == "" {
			return fmt.Errorf("service account %q requires namespace", i.ID)
		}
	case "User", "Group":
		if i.Namespace != "" {
			return fmt.Errorf("%s identity %q has a namespace", i.Kind, i.ID)
		}
	default:
		return fmt.Errorf("unsupported identity kind %q", i.Kind)
	}
	return nil
}

type Identities struct {
	SchemaVersion string     `json:"schemaVersion"`
	Identities    []Identity `json:"identities"`
}

type State struct {
	Objects     map[string]*Object
	Identities  []Identity
	Namespaces  []string
	Catalog     Catalog
	Diagnostics []Diagnostic
	Origin      string
}

func NewState() *State { return &State{Objects: map[string]*Object{}} }
func (s *State) SortedObjects() []*Object {
	keys := make([]string, 0, len(s.Objects))
	for key := range s.Objects {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	objects := make([]*Object, 0, len(keys))
	for _, key := range keys {
		objects = append(objects, s.Objects[key])
	}
	return objects
}
func (s *State) Digest() string { return Hash(s.SortedObjects()) }

type Request struct {
	Verb        string `json:"verb"`
	APIGroup    string `json:"apiGroup"`
	Resource    string `json:"resource,omitempty"`
	Subresource string `json:"subresource,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	Name        string `json:"name,omitempty"`
	URL         string `json:"url,omitempty"`
}

func (r Request) ResourcePath() string {
	if r.Subresource != "" {
		return r.Resource + "/" + r.Subresource
	}
	return r.Resource
}
func (r Request) String() string {
	if r.URL != "" {
		return r.Verb + " " + r.URL
	}
	return fmt.Sprintf("%s %s/%s group=%q namespace=%q", r.Verb, r.ResourcePath(), r.Name, r.APIGroup, r.Namespace)
}

type Evidence struct {
	Subject         string   `json:"subject"`
	Binding         string   `json:"binding"`
	Role            string   `json:"role"`
	RuleID          string   `json:"ruleId"`
	AggregationPath []string `json:"aggregationPath,omitempty"`
	BindingSource   Source   `json:"bindingSource"`
	RuleSource      Source   `json:"ruleSource"`
}

func (e Evidence) Key() string {
	return strings.Join([]string{e.Subject, e.Binding, e.Role, e.RuleID, strings.Join(e.AggregationPath, ">")}, "|")
}

type Decision struct {
	Result      string       `json:"result"`
	Evidence    []Evidence   `json:"evidence"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

func Unique(values []string) []string {
	result := slices.Clone(values)
	slices.Sort(result)
	return slices.Compact(result)
}
func RuleID(rule rbac.PolicyRule) string {
	rule.Verbs = Unique(rule.Verbs)
	rule.APIGroups = Unique(rule.APIGroups)
	rule.Resources = Unique(rule.Resources)
	rule.ResourceNames = Unique(rule.ResourceNames)
	rule.NonResourceURLs = Unique(rule.NonResourceURLs)
	return Hash(rule)
}
