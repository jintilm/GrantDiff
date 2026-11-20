// Command benchmark measures a fixed synthetic review without a running cluster.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/jintilm/GrantDiff/internal/diff"
	"github.com/jintilm/GrantDiff/internal/input"
	"github.com/jintilm/GrantDiff/internal/model"
	"github.com/jintilm/GrantDiff/internal/report"
	rbac "k8s.io/api/rbac/v1"
)

func main() {
	mode := flag.String("mode", "memory", "memory or files (load, analyze, render and write)")
	scenario := flag.String("scenario", "sparse", "sparse or wildcard workload")
	compact := flag.Bool("compact", false, "compact equivalent result ranges")
	budget := flag.Int("max-classes", 1000000, "maximum request classes evaluated")
	flag.Parse()
	if err := run(*mode, *scenario, *compact, *budget); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(mode, scenario string, compact bool, budget int) error {
	if mode != "memory" && mode != "files" {
		return fmt.Errorf("mode must be memory or files")
	}
	if scenario != "sparse" && scenario != "wildcard" || budget < 1 {
		return fmt.Errorf("scenario must be sparse or wildcard and max-classes must be positive")
	}
	var before, after *model.State
	var identities []model.Identity
	if scenario == "wildcard" {
		before, after, identities = wildcardScenario()
	} else {
		before, after, identities = sparseScenario()
	}
	catalog := before.Catalog
	directory := ""
	if mode == "files" {
		var err error
		directory, err = os.MkdirTemp("", "grantdiff-benchmark-")
		if err != nil {
			return err
		}
		defer func() {
			for _, name := range []string{"before.json", "after.json", "identities.json", "catalog.json", "report.json"} {
				_ = os.Remove(filepath.Join(directory, name))
			}
			_ = os.Remove(directory)
		}()
		files := map[string]any{
			"before.json":     map[string]any{"apiVersion": "v1", "kind": "List", "items": before.SortedObjects()},
			"after.json":      map[string]any{"apiVersion": "v1", "kind": "List", "items": after.SortedObjects()},
			"identities.json": model.Identities{SchemaVersion: "1.0", Identities: identities},
			"catalog.json":    catalog,
		}
		for name, value := range files {
			data, err := json.Marshal(value)
			if err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(directory, name), data, 0600); err != nil {
				return err
			}
		}
	}
	runtime.GC()
	var initial, final runtime.MemStats
	runtime.ReadMemStats(&initial)
	start := time.Now()
	if mode == "files" {
		var err error
		before, err = input.Load(filepath.Join(directory, "before.json"), input.Options{})
		if err != nil {
			return err
		}
		after, err = input.Load(filepath.Join(directory, "after.json"), input.Options{})
		if err != nil {
			return err
		}
		identities, err = input.LoadIdentities(filepath.Join(directory, "identities.json"))
		if err != nil {
			return err
		}
		catalog, err = input.LoadCatalog(filepath.Join(directory, "catalog.json"))
		if err != nil {
			return err
		}
		before.Catalog, after.Catalog = catalog, catalog
	}
	loaded := time.Now()
	result := diff.Compare(before, after, identities, budget)
	compared := time.Now()
	rawChanges := len(result.Changes)
	if compact {
		result = diff.Compact(result)
	}
	compacted := time.Now()
	reportBytes := 0
	if mode == "files" {
		data, err := report.Render(result, "json")
		if err != nil {
			return err
		}
		reportBytes = len(data)
		if err := report.Write(filepath.Join(directory, "report.json"), data, os.Stdout); err != nil {
			return err
		}
	}
	finished := time.Now()
	runtime.ReadMemStats(&final)
	output := map[string]any{
		"goVersion": runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH, "mode": mode, "scenario": scenario,
		"objectsPerState": len(before.Objects), "identities": len(identities), "changedObjects": 1, "compact": compact, "maxClasses": budget,
		"elapsedMilliseconds": finished.Sub(start).Milliseconds(), "loadMilliseconds": loaded.Sub(start).Milliseconds(),
		"compareMilliseconds": compared.Sub(loaded).Milliseconds(), "compactMilliseconds": compacted.Sub(compared).Milliseconds(), "outputMilliseconds": finished.Sub(compacted).Milliseconds(),
		"totalAllocatedBytes": final.TotalAlloc - initial.TotalAlloc, "heapInUseBytesAtEnd": final.HeapInuse,
		"comparedClasses": result.Compared, "rawChanges": rawChanges, "changes": len(result.Changes), "reportBytes": reportBytes, "calculation": result.Scope.Calculation,
		"measurement": "memory mode times analysis; files mode includes input loading and report writing; heap sample at end, process peak RSS not measured",
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return err
	}
	if result.Scope.Calculation == "incomplete" && result.Compared == budget {
		return nil
	}
	if result.Scope.Calculation != "complete" || scenario == "sparse" && len(result.Changes) != 2 || scenario == "wildcard" && len(result.Changes) == 0 {
		return fmt.Errorf("unexpected benchmark result")
	}
	return nil
}

