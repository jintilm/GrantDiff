package input

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jintilm/GrantDiff/internal/model"
	yamlv3 "go.yaml.in/yaml/v3"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/yaml"
)

const MaxBytes = 64 << 20
const MaxFiles = 10000

type Options struct {
	DefaultNamespace string
	AllowEmpty       bool
}
type File struct {
	Name string
	Data []byte
}

func Load(path string, opts Options) (*model.State, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		data, err := ReadLimited(path)
		if err != nil {
			return nil, err
		}
		return Parse([]File{{filepath.Base(path), data}}, path, opts)
	}
	root, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var files []File
	var total int
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, resolved)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
			return fmt.Errorf("input path escapes root: %s", path)
		}
		if entry.IsDir() {
			if path != root && entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !IsManifest(entry.Name()) {
			return nil
		}
		if len(files) >= MaxFiles {
			return fmt.Errorf("input exceeds %d files", MaxFiles)
		}
		data, err := ReadLimited(path)
		if err != nil {
			return err
		}
		total += len(data)
		if total > MaxBytes {
			return fmt.Errorf("input exceeds %d bytes", MaxBytes)
		}
		display, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, File{filepath.ToSlash(display), data})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return Parse(files, path, opts)
}

func IsManifest(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml", ".json":
		return true
	}
	return false
}

func ReadLimited(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxBytes+1))
	if len(data) > MaxBytes {
		return nil, fmt.Errorf("file exceeds %d bytes: %s", MaxBytes, path)
	}
	return data, err
}

func Parse(files []File, origin string, opts Options) (*model.State, error) {
	state := model.NewState()
	state.Origin = origin
	slices.SortFunc(files, func(a, b File) int { return strings.Compare(a.Name, b.Name) })
	for _, file := range files {
		decoder := yamlv3.NewDecoder(bytes.NewReader(bytes.TrimPrefix(file.Data, []byte{0xef, 0xbb, 0xbf})))
		for document := 1; ; document++ {
			var node yamlv3.Node
			err := decoder.Decode(&node)
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("%s document %d: %w", file.Name, document, err)
			}
			if len(node.Content) == 0 || node.Content[0].Tag == "!!null" {
				continue
			}
			if err := validateNode(&node, 0); err != nil {
				return nil, fmt.Errorf("%s: %w", file.Name, err)
			}
			source := model.Source{File: file.Name, Document: document, Line: node.Content[0].Line}
			if err := addNode(state, node.Content[0], source, opts); err != nil {
				return nil, err
			}
		}
	}
	if len(state.Objects) == 0 && !opts.AllowEmpty {
		return nil, fmt.Errorf("no RBAC objects in %s; explicitly allow an empty state if intended", origin)
	}
	state.Namespaces = model.Unique(state.Namespaces)
	return state, nil
}

