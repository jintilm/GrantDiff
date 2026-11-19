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
	flag.Parse()
	if err := run(*mode); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(mode string) error {
	if mode != "memory" && mode != "files" {
		return fmt.Errorf("mode must be memory or files")
	}
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
	result := diff.Compare(before, after, identities, 1000000)
	if mode == "files" {
		data, err := report.Render(result, "json")
		if err != nil {
			return err
		}
		if err := report.Write(filepath.Join(directory, "report.json"), data, os.Stdout); err != nil {
			return err
		}
	}
	elapsed := time.Since(start)
	runtime.ReadMemStats(&final)
	output := map[string]any{"goVersion": runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH, "mode": mode, "objectsPerState": len(before.Objects), "identities": len(identities), "changedObjects": 1, "elapsedMilliseconds": elapsed.Milliseconds(), "totalAllocatedBytes": final.TotalAlloc - initial.TotalAlloc, "heapInUseBytesAtEnd": final.HeapInuse, "comparedClasses": result.Compared, "changes": len(result.Changes), "calculation": result.Scope.Calculation, "measurement": "memory mode times analysis; files mode includes input loading and report writing; heap sample at end, process peak RSS not measured"}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return err
	}
	if result.Scope.Calculation != "complete" || len(result.Changes) != 2 {
		return fmt.Errorf("unexpected benchmark result")
	}
	return nil
}
