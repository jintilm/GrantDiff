package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jintilm/GrantDiff/internal/cluster"
	"github.com/jintilm/GrantDiff/internal/diff"
	"github.com/jintilm/GrantDiff/internal/evaluate"
	"github.com/jintilm/GrantDiff/internal/model"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type generatedScenario struct {
	Seed            uint64                    `json:"seed"`
	Namespace       string                    `json:"namespace"`
	Identities      []model.Identity          `json:"identities"`
	Roles           []rbac.Role               `json:"roles"`
	ClusterRoles    []rbac.ClusterRole        `json:"clusterRoles"`
	Bindings        []rbac.RoleBinding        `json:"roleBindings"`
	ClusterBindings []rbac.ClusterRoleBinding `json:"clusterRoleBindings"`
	Requests        []model.Request           `json:"requests"`
}

func TestGeneratedAuthorizationCombinations(t *testing.T) {
	configPath := os.Getenv("GRANTDIFF_TEST_KUBECONFIG")
	if configPath == "" {
		t.Skip("set GRANTDIFF_TEST_KUBECONFIG to run generated policy cases")
	}
	contextName := os.Getenv("GRANTDIFF_TEST_CONTEXT")
	if !strings.HasPrefix(contextName, "kind-grantdiff-") {
		t.Fatal("generated cases require a kind-grantdiff-* test context")
	}
	config, err := cluster.Config(configPath, contextName)
	if err != nil {
		t.Fatal(err)
	}
	config.QPS, config.Burst = 100, 200
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	seeds := []uint64{17, 73, 101, 307, 997, 65537, 20261119, 20261120}
	if value := os.Getenv("GRANTDIFF_TEST_SEED"); value != "" {
		seed, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		seeds = []uint64{seed}
	}
	version, err := client.Discovery().ServerVersion()
	if err != nil {
		t.Fatal(err)
	}
	checks, diffChecks, allowed, denied := 0, 0, 0, 0
	for _, seed := range seeds {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			scenario := generateScenario(seed)
			createScenario(t, ctx, client, scenario)
			before, _, _, err := cluster.Capture(ctx, client)
			if err != nil {
				t.Fatal(err)
			}
			old, beforeAllowed := compareGeneratedRequests(t, ctx, client, before, scenario)
			allowed += beforeAllowed
			denied += len(old) - beforeAllowed
			role, err := client.RbacV1().Roles(scenario.Namespace).Get(ctx, scenario.Roles[0].Name, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}
			role.Rules = []rbac.PolicyRule{{APIGroups: []string{""}, Resources: []string{"secrets"}, Verbs: []string{"get", "list"}}}
			if _, err := client.RbacV1().Roles(scenario.Namespace).Update(ctx, role, metav1.UpdateOptions{}); err != nil {
				t.Fatal(err)
			}
			if err := client.RbacV1().RoleBindings(scenario.Namespace).Delete(ctx, scenario.Bindings[1].Name, metav1.DeleteOptions{}); err != nil {
				t.Fatal(err)
			}
			after, _, _, err := cluster.Capture(ctx, client)
			if err != nil {
				t.Fatal(err)
			}
			current, afterAllowed := compareGeneratedRequests(t, ctx, client, after, scenario)
			allowed += afterAllowed
			denied += len(current) - afterAllowed
			checks += len(old) + len(current)
			review := diff.Compare(before, after, scenario.Identities, 100000)
			for identityIndex, identity := range scenario.Identities {
				for requestIndex, request := range scenario.Requests {
					index := identityIndex*len(scenario.Requests) + requestIndex
					expected := ""
					if !old[index] && current[index] {
						expected = "added"
					}
					if old[index] && !current[index] {
						expected = "removed"
					}
					matches := []diff.Change{}
					for _, change := range review.Changes {
						if change.Subject == identity.Username() && rangeContains(change.Range, request) && (change.Type == "added" || change.Type == "removed" || change.Type == "indeterminate") {
							matches = append(matches, change)
						}
					}
					if expected == "" && len(matches) != 0 || expected != "" && (len(matches) != 1 || matches[0].Type != expected) {
						writeScenarioFailure(t, scenario, before, after, identity, request, review)
						t.Fatalf("seed=%d subject=%s request=%+v expected=%q matched=%+v", seed, identity.Username(), request, expected, matches)
					}
					diffChecks++
				}
			}
		})
	}
	if t.Failed() {
		return
	}
	if allowed == 0 || denied == 0 {
		t.Fatal("generated suite must exercise allowed and denied decisions")
	}
	t.Logf("Kubernetes %s: seeds=%v authorization comparisons=%d (allowed=%d denied=%d), diff checks=%d", version.GitVersion, seeds, checks, allowed, denied, diffChecks)
	if path := os.Getenv("GRANTDIFF_TEST_RESULT"); path != "" {
		result := map[string]any{"kubernetesVersion": version.GitVersion, "seeds": seeds, "authorizationComparisons": checks, "diffChecks": diffChecks, "allowed": allowed, "denied": denied, "result": "passed", "completedAt": time.Now().UTC()}
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(strings.TrimSuffix(path, filepath.Ext(path))+"-randomized.json", data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func generateScenario(seed uint64) generatedScenario {
	random := rand.New(rand.NewPCG(seed, 41))
	prefix := fmt.Sprintf("grantdiff-seed-%d", seed)
	s := generatedScenario{Seed: seed, Namespace: prefix}
	s.Identities = []model.Identity{
		{ID: "alice", Kind: "User", Name: prefix + "-alice", Authenticated: true, Groups: []string{prefix + "-team"}, GroupsComplete: true},
		{ID: "bob", Kind: "User", Name: prefix + "-bob", Authenticated: true, GroupsComplete: true},
		{ID: "bot", Kind: "ServiceAccount", Name: "bot", Namespace: prefix},
		{ID: "outsider", Kind: "User", Name: prefix + "-outsider", Authenticated: true, GroupsComplete: true},
	}
	if random.IntN(2) == 0 {
		s.Identities[1].Groups = []string{prefix + "-team"}
	}
	subjects := []rbac.Subject{
		{Kind: "User", Name: s.Identities[0].Name, APIGroup: rbac.GroupName},
		{Kind: "User", Name: s.Identities[1].Name, APIGroup: rbac.GroupName},
		{Kind: "Group", Name: prefix + "-team", APIGroup: rbac.GroupName},
		{Kind: "ServiceAccount", Name: "bot", Namespace: prefix},
		{Kind: "Group", Name: "system:serviceaccounts:" + prefix, APIGroup: rbac.GroupName},
	}
	for index := range 4 {
		name := fmt.Sprintf("role-%d", index)
		role := rbac.Role{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: prefix}}
		for count := 1 + random.IntN(3); count > 0; count-- {
			role.Rules = append(role.Rules, generatedRule(random))
		}
		s.Roles = append(s.Roles, role)
		s.Bindings = append(s.Bindings, rbac.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "access-" + name, Namespace: prefix}, Subjects: []rbac.Subject{subjects[random.IntN(len(subjects))]}, RoleRef: rbac.RoleRef{APIGroup: rbac.GroupName, Kind: "Role", Name: name}})
	}
	// Always give Alice one changed path, alongside the generated group paths.
	s.Bindings[0].Subjects = append(s.Bindings[0].Subjects, subjects[0])
	for index := range 2 {
		name := fmt.Sprintf("%s-cluster-%d", prefix, index)
		s.ClusterRoles = append(s.ClusterRoles, rbac.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: name}, Rules: []rbac.PolicyRule{generatedRule(random), generatedRule(random)}})
		s.ClusterBindings = append(s.ClusterBindings, rbac.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: name}, Subjects: []rbac.Subject{subjects[random.IntN(len(subjects))]}, RoleRef: rbac.RoleRef{APIGroup: rbac.GroupName, Kind: "ClusterRole", Name: name}})
	}
	s.Bindings = append(s.Bindings, rbac.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "cluster-in-namespace", Namespace: prefix}, Subjects: []rbac.Subject{subjects[3]}, RoleRef: rbac.RoleRef{APIGroup: rbac.GroupName, Kind: "ClusterRole", Name: s.ClusterRoles[0].Name}})
	for _, resource := range []string{"configmaps", "secrets", "pods", "pods/log", "nodes", "deployments"} {
		for _, verb := range []string{"get", "list", "patch", "create"} {
			request := model.Request{Verb: verb, Resource: resource, Namespace: prefix, Name: "demo"}
			if resource == "nodes" {
				request.Namespace = ""
			}
			if resource == "deployments" {
				request.APIGroup = "apps"
			}
			if resource == "pods/log" {
				request.Resource, request.Subresource = "pods", "log"
				if verb != "get" {
					continue
				}
			}
			if verb == "list" || verb == "create" {
				request.Name = ""
			}
			s.Requests = append(s.Requests, request)
			if resource != "nodes" {
				request.Namespace = "other"
				s.Requests = append(s.Requests, request)
			}
		}
	}
	s.Requests = append(s.Requests, model.Request{Verb: "list", Resource: "configmaps", Namespace: prefix, Name: "demo"})
	return s
}