func validateNode(node *yamlv3.Node, depth int) error {
	if depth > 64 {
		return fmt.Errorf("YAML nesting exceeds 64 levels")
	}
	if node.Kind == yamlv3.AliasNode {
		return fmt.Errorf("YAML aliases require expansion before review (line %d)", node.Line)
	}
	if node.Kind == yamlv3.MappingNode {
		keys := map[string]bool{}
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Kind != yamlv3.ScalarNode || key.Tag != "!!str" {
				return fmt.Errorf("mapping keys must be strings at line %d", key.Line)
			}
			if keys[key.Value] {
				return fmt.Errorf("duplicate key %q at line %d", key.Value, key.Line)
			}
			keys[key.Value] = true
		}
	}
	for _, child := range node.Content {
		if err := validateNode(child, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func child(node *yamlv3.Node, name string) *yamlv3.Node {
	if node.Kind != yamlv3.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == name {
			return node.Content[i+1]
		}
	}
	return nil
}

func addNode(state *model.State, node *yamlv3.Node, source model.Source, opts Options) error {
	if node.Kind != yamlv3.MappingNode {
		return fmt.Errorf("%s:%d: expected a Kubernetes object", source.File, source.Line)
	}
	kindName := ""
	if kind := child(node, "kind"); kind != nil {
		if kind.Kind != yamlv3.ScalarNode || kind.Tag != "!!str" {
			return fmt.Errorf("%s:%d: kind must be a string", source.File, source.Line)
		}
		kindName = kind.Value
	}
	if kindName == "List" || slices.Contains([]string{"RoleList", "ClusterRoleList", "RoleBindingList", "ClusterRoleBindingList", "NamespaceList", "ServiceAccountList"}, kindName) {
		itemKind, itemVersion := "", ""
		if kindName != "List" {
			itemKind = strings.TrimSuffix(kindName, "List")
			itemVersion = "rbac.authorization.k8s.io/v1"
			if itemKind == "Namespace" || itemKind == "ServiceAccount" {
				itemVersion = "v1"
			}
			version := child(node, "apiVersion")
			if version == nil || version.Tag != "!!str" || version.Value != itemVersion {
				return fmt.Errorf("%s:%d: %s requires apiVersion %s", source.File, source.Line, kindName, itemVersion)
			}
		}
		items := child(node, "items")
		if items == nil || items.Kind != yamlv3.SequenceNode {
			return fmt.Errorf("%s:%d: List requires items", source.File, source.Line)
		}
		for _, item := range items.Content {
			itemSource := source
			itemSource.Line = item.Line
			if itemKind != "" {
				var err error
				item, err = typedListItem(item, itemKind, itemVersion)
				if err != nil {
					return fmt.Errorf("%s:%d: %w", source.File, itemSource.Line, err)
				}
			}
			if err := addNode(state, item, itemSource, opts); err != nil {
				return err
			}
		}
		return nil
	}
	data, err := yamlv3.Marshal(node)
	if err != nil {
		return err
	}
	var object model.Object
	if err := yaml.Unmarshal(data, &object); err != nil {
		return fmt.Errorf("%s:%d: %w", source.File, source.Line, err)
	}
	switch object.Kind {
	case "Namespace":
		state.Namespaces = append(state.Namespaces, object.Metadata.Name)
		return nil
	case "ServiceAccount":
		namespace := object.Metadata.Namespace
		if namespace == "" {
			namespace = opts.DefaultNamespace
		}
		if namespace == "" {
			return fmt.Errorf("ServiceAccount %s requires namespace", object.Metadata.Name)
		}
		identity := model.Identity{ID: "system:serviceaccount:" + namespace + ":" + object.Metadata.Name, Kind: "ServiceAccount", Name: object.Metadata.Name, Namespace: namespace, GroupsComplete: true}
		state.Identities = append(state.Identities, identity)
		return nil
	case "Role", "ClusterRole", "RoleBinding", "ClusterRoleBinding":
		if object.APIVersion != "rbac.authorization.k8s.io/v1" {
			return fmt.Errorf("%s: unsupported RBAC apiVersion %q", source.File, object.APIVersion)
		}
		if err := fields(node, []string{"apiVersion", "kind", "metadata", "rules", "subjects", "roleRef", "aggregationRule"}); err != nil {
			return err
		}
		if metadata := child(node, "metadata"); metadata != nil {
			if err := fields(metadata, []string{"name", "namespace", "labels", "annotations", "uid", "resourceVersion", "generation", "creationTimestamp", "deletionTimestamp", "deletionGracePeriodSeconds", "ownerReferences", "finalizers", "managedFields", "generateName", "selfLink"}); err != nil {
				return err
			}
		}
		if aggregation := child(node, "aggregationRule"); aggregation != nil && aggregation.Tag != "!!null" {
			if err := fields(aggregation, []string{"clusterRoleSelectors"}); err != nil {
				return err
			}
			if selectors := child(aggregation, "clusterRoleSelectors"); selectors != nil {
				for _, selector := range selectors.Content {
					if err := fields(selector, []string{"matchLabels", "matchExpressions"}); err != nil {
						return err
					}
					if expressions := child(selector, "matchExpressions"); expressions != nil {
						for _, expression := range expressions.Content {
							if err := fields(expression, []string{"key", "operator", "values"}); err != nil {
								return err
							}
						}
					}
				}
			}
		}
		if rules := child(node, "rules"); rules != nil {
			for _, rule := range rules.Content {
				if err := fields(rule, []string{"verbs", "apiGroups", "resources", "resourceNames", "nonResourceURLs"}); err != nil {
					return err
				}
			}
		}
		if ref := child(node, "roleRef"); ref != nil {
			if err := fields(ref, []string{"apiGroup", "kind", "name"}); err != nil {
				return err
			}
		}
		if subjects := child(node, "subjects"); subjects != nil {
			for _, subject := range subjects.Content {
				if err := fields(subject, []string{"apiGroup", "kind", "name", "namespace"}); err != nil {
					return err
				}
			}
		}
	default:
		if strings.HasPrefix(object.APIVersion, "rbac.authorization.k8s.io/") {
			return fmt.Errorf("unsupported RBAC kind %q", object.Kind)
		}
		state.Diagnostics = append(state.Diagnostics, model.Diagnostic{Code: "object_skipped", Message: "object is outside RBAC scope: " + object.Kind, Source: &source})
		return nil
	}
	if object.Kind == "Role" || object.Kind == "RoleBinding" {
		if object.Metadata.Namespace == "" {
			object.Metadata.Namespace = opts.DefaultNamespace
		}
		state.Namespaces = append(state.Namespaces, object.Metadata.Namespace)
	}
	if err := ValidateObject(&object); err != nil {
		return fmt.Errorf("%s:%d: %w", source.File, source.Line, err)
	}
	if previous, exists := state.Objects[object.Key()]; exists {
		return fmt.Errorf("duplicate object %s at %s:%d and %s:%d", object.Key(), previous.Source.File, previous.Source.Line, source.File, source.Line)
	}
	object.Source = source
	if rules := child(node, "rules"); rules != nil {
		for _, rule := range rules.Content {
			position := source
			position.Line = rule.Line
			object.RuleSources = append(object.RuleSources, position)
		}
	}
	state.Objects[object.Key()] = &object
	return nil
}

// Typed Kubernetes lists carry the item kind and version on the envelope.
// Add missing metadata on a copy and retain the original item coordinates.
func typedListItem(item *yamlv3.Node, kind, version string) (*yamlv3.Node, error) {
	if item.Kind != yamlv3.MappingNode {
		return nil, fmt.Errorf("%sList items must be objects", kind)
	}
	result := *item
	result.Content = slices.Clone(item.Content)
	for _, field := range []struct{ name, value string }{{"kind", kind}, {"apiVersion", version}} {
		if value := child(item, field.name); value != nil {
			if value.Tag != "!!str" || value.Value != field.value {
				return nil, fmt.Errorf("%sList item conflicts with %s %q", kind, field.name, field.value)
			}
			continue
		}
		result.Content = append(result.Content,
			&yamlv3.Node{Kind: yamlv3.ScalarNode, Tag: "!!str", Value: field.name, Line: item.Line},
			&yamlv3.Node{Kind: yamlv3.ScalarNode, Tag: "!!str", Value: field.value, Line: item.Line},
		)
	}
	return &result, nil
}

func ValidateObject(object *model.Object) error {
	if object.Metadata.Name == "" {
		return fmt.Errorf("%s requires metadata.name", object.Kind)
	}
	if strings.ContainsAny(object.Metadata.Name, "/%") || object.Metadata.Name == "." || object.Metadata.Name == ".." {
		return fmt.Errorf("invalid RBAC name %q", object.Metadata.Name)
	}
	if object.Kind == "Role" || object.Kind == "RoleBinding" {
		if object.Metadata.Namespace == "" {
			return fmt.Errorf("%s %q requires metadata.namespace", object.Kind, object.Metadata.Name)
		}
		if errs := validation.IsDNS1123Label(object.Metadata.Namespace); len(errs) > 0 {
			return fmt.Errorf("invalid namespace %q: %s", object.Metadata.Namespace, strings.Join(errs, "; "))
		}
	}
	if object.Kind == "Role" || object.Kind == "ClusterRole" {
		if len(object.Subjects) > 0 || object.RoleRef.Name != "" {
			return fmt.Errorf("role has binding fields")
		}
		if object.AggregationRule != nil && object.Kind != "ClusterRole" {
			return fmt.Errorf("aggregationRule is only valid for ClusterRole")
		}
	}
	if strings.HasPrefix(object.Kind, "Cluster") && object.Metadata.Namespace != "" {
		return fmt.Errorf("%s must have cluster scope", object.Kind)
	}
	if object.Kind == "RoleBinding" || object.Kind == "ClusterRoleBinding" {
		if len(object.Rules) > 0 || object.AggregationRule != nil {
			return fmt.Errorf("binding has role fields")
		}
		if object.RoleRef.Name == "" || object.RoleRef.Kind == "" {
			return fmt.Errorf("%s %q requires roleRef.kind and roleRef.name", object.Kind, object.Metadata.Name)
		}
		if object.RoleRef.Kind != "Role" && object.RoleRef.Kind != "ClusterRole" {
			return fmt.Errorf("invalid roleRef.kind %q", object.RoleRef.Kind)
		}
		if object.Kind == "ClusterRoleBinding" && object.RoleRef.Kind != "ClusterRole" {
			return fmt.Errorf("ClusterRoleBinding must refer to ClusterRole")
		}
		if object.RoleRef.APIGroup != "rbac.authorization.k8s.io" {
			return fmt.Errorf("invalid roleRef.apiGroup %q", object.RoleRef.APIGroup)
		}
		for index := range object.Subjects {
			subject := &object.Subjects[index]
			if subject.Name == "" {
				return fmt.Errorf("%s %q contains subject without name", object.Kind, object.Metadata.Name)
			}
			if subject.Kind != "User" && subject.Kind != "Group" && subject.Kind != "ServiceAccount" {
				return fmt.Errorf("unsupported subject kind %q", subject.Kind)
			}
			if subject.Kind == "ServiceAccount" {
				if subject.Namespace == "" {
					subject.Namespace = object.Metadata.Namespace
				}
				if subject.Namespace == "" || subject.APIGroup != "" {
					return fmt.Errorf("ServiceAccount subject requires namespace and empty apiGroup")
				}
			} else {
				if subject.APIGroup == "" {
					subject.APIGroup = "rbac.authorization.k8s.io"
				}
				if subject.APIGroup != "rbac.authorization.k8s.io" {
					return fmt.Errorf("invalid subject apiGroup")
				}
			}
		}
	}
	for index, rule := range object.Rules {
		if len(rule.Verbs) == 0 {
			return fmt.Errorf("rule %d requires verbs", index)
		}
		if len(rule.Resources) > 0 && len(rule.APIGroups) == 0 {
			return fmt.Errorf("rule %d requires apiGroups", index)
		}
		if len(rule.Resources) == 0 && len(rule.NonResourceURLs) == 0 {
			return fmt.Errorf("%s %q rule %d has no resources or nonResourceURLs", object.Kind, object.Metadata.Name, index)
		}
		if len(rule.Resources) > 0 && len(rule.NonResourceURLs) > 0 {
			return fmt.Errorf("%s %q rule %d mixes resources and nonResourceURLs", object.Kind, object.Metadata.Name, index)
		}
		if len(rule.NonResourceURLs) > 0 && object.Kind == "Role" {
			return fmt.Errorf("nonResourceURLs require ClusterRole")
		}
		if len(rule.NonResourceURLs) > 0 && (len(rule.APIGroups) > 0 || len(rule.ResourceNames) > 0) {
			return fmt.Errorf("non-resource rule has resource attributes")
		}
		for _, url := range rule.NonResourceURLs {
			if url != "*" && (!strings.HasPrefix(url, "/") || strings.Contains(strings.TrimSuffix(url, "*"), "*")) {
				return fmt.Errorf("invalid nonResourceURL %q", url)
			}
		}
	}
	return nil
}

func fields(node *yamlv3.Node, allowed []string) error {
	if node.Kind != yamlv3.MappingNode {
		return fmt.Errorf("expected mapping at line %d", node.Line)
	}
	for i := 0; i < len(node.Content); i += 2 {
		if !slices.Contains(allowed, node.Content[i].Value) {
			return fmt.Errorf("unknown field %q at line %d", node.Content[i].Value, node.Content[i].Line)
		}
	}
	return nil
}
