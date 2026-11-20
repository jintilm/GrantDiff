package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jintilm/GrantDiff/internal/input"
	"github.com/jintilm/GrantDiff/internal/model"
	auth "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var ErrCaptureChanged = errors.New("RBAC or identity objects changed during capture; retry snapshot")

type Snapshot struct {
	SchemaVersion     string            `json:"schemaVersion"`
	KubernetesVersion string            `json:"kubernetesVersion"`
	Started           time.Time         `json:"started"`
	Finished          time.Time         `json:"finished"`
	Digest            string            `json:"digest"`
	Consistency       string            `json:"consistency"`
	ResourceVersions  map[string]string `json:"resourceVersions"`
}

func Config(path, contextName string) (*rest.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if path != "" {
		rules.ExplicitPath = path
	}
	overrides := &clientcmd.ConfigOverrides{CurrentContext: contextName}
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		return nil, err
	}
	config.Timeout = 30 * time.Second
	return config, nil
}

func Connect(path, contextName string) (*kubernetes.Clientset, error) {
	config, err := Config(path, contextName)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func Capture(ctx context.Context, client kubernetes.Interface) (*model.State, Snapshot, []byte, error) {
	metadata := Snapshot{SchemaVersion: "1.0", Started: time.Now().UTC(), ResourceVersions: map[string]string{}}
	version, err := client.Discovery().ServerVersion()
	if err != nil {
		return nil, metadata, nil, err
	}
	metadata.KubernetesVersion = version.GitVersion
	first, _, err := collect(ctx, client)
	if err != nil {
		return nil, metadata, nil, err
	}
	second, versions, err := collect(ctx, client)
	if err != nil {
		return nil, metadata, nil, err
	}
	// Two complete reads expose changes observed during collection. They are not
	// a cross-resource transaction; capture metadata records that limitation.
	if canonicalDigest(first) != canonicalDigest(second) {
		return nil, metadata, nil, ErrCaptureChanged
	}
	metadata.ResourceVersions = versions
	data, err := json.Marshal(map[string]any{"apiVersion": "v1", "kind": "List", "items": second})
	if err != nil {
		return nil, metadata, nil, err
	}
	state, err := input.Parse([]input.File{{Name: "rbac.json", Data: data}}, "cluster-snapshot", input.Options{})
	if err != nil {
		return nil, metadata, nil, err
	}
	catalog, err := discover(client)
	if err != nil {
		return nil, metadata, nil, err
	}
	state.Catalog = catalog
	state.Catalog.KubernetesVersion = version.GitVersion
	metadata.Digest = state.Digest()
	metadata.Finished = time.Now().UTC()
	metadata.Consistency = "two-equal-observations"
	return state, metadata, data, nil
}

func collect(ctx context.Context, client kubernetes.Interface) ([]json.RawMessage, map[string]string, error) {
	lists := []struct {
		kind string
		get  func() (any, string, error)
	}{
		{"Role", func() (any, string, error) {
			value, err := client.RbacV1().Roles("").List(ctx, metav1.ListOptions{})
			if err != nil {
				return nil, "", err
			}
			return value.Items, value.ResourceVersion, nil
		}},
		{"ClusterRole", func() (any, string, error) {
			value, err := client.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
			if err != nil {
				return nil, "", err
			}
			return value.Items, value.ResourceVersion, nil
		}},
		{"RoleBinding", func() (any, string, error) {
			value, err := client.RbacV1().RoleBindings("").List(ctx, metav1.ListOptions{})
			if err != nil {
				return nil, "", err
			}
			return value.Items, value.ResourceVersion, nil
		}},
		{"ClusterRoleBinding", func() (any, string, error) {
			value, err := client.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
			if err != nil {
				return nil, "", err
			}
			return value.Items, value.ResourceVersion, nil
		}},
		{"Namespace", func() (any, string, error) {
			value, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
			if err != nil {
				return nil, "", err
			}
			return value.Items, value.ResourceVersion, nil
		}},
		{"ServiceAccount", func() (any, string, error) {
			value, err := client.CoreV1().ServiceAccounts("").List(ctx, metav1.ListOptions{})
			if err != nil {
				return nil, "", err
			}
			return value.Items, value.ResourceVersion, nil
		}},
	}
	objects := []json.RawMessage{}
	versions := map[string]string{}
	for _, list := range lists {
		value, version, err := list.get()
		if err != nil {
			return nil, nil, fmt.Errorf("list %s: %w", list.kind, err)
		}
		versions[list.kind] = version
		data, err := json.Marshal(value)
		if err != nil {
			return nil, nil, err
		}
		var items []map[string]any
		if err := json.Unmarshal(data, &items); err != nil {
			return nil, nil, err
		}
		for _, item := range items {
			apiVersion := "rbac.authorization.k8s.io/v1"
			if list.kind == "Namespace" || list.kind == "ServiceAccount" {
				apiVersion = "v1"
			}
			meta := item["metadata"].(map[string]any)
			clean := map[string]any{"apiVersion": apiVersion, "kind": list.kind, "metadata": map[string]any{"name": meta["name"], "namespace": meta["namespace"], "labels": meta["labels"]}}
			for _, field := range []string{"rules", "subjects", "roleRef", "aggregationRule"} {
				if value, ok := item[field]; ok {
					clean[field] = value
				}
			}
			encoded, err := json.Marshal(clean)
			if err != nil {
				return nil, nil, err
			}
			objects = append(objects, encoded)
		}
	}
	slices.SortFunc(objects, func(a, b json.RawMessage) int { return strings.Compare(string(a), string(b)) })
	return objects, versions, nil
}
func canonicalDigest(objects []json.RawMessage) string { return model.Hash(objects) }

func discover(client kubernetes.Interface) (model.Catalog, error) {
	catalog := model.Catalog{SchemaVersion: "1.0", Resources: []model.Resource{}}
	_, lists, err := client.Discovery().ServerGroupsAndResources()
	if err != nil {
		return catalog, err
	}
	resources := map[string]model.Resource{}
	for _, list := range lists {
		groupVersion, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			return catalog, err
		}
		for _, resource := range list.APIResources {
			name, sub, hasSub := strings.Cut(resource.Name, "/")
			key := groupVersion.Group + "/" + name
			entry := resources[key]
			entry.Name = name
			entry.APIGroup = groupVersion.Group
			entry.Namespaced = resource.Namespaced
			if hasSub {
				entry.Subresources = append(entry.Subresources, sub)
			}
			resources[key] = entry
		}
	}
	for _, resource := range resources {
		resource.Subresources = model.Unique(resource.Subresources)
		catalog.Resources = append(catalog.Resources, resource)
	}
	slices.SortFunc(catalog.Resources, func(a, b model.Resource) int { return strings.Compare(a.APIGroup+"/"+a.Name, b.APIGroup+"/"+b.Name) })
	return catalog, nil
}

