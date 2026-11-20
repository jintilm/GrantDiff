package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jintilm/GrantDiff/internal/cluster"
	"github.com/jintilm/GrantDiff/internal/evaluate"
	"github.com/jintilm/GrantDiff/internal/input"
	"github.com/jintilm/GrantDiff/internal/model"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

func TestCRDCatalogAgainstDiscovery(t *testing.T) {
	configPath := os.Getenv("GRANTDIFF_TEST_KUBECONFIG")
	if configPath == "" {
		t.Skip("set GRANTDIFF_TEST_KUBECONFIG for CRD discovery validation")
	}
	contextName := os.Getenv("GRANTDIFF_TEST_CONTEXT")
	if !strings.HasPrefix(contextName, "kind-grantdiff-") {
		t.Fatal("CRD validation requires a kind-grantdiff-* context")
	}
	config, err := cluster.Config(configPath, contextName)
	if err != nil {
		t.Fatal(err)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	namespace := "grantdiff-crd-" + suffix
	group := "g" + suffix + ".example.test"
	if _, err := client.CoreV1().Namespaces().Create(ctx, &core.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 15*time.Second)
		defer stop()
		if err := client.CoreV1().Namespaces().Delete(cleanup, namespace, metav1.DeleteOptions{}); err != nil {
			t.Errorf("cleanup namespace: %v", err)
		}
	})
	data, err := os.ReadFile("../examples/crd/after/resources.yaml")
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.ReplaceAll(strings.ReplaceAll(string(data), "example.test", group), "namespace: apps", "namespace: "+namespace))
	state, err := input.Parse([]input.File{{Name: "crd-scenario.yaml", Data: data}}, "CRD scenario", input.Options{})
	if err != nil {
		t.Fatal(err)
	}
	crdJSON, err := yaml.YAMLToJSON([]byte(strings.Split(string(data), "---")[0]))
	if err != nil {
		t.Fatal(err)
	}
	var definition unstructured.Unstructured
	if err := json.Unmarshal(crdJSON, &definition); err != nil {
		t.Fatal(err)
	}
	// Persist a status field so the actual subresource request can be checked.
	versions, _, _ := unstructured.NestedSlice(definition.Object, "spec", "versions")
	version := versions[0].(map[string]any)
	version["schema"] = map[string]any{"openAPIV3Schema": map[string]any{"type": "object", "properties": map[string]any{"status": map[string]any{"type": "object", "properties": map[string]any{"phase": map[string]any{"type": "string"}}}}}}
	if err := unstructured.SetNestedSlice(definition.Object, versions, "spec", "versions"); err != nil {
		t.Fatal(err)
	}
	crds := dynamicClient.Resource(schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"})
	if _, err := crds.Create(ctx, &definition, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 15*time.Second)
		defer stop()
		if err := crds.Delete(cleanup, definition.GetName(), metav1.DeleteOptions{}); err != nil {
			t.Errorf("cleanup CRD: %v", err)
		}
	})
	established := false
	for !established {
		current, err := crds.Get(ctx, definition.GetName(), metav1.GetOptions{})
		if err != nil {
			t.Fatal(err)
		}
		conditions, _, _ := unstructured.NestedSlice(current.Object, "status", "conditions")
		for _, value := range conditions {
			condition := value.(map[string]any)
			if condition["type"] == "Established" && condition["status"] == "True" {
				established = true
			}
		}
		if !established {
			select {
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
	role := state.Objects[model.Key("Role", namespace, "widget-reader")]
	if _, err := client.RbacV1().Roles(namespace).Create(ctx, &rbac.Role{ObjectMeta: metav1.ObjectMeta{Name: role.Metadata.Name}, Rules: role.Rules}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	binding := state.Objects[model.Key("RoleBinding", namespace, "widget-access")]
	if _, err := client.RbacV1().RoleBindings(namespace).Create(ctx, &rbac.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: binding.Metadata.Name}, Subjects: binding.Subjects, RoleRef: binding.RoleRef}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	_, _, _, err = captureStable(t, ctx, client)
	if err != nil {
		t.Fatal(err)
	}
	discovery, err := client.Discovery().ServerResourcesForGroupVersion(group + "/v1")
	if err != nil {
		t.Fatal(err)
	}
	baseFound, statusFound := false, false
	for _, resource := range discovery.APIResources {
		if resource.Name == "widgets" {
			baseFound = resource.Namespaced
		}
		if resource.Name == "widgets/status" {
			statusFound = resource.Namespaced
		}
	}
	if !baseFound || !statusFound {
		t.Fatal("CRD-derived scope or status subresource disagrees with discovery")
	}
	identity := model.Identity{ID: "widget-bot", Kind: "ServiceAccount", Name: "widget-bot", Namespace: namespace}
	requests := []model.Request{
		{Verb: "get", APIGroup: group, Resource: "widgets", Namespace: namespace, Name: "demo"},
		{Verb: "list", APIGroup: group, Resource: "widgets", Namespace: namespace},
		{Verb: "patch", APIGroup: group, Resource: "widgets", Subresource: "status", Namespace: namespace, Name: "demo"},
		{Verb: "patch", APIGroup: group, Resource: "widgets", Namespace: namespace, Name: "demo"},
		{Verb: "get", APIGroup: group, Resource: "widgets", Namespace: "other", Name: "demo"},
	}
	for _, request := range requests {
		if err := evaluate.CheckRequest(request, state.Catalog); err != nil {
			t.Fatal(err)
		}
	}
	checks := checkRequests(t, ctx, client, state, []model.Identity{identity}, requests)
	widgets := schema.GroupVersionResource{Group: group, Version: "v1", Resource: "widgets"}
	if _, err := dynamicClient.Resource(widgets).Namespace(namespace).Create(ctx, &unstructured.Unstructured{Object: map[string]any{"apiVersion": group + "/v1", "kind": "Widget", "metadata": map[string]any{"name": "demo"}}}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	config.Impersonate.UserName, config.Impersonate.Groups = identity.Username(), identity.EffectiveGroups()
	impersonated, err := dynamic.NewForConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	patched, err := impersonated.Resource(widgets).Namespace(namespace).Patch(ctx, "demo", types.MergePatchType, []byte(`{"status":{"phase":"verified"}}`), metav1.PatchOptions{}, "status")
	if err != nil {
		t.Fatal(err)
	}
	phase, _, _ := unstructured.NestedString(patched.Object, "status", "phase")
	if phase != "verified" {
		t.Fatal("status update was not persisted")
	}
	t.Logf("CRD scope and subresources match discovery; %d authorization comparisons and one actual status patch passed", checks)
	if path := os.Getenv("GRANTDIFF_TEST_RESULT"); path != "" {
		data, err := json.MarshalIndent(map[string]any{"authorizationComparisons": checks, "actualRequests": 1, "discovery": "passed", "result": "passed"}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(strings.TrimSuffix(path, filepath.Ext(path))+"-crd.json", data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