func generatedRule(random *rand.Rand) rbac.PolicyRule {
	resources := []string{"configmaps", "secrets", "pods", "pods/log", "*/log", "nodes", "deployments", "*"}
	resource := resources[random.IntN(len(resources))]
	group := ""
	if resource == "deployments" {
		group = "apps"
	}
	verbs := []string{"get", "list", "watch", "patch", "create", "*"}
	rule := rbac.PolicyRule{APIGroups: []string{group}, Resources: []string{resource}, Verbs: []string{verbs[random.IntN(len(verbs))]}}
	if random.IntN(3) == 0 {
		rule.ResourceNames = []string{"demo"}
	}
	return rule
}

func createScenario(t *testing.T, ctx context.Context, client kubernetes.Interface, scenario generatedScenario) {
	t.Helper()
	if _, err := client.CoreV1().Namespaces().Create(ctx, &core.Namespace{ObjectMeta: metav1.ObjectMeta{Name: scenario.Namespace}}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := client.CoreV1().Namespaces().Delete(cleanup, scenario.Namespace, metav1.DeleteOptions{}); err != nil {
			t.Errorf("cleanup namespace: %v", err)
		}
	})
	for _, role := range scenario.Roles {
		if _, err := client.RbacV1().Roles(scenario.Namespace).Create(ctx, &role, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, role := range scenario.ClusterRoles {
		if _, err := client.RbacV1().ClusterRoles().Create(ctx, &role, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			cleanup, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := client.RbacV1().ClusterRoles().Delete(cleanup, role.Name, metav1.DeleteOptions{}); err != nil {
				t.Errorf("cleanup ClusterRole: %v", err)
			}
		})
	}
	for _, binding := range scenario.Bindings {
		if _, err := client.RbacV1().RoleBindings(scenario.Namespace).Create(ctx, &binding, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, binding := range scenario.ClusterBindings {
		if _, err := client.RbacV1().ClusterRoleBindings().Create(ctx, &binding, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			cleanup, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := client.RbacV1().ClusterRoleBindings().Delete(cleanup, binding.Name, metav1.DeleteOptions{}); err != nil {
				t.Errorf("cleanup ClusterRoleBinding: %v", err)
			}
		})
	}
}

func compareGeneratedRequests(t *testing.T, ctx context.Context, client kubernetes.Interface, state *model.State, scenario generatedScenario) ([]bool, int) {
	t.Helper()
	engine := evaluate.New(state)
	results := []bool{}
	allowed := 0
	for _, identity := range scenario.Identities {
		principal := engine.Prepare(identity)
		for _, request := range scenario.Requests {
			actual, err := cluster.Review(ctx, client, identity, request)
			if err != nil {
				t.Fatal(err)
			}
			predicted := principal.Decide(request)
			if predicted.Result == "unknown" || (predicted.Result == "granted") != actual {
				writeScenarioFailure(t, scenario, state, state, identity, request, diff.Report{})
				t.Fatalf("seed=%d subject=%s request=%+v predicted=%s actual=%v", scenario.Seed, identity.Username(), request, predicted.Result, actual)
			}
			results = append(results, actual)
			if actual {
				allowed++
			}
		}
	}
	return results, allowed
}

func rangeContains(region diff.Region, request model.Request) bool {
	if request.URL == "" && region["url"].Value != nil {
		return false
	}
	values := map[string]string{"verb": request.Verb, "apiGroup": request.APIGroup, "resource": request.Resource, "subresource": request.Subresource, "namespace": request.Namespace, "name": request.Name}
	for dimension, value := range values {
		selector, ok := region[dimension]
		if !ok {
			return false
		}
		if selector.Value != nil && *selector.Value != value || selector.Value == nil && slices.Contains(selector.Except, value) {
			return false
		}
	}
	return true
}

func writeScenarioFailure(t *testing.T, scenario generatedScenario, before, after *model.State, identity model.Identity, request model.Request, review diff.Report) {
	t.Helper()
	directory := os.Getenv("GRANTDIFF_FAILURE_DIR")
	if directory == "" {
		directory = filepath.Join("..", ".work", "failures")
	}
	directory = filepath.Join(directory, fmt.Sprintf("seed-%d", scenario.Seed))
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Logf("write failure fixture: %v", err)
		return
	}
	files := map[string]any{
		"scenario.json":   scenario,
		"before.json":     map[string]any{"apiVersion": "v1", "kind": "List", "items": before.SortedObjects()},
		"after.json":      map[string]any{"apiVersion": "v1", "kind": "List", "items": after.SortedObjects()},
		"catalog.json":    before.Catalog,
		"identities.json": model.Identities{SchemaVersion: "1.0", Identities: scenario.Identities},
		"request.json":    map[string]any{"identity": identity, "request": request},
		"report.json":     review,
	}
	for name, value := range files {
		data, err := json.MarshalIndent(value, "", "  ")
		if err == nil {
			err = os.WriteFile(filepath.Join(directory, name), data, 0600)
		}
		if err != nil {
			t.Logf("write %s: %v", name, err)
		}
	}
	t.Logf("Failure fixture %s; replay with GRANTDIFF_TEST_SEED=%d and go run ./scripts/integration", directory, scenario.Seed)
}
