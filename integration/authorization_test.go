package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jintilm/GrantDiff/internal/cluster"
	"github.com/jintilm/GrantDiff/internal/diff"
	"github.com/jintilm/GrantDiff/internal/evaluate"
	"github.com/jintilm/GrantDiff/internal/input"
	"github.com/jintilm/GrantDiff/internal/model"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func TestAuthorizationAgainstKubernetes(t *testing.T) {
	configPath := os.Getenv("GRANTDIFF_TEST_KUBECONFIG")
	if configPath == "" {
		t.Skip("set GRANTDIFF_TEST_KUBECONFIG to an isolated kind kubeconfig")
	}
	contextName := os.Getenv("GRANTDIFF_TEST_CONTEXT")
	if !strings.HasPrefix(contextName, "kind-grantdiff-") {
		t.Fatal("integration tests require an explicitly named kind-grantdiff-* context")
	}
	clientConfig, err := cluster.Config(configPath, contextName)
	if err != nil {
		t.Fatal(err)
	}
	clientConfig.QPS = 100
	clientConfig.Burst = 200
	client, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	namespace := "grantdiff-" + suffix
	prefix := "grantdiff-" + suffix
	_, err = client.CoreV1().Namespaces().Create(ctx, &core.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		_ = client.CoreV1().Namespaces().Delete(cleanup, namespace, metav1.DeleteOptions{})
		for _, name := range []string{prefix + "-aggregate", prefix + "-leaf", prefix + "-cluster"} {
			_ = client.RbacV1().ClusterRoles().Delete(cleanup, name, metav1.DeleteOptions{})
		}
		_ = client.RbacV1().ClusterRoleBindings().Delete(cleanup, prefix, metav1.DeleteOptions{})
	})
	account := model.Identity{ID: "bot", Kind: "ServiceAccount", Name: "bot", Namespace: namespace}
	user := model.Identity{ID: "alice", Kind: "User", Name: prefix + "-alice", Authenticated: true, Groups: []string{prefix + "-team"}, GroupsComplete: true}
	stranger := model.Identity{ID: "stranger", Kind: "User", Name: prefix + "-stranger", Authenticated: true, GroupsComplete: true}
	identities := []model.Identity{account, user, stranger}
	_, err = client.CoreV1().ServiceAccounts(namespace).Create(ctx, &core.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "bot"}}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.CoreV1().ConfigMaps(namespace).Create(ctx, &core.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "billing-config"}, Data: map[string]string{"sample": "safe"}}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	role := &rbac.Role{ObjectMeta: metav1.ObjectMeta{Name: "reader", Namespace: namespace}, Rules: []rbac.PolicyRule{
		{APIGroups: []string{""}, Resources: []string{"configmaps"}, Verbs: []string{"get", "list", "watch"}, ResourceNames: []string{"billing-config"}},
		{APIGroups: []string{""}, Resources: []string{"pods/log"}, Verbs: []string{"get"}},
	}}
	if _, err := client.RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	for name, subjects := range map[string][]rbac.Subject{
		"direct": {{Kind: "ServiceAccount", Name: "bot", Namespace: namespace}},
		"group":  {{Kind: "Group", Name: "system:serviceaccounts:" + namespace, APIGroup: rbac.GroupName}},
		"user":   {{Kind: "Group", Name: prefix + "-team", APIGroup: rbac.GroupName}},
	} {
		binding := &rbac.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}, Subjects: subjects, RoleRef: rbac.RoleRef{APIGroup: rbac.GroupName, Kind: "Role", Name: "reader"}}
		if _, err := client.RbacV1().RoleBindings(namespace).Create(ctx, binding, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	leaf := &rbac.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: prefix + "-leaf", Labels: map[string]string{"grantdiff-case": prefix}}, Rules: []rbac.PolicyRule{{APIGroups: []string{""}, Resources: []string{"nodes"}, Verbs: []string{"get"}}}}
	if _, err := client.RbacV1().ClusterRoles().Create(ctx, leaf, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	aggregate := &rbac.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: prefix + "-aggregate"}, AggregationRule: &rbac.AggregationRule{ClusterRoleSelectors: []metav1.LabelSelector{{MatchLabels: map[string]string{"grantdiff-case": prefix}}}}}
	if _, err := client.RbacV1().ClusterRoles().Create(ctx, aggregate, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	clusterBinding := &rbac.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: prefix}, Subjects: []rbac.Subject{{Kind: "User", Name: user.Name, APIGroup: rbac.GroupName}}, RoleRef: rbac.RoleRef{APIGroup: rbac.GroupName, Kind: "ClusterRole", Name: aggregate.Name}}
	if _, err := client.RbacV1().ClusterRoleBindings().Create(ctx, clusterBinding, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	waitAggregate(t, ctx, client, aggregate.Name, "get")
	before, _, _, err := captureStable(t, ctx, client)
	if err != nil {
		t.Fatal(err)
	}
	checkTypedExports(t, ctx, client, before, namespace, prefix)
	requests := requests(namespace)
	checks := checkRequests(t, ctx, client, before, identities, requests)
	// Named list authorization must match a real HTTP request with a selector.
	config, err := cluster.Config(configPath, contextName)
	if err != nil {
		t.Fatal(err)
	}
	config.Impersonate.UserName = account.Username()
	config.Impersonate.Groups = account.EffectiveGroups()
	impersonated, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := impersonated.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{FieldSelector: "metadata.name=billing-config"}); err != nil {
		t.Fatalf("named list should succeed: %v", err)
	}
	if _, err := impersonated.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{}); !apierrors.IsForbidden(err) {
		t.Fatalf("unfiltered list should be forbidden: %v", err)
	}
	if err := client.RbacV1().RoleBindings(namespace).Delete(ctx, "direct", metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	leaf, err = client.RbacV1().ClusterRoles().Get(ctx, leaf.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	leaf.Rules[0].Verbs = []string{"list"}
	if _, err := client.RbacV1().ClusterRoles().Update(ctx, leaf, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	waitAggregate(t, ctx, client, aggregate.Name, "list")
	after, metadata, _, err := captureStable(t, ctx, client)
	if err != nil {
		t.Fatal(err)
	}
	checks += checkRequests(t, ctx, client, after, identities, requests)
	review := diff.Compare(before, after, identities, 100000)
	if review.Scope.Calculation != "complete" {
		t.Fatalf("incomplete integration diff: %+v", review.Diagnostics)
	}
	retained, added, removed := false, false, false
	for _, change := range review.Changes {
		if change.Subject == account.Username() && change.Type == "retained_via_other_path" {
			retained = true
		}
		if change.Subject == user.Username() && change.Type == "added" && change.Request.Resource == "nodes" {
			added = true
		}
		if change.Subject == user.Username() && change.Type == "removed" && change.Request.Resource == "nodes" {
			removed = true
		}
	}
	if !retained || !added || !removed {
		t.Fatalf("missing semantic changes: retained=%v added=%v removed=%v", retained, added, removed)
	}
	checks += checkWildcardCatalog(t, ctx, client, after, stranger, namespace, prefix)
	t.Logf("Kubernetes %s: %d SubjectAccessReview comparisons, 2 actual list requests, 4 typed RBAC exports, retained + aggregate + wildcard scope verified; seed 20261119/7", metadata.KubernetesVersion, checks)
	if path := os.Getenv("GRANTDIFF_TEST_RESULT"); path != "" {
		result := map[string]any{"kubernetesVersion": metadata.KubernetesVersion, "authorizationComparisons": checks, "actualRequests": 2, "typedRBACExports": 4, "seed": "20261119/7", "result": "passed", "completedAt": time.Now().UTC()}
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func checkTypedExports(t *testing.T, ctx context.Context, client kubernetes.Interface, snapshot *model.State, namespace, prefix string) {
	t.Helper()
	files := []input.File{}
	for _, resource := range []string{"roles", "clusterroles", "rolebindings", "clusterrolebindings"} {
		data, err := client.RbacV1().RESTClient().Get().Resource(resource).SetHeader("Accept", "application/json").Do(ctx).Raw()
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, input.File{Name: resource + ".json", Data: data})
	}
	state, err := input.Parse(files, "Kubernetes API exports", input.Options{})
	if err != nil {
		t.Fatal(err)
	}
	keys := []string{
		model.Key("Role", namespace, "reader"),
		model.Key("RoleBinding", namespace, "group"),
		model.Key("ClusterRole", "", prefix+"-aggregate"),
		model.Key("ClusterRoleBinding", "", prefix),
	}
	for _, key := range keys {
		object := state.Objects[key]
		if object == nil || object.Digest() != snapshot.Objects[key].Digest() {
			t.Fatalf("API export changed or omitted %s", key)
		}
	}
}

func checkWildcardCatalog(t *testing.T, ctx context.Context, client kubernetes.Interface, before *model.State, identity model.Identity, namespace, prefix string) int {
	t.Helper()
	requests := []model.Request{
		{Verb: "get", APIGroup: "apps", Resource: "deployments", Namespace: namespace, Name: "app"},
		{Verb: "get", APIGroup: "apps", Resource: "deployments", Namespace: "other", Name: "app"},
		{Verb: "get", APIGroup: "certificates.k8s.io", Resource: "certificatesigningrequests", Name: "sample"},
		{Verb: "get", APIGroup: "certificates.k8s.io", Resource: "certificatesigningrequests", Namespace: namespace, Name: "sample"},
	}
	checks := checkRequests(t, ctx, client, before, []model.Identity{identity}, requests)
	role := &rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: prefix + "-cluster"},
		Rules:      []rbac.PolicyRule{{APIGroups: []string{"*"}, Resources: []string{"deployments", "certificatesigningrequests"}, Verbs: []string{"get"}}},
	}
	if _, err := client.RbacV1().ClusterRoles().Create(ctx, role, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	binding := &rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "wildcard", Namespace: namespace},
		Subjects:   []rbac.Subject{{Kind: "User", Name: identity.Name, APIGroup: rbac.GroupName}},
		RoleRef:    rbac.RoleRef{APIGroup: rbac.GroupName, Kind: "ClusterRole", Name: role.Name},
	}
	if _, err := client.RbacV1().RoleBindings(namespace).Create(ctx, binding, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	after, _, _, err := captureStable(t, ctx, client)
	if err != nil {
		t.Fatal(err)
	}
	checks += checkRequests(t, ctx, client, after, []model.Identity{identity}, requests)
	review := diff.Compare(before, after, []model.Identity{identity}, 100000)
	foundDeployment := false
	for _, change := range review.Changes {
		if change.Type == "indeterminate" && review.Scope.Calculation != "incomplete" {
			t.Fatal("wildcard unknown scope was marked complete")
		}
		if change.Type != "added" {
			continue
		}
		group, resource := change.Range["apiGroup"], change.Range["resource"]
		if selectorContains(group, "certificates.k8s.io") && selectorContains(resource, "certificatesigningrequests") {
			t.Fatal("namespace binding reported access to a cluster-scoped resource")
		}
		if group.Value != nil && *group.Value == "apps" && resource.Value != nil && *resource.Value == "deployments" {
			foundDeployment = true
		}
	}
	if !foundDeployment {
		t.Fatal("catalogued API group was not separated from wildcard complement")
	}
	if err := evaluate.CheckRequest(requests[3], after.Catalog); err == nil {
		t.Fatal("invalid namespaced cluster-resource query was accepted")
	}
	return checks
}

func selectorContains(selector diff.Selector, value string) bool {
	if selector.Value != nil {
		return value == *selector.Value
	}
	return !slices.Contains(selector.Except, value)
}

func waitAggregate(t *testing.T, ctx context.Context, client kubernetes.Interface, name, verb string) {
	t.Helper()
	for range 100 {
		role, err := client.RbacV1().ClusterRoles().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if len(role.Rules) == 1 && slices.Equal(role.Rules[0].Verbs, []string{verb}) {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
	t.Fatal("aggregation did not converge")
}

func requests(namespace string) []model.Request {
	result := []model.Request{
		{Verb: "get", Resource: "configmaps", Name: "billing-config", Namespace: namespace},
		{Verb: "list", Resource: "configmaps", Name: "billing-config", Namespace: namespace},
		{Verb: "list", Resource: "configmaps", Namespace: namespace},
		{Verb: "get", Resource: "nodes", Name: "sample"},
		{Verb: "list", Resource: "nodes"},
		{Verb: "get", Resource: "pods", Subresource: "log", Name: "sample", Namespace: namespace},
		{Verb: "get", URL: "/healthz"}, {Verb: "get", URL: "/api"}, {Verb: "get", URL: "/grantdiff-private"},
	}
	random := rand.New(rand.NewPCG(20261119, 7))
	verbs := []string{"get", "list", "watch", "create", "update", "delete", "patch"}
	resources := []string{"pods", "configmaps", "secrets", "nodes", "deployments"}
	names := []string{"", "billing-config", "other"}
	namespaces := []string{"", namespace, "other"}
	for range 200 {
		request := model.Request{Verb: verbs[random.IntN(len(verbs))], Resource: resources[random.IntN(len(resources))], Namespace: namespaces[random.IntN(len(namespaces))], Name: names[random.IntN(len(names))]}
		if request.Resource == "deployments" {
			request.APIGroup = "apps"
		}
		if request.Resource == "pods" && random.IntN(2) == 0 {
			request.Subresource = "log"
		}
		result = append(result, request)
	}
	return result
}

func checkRequests(t *testing.T, ctx context.Context, client kubernetes.Interface, state *model.State, identities []model.Identity, requests []model.Request) int {
	t.Helper()
	engine := evaluate.New(state)
	checks := 0
	for _, identity := range identities {
		principal := engine.Prepare(identity)
		for _, request := range requests {
			prediction := principal.Decide(request)
			actual, err := cluster.Review(ctx, client, identity, request)
			if err != nil {
				t.Fatal(err)
			}
			if prediction.Result == "unknown" || (prediction.Result == "granted") != actual {
				t.Fatalf("identity=%s request=%+v predicted=%+v actual=%v", identity.Username(), request, prediction, actual)
			}
			checks++
		}
	}
	return checks
}