func sparseScenario() (*model.State, *model.State, []model.Identity) {
	before, after := model.NewState(), model.NewState()
	identities := []model.Identity{}
	catalog := model.Catalog{SchemaVersion: "1.0", Resources: []model.Resource{{Name: "configmaps", Namespaced: true}}}
	before.Catalog, after.Catalog = catalog, catalog
	for index := 0; index < 1000; index++ {
		identity := model.Identity{ID: fmt.Sprintf("bot-%d", index), Kind: "ServiceAccount", Name: fmt.Sprintf("bot-%d", index), Namespace: "benchmark"}
		identities = append(identities, identity)
		for offset := 0; offset < 5; offset++ {
			name := fmt.Sprintf("reader-%d-%d", index, offset)
			role := &model.Object{APIVersion: rbac.SchemeGroupVersion.String(), Kind: "Role", Metadata: model.Metadata{Name: name, Namespace: "benchmark"}, Rules: []rbac.PolicyRule{{APIGroups: []string{""}, Resources: []string{"configmaps"}, ResourceNames: []string{name}, Verbs: []string{"get"}}}}
			binding := &model.Object{APIVersion: rbac.SchemeGroupVersion.String(), Kind: "RoleBinding", Metadata: model.Metadata{Name: name, Namespace: "benchmark"}, Subjects: []rbac.Subject{{Kind: "ServiceAccount", Name: identity.Name, Namespace: identity.Namespace}}, RoleRef: rbac.RoleRef{APIGroup: rbac.GroupName, Kind: "Role", Name: name}}
			before.Objects[role.Key()], after.Objects[role.Key()] = role, role
			before.Objects[binding.Key()], after.Objects[binding.Key()] = binding, binding
		}
	}
	key := "Role/benchmark/reader-0-0"
	changed := *after.Objects[key]
	rule := changed.Rules[0]
	rule.Verbs = []string{"get", "list"}
	changed.Rules = []rbac.PolicyRule{rule}
	after.Objects[key] = &changed
	return before, after, identities
}

func wildcardScenario() (*model.State, *model.State, []model.Identity) {
	before, after := model.NewState(), model.NewState()
	catalog := model.Catalog{SchemaVersion: "1.0", Resources: []model.Resource{
		{Name: "secrets", Namespaced: true}, {Name: "configmaps", Namespaced: true},
		{Name: "pods", Namespaced: true, Subresources: []string{"log", "exec", "status"}},
		{Name: "nodes", Namespaced: false, Subresources: []string{"proxy", "status"}},
	}}
	for index := range 16 {
		catalog.Resources = append(catalog.Resources, model.Resource{APIGroup: "benchmark.example", Name: fmt.Sprintf("widgets-%d", index), Namespaced: index%4 != 0, Subresources: []string{"status", "scale"}})
	}
	before.Catalog, after.Catalog = catalog, catalog
	role := &model.Object{APIVersion: rbac.SchemeGroupVersion.String(), Kind: "ClusterRole", Metadata: model.Metadata{Name: "shared"}, Rules: []rbac.PolicyRule{
		{APIGroups: []string{""}, Resources: []string{"configmaps"}, Verbs: []string{"get"}, ResourceNames: []string{"selected"}},
		{APIGroups: []string{"benchmark.example"}, Resources: []string{"widgets-0"}, Verbs: []string{"get"}, ResourceNames: []string{"selected"}},
	}}
	before.Objects[role.Key()] = role
	changed := *role
	changed.Rules = []rbac.PolicyRule{{APIGroups: []string{"benchmark.example"}, Resources: []string{"*"}, Verbs: []string{"*"}}, {APIGroups: []string{""}, Resources: []string{"configmaps", "secrets", "pods", "pods/log", "pods/exec", "nodes", "nodes/proxy"}, Verbs: []string{"get", "list", "watch", "create", "patch"}}}
	after.Objects[role.Key()] = &changed
	identities := []model.Identity{}
	for index := range 25 {
		identity := model.Identity{ID: fmt.Sprintf("bot-%d", index), Kind: "ServiceAccount", Name: fmt.Sprintf("bot-%d", index), Namespace: "benchmark"}
		identities = append(identities, identity)
		binding := &model.Object{APIVersion: rbac.SchemeGroupVersion.String(), Kind: "ClusterRoleBinding", Metadata: model.Metadata{Name: identity.ID}, Subjects: []rbac.Subject{{Kind: "ServiceAccount", Name: identity.Name, Namespace: identity.Namespace}}, RoleRef: rbac.RoleRef{APIGroup: rbac.GroupName, Kind: "ClusterRole", Name: role.Metadata.Name}}
		before.Objects[binding.Key()], after.Objects[binding.Key()] = binding, binding
	}
	return before, after, identities
}