func Save(directory string, state *model.State, metadata Snapshot, data []byte) error {
	if _, err := os.Stat(directory); err == nil {
		return fmt.Errorf("snapshot destination already exists: %s", directory)
	} else if !os.IsNotExist(err) {
		return err
	}
	parent := filepath.Dir(directory)
	if err := os.MkdirAll(parent, 0700); err != nil {
		return err
	}
	temp, err := os.MkdirTemp(parent, ".rbac-snapshot-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	catalog, err := json.MarshalIndent(state.Catalog, "", "  ")
	if err != nil {
		return err
	}
	meta, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	for name, content := range map[string][]byte{"rbac.json": data, "catalog.json": catalog, "metadata.json": meta} {
		if err := os.WriteFile(filepath.Join(temp, name), content, 0600); err != nil {
			return err
		}
	}
	return os.Rename(temp, directory)
}

func Review(ctx context.Context, client kubernetes.Interface, identity model.Identity, request model.Request) (bool, error) {
	spec := auth.SubjectAccessReviewSpec{User: identity.Username(), Groups: identity.EffectiveGroups()}
	if request.URL != "" {
		spec.NonResourceAttributes = &auth.NonResourceAttributes{Verb: request.Verb, Path: request.URL}
	} else {
		spec.ResourceAttributes = &auth.ResourceAttributes{Verb: request.Verb, Group: request.APIGroup, Resource: request.Resource, Subresource: request.Subresource, Namespace: request.Namespace, Name: request.Name}
	}
	response, err := client.AuthorizationV1().SubjectAccessReviews().Create(ctx, &auth.SubjectAccessReview{Spec: spec}, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	if response.Status.EvaluationError != "" {
		return false, fmt.Errorf("authorization evaluation: %s", response.Status.EvaluationError)
	}
	return response.Status.Allowed, nil
}
