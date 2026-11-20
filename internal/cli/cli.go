package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jintilm/GrantDiff/internal/cluster"
	"github.com/jintilm/GrantDiff/internal/diff"
	"github.com/jintilm/GrantDiff/internal/evaluate"
	"github.com/jintilm/GrantDiff/internal/input"
	"github.com/jintilm/GrantDiff/internal/model"
	"github.com/jintilm/GrantDiff/internal/report"
)

var Version = "1.0.0-rc.1"

type options struct {
	before, after, repo, base, head, path, baseline, changes               string
	input, identities, catalog, identity, format, output, defaultNamespace string
	context, kubeconfig, failOn                                            string
	allowBefore, allowAfter                                                bool
	maxClasses                                                             int
	summaryOnly                                                            bool
	compact                                                                bool
	request                                                                model.Request
}

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || slices.Contains([]string{"help", "--help", "-h"}, args[0]) {
		fmt.Fprintln(stdout, "grantdiff diff | query | catalog | snapshot | version\n\nReview Kubernetes RBAC changes with explicit authorization evidence.\nUse COMMAND --help for flags.")
		return 0
	}
	if args[0] == "version" || args[0] == "--version" {
		fmt.Fprintln(stdout, Version)
		return 0
	}
	if !slices.Contains([]string{"diff", "query", "catalog", "snapshot"}, args[0]) {
		fmt.Fprintln(stderr, "unknown command:", args[0])
		return 2
	}
	opts, err := parse(args[0], args[1:], stderr)
	if err == flag.ErrHelp {
		return 0
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if args[0] == "snapshot" {
		return snapshot(opts, stdout, stderr)
	}
	if args[0] == "query" {
		return query(opts, stdout, stderr)
	}
	if args[0] == "catalog" {
		return catalog(opts, stdout, stderr)
	}
	return compare(opts, stdout, stderr)
}

func parse(command string, args []string, stderr io.Writer) (options, error) {
	var opts options
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&opts.output, "output", "", "write report or snapshot to path")
	if command == "snapshot" {
		flags.StringVar(&opts.context, "context", "", "explicit Kubernetes context")
		flags.StringVar(&opts.kubeconfig, "kubeconfig", "", "trusted kubeconfig file")
	} else if command == "catalog" {
		flags.StringVar(&opts.input, "input", "", "CRD manifests to turn into a resource catalog")
		flags.StringVar(&opts.catalog, "catalog", "", "optional base catalog to merge")
	} else {
		flags.StringVar(&opts.identities, "identities", "", "identity definitions (otherwise infer subjects)")
		flags.StringVar(&opts.catalog, "catalog", "", "resource catalog JSON or YAML")
		flags.StringVar(&opts.format, "format", "text", "text, markdown, or json")
		flags.StringVar(&opts.defaultNamespace, "default-namespace", "", "explicit namespace for unqualified objects")
	}
	if command == "query" {
		flags.StringVar(&opts.input, "input", "", "complete manifest file or directory")
		flags.StringVar(&opts.identity, "identity", "", "identity id from identities file")
		flags.StringVar(&opts.request.Verb, "verb", "", "authorization verb")
		flags.StringVar(&opts.request.APIGroup, "api-group", "", "API group (empty for core)")
		flags.StringVar(&opts.request.Resource, "resource", "", "resource name, such as configmaps")
		flags.StringVar(&opts.request.Subresource, "subresource", "", "subresource, such as log")
		flags.StringVar(&opts.request.Namespace, "namespace", "", "request namespace")
		flags.StringVar(&opts.request.Name, "name", "", "object name")
		flags.StringVar(&opts.request.URL, "url", "", "non-resource URL path")
	}
	if command == "diff" {
		flags.StringVar(&opts.before, "before", "", "complete before manifests")
		flags.StringVar(&opts.after, "after", "", "complete after manifests")
		flags.StringVar(&opts.repo, "repo", "", "local Git repository")
		flags.StringVar(&opts.base, "base", "", "base Git commit")
		flags.StringVar(&opts.head, "head", "", "head Git commit")
		flags.StringVar(&opts.path, "path", ".", "manifest directory within Git tree")
		flags.StringVar(&opts.baseline, "baseline", "", "snapshot directory")
		flags.StringVar(&opts.changes, "changes", "", "explicit candidate changes file")
		flags.StringVar(&opts.failOn, "fail-on", "", "added, removed, or high")
		flags.BoolVar(&opts.allowBefore, "allow-empty-before", false, "declare an empty before state")
		flags.BoolVar(&opts.allowAfter, "allow-empty-after", false, "declare an empty after state")
		flags.IntVar(&opts.maxClasses, "max-classes", 1000000, "maximum request classes evaluated")
		flags.BoolVar(&opts.summaryOnly, "summary", false, "show the review summary without detailed changes (text or markdown)")
		flags.BoolVar(&opts.compact, "compact", false, "merge compatible request ranges with identical decisions, risks and evidence")
	}
	if err := flags.Parse(args); err != nil {
		return opts, err
	}
	if flags.NArg() != 0 {
		return opts, fmt.Errorf("unexpected positional arguments")
	}
	if command != "snapshot" && command != "catalog" && !slices.Contains([]string{"text", "markdown", "json"}, opts.format) {
		return opts, fmt.Errorf("invalid format %q", opts.format)
	}
	if opts.summaryOnly && opts.format == "json" {
		return opts, fmt.Errorf("--summary supports text and markdown; JSON always includes full changes")
	}
	return opts, nil
}

func compare(opts options, stdout, stderr io.Writer) int {
	if opts.maxClasses < 1 || !slices.Contains([]string{"", "added", "removed", "high"}, opts.failOn) {
		fmt.Fprintln(stderr, "invalid comparison budget or --fail-on")
		return 2
	}
	before, after, err := states(opts)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	identities, err := identitySet(opts.identities, before, after)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if opts.catalog != "" {
		catalog, err := input.LoadCatalog(opts.catalog)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		before.Catalog, err = model.MergeCatalogs(before.Catalog, catalog)
		if err == nil {
			after.Catalog, err = model.MergeCatalogs(after.Catalog, catalog)
		}
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
	}
	result := diff.Compare(before, after, identities, opts.maxClasses)
	if opts.compact {
		result = diff.Compact(result)
	}
	var data []byte
	if opts.summaryOnly {
		data, err = report.RenderSummary(result, opts.format)
	} else {
		data, err = report.Render(result, opts.format)
	}
	if err == nil {
		err = report.Write(opts.output, data, stdout)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if result.Scope.Calculation != "complete" {
		return 3
	}
	for _, change := range result.Changes {
		if opts.failOn == change.Type {
			return 1
		}
		if opts.failOn == "high" && len(change.Risk) > 0 {
			return 1
		}
	}
	return 0
}

func states(opts options) (*model.State, *model.State, error) {
	modes := 0
	if opts.before != "" || opts.after != "" {
		modes++
	}
	if opts.repo != "" || opts.base != "" || opts.head != "" {
		modes++
	}
	if opts.baseline != "" || opts.changes != "" {
		modes++
	}
	if modes != 1 {
		return nil, nil, fmt.Errorf("select exactly one input pair: --before/--after, --repo/--base/--head, or --baseline/--changes")
	}
	leftOpts := input.Options{DefaultNamespace: opts.defaultNamespace, AllowEmpty: opts.allowBefore}
	rightOpts := input.Options{DefaultNamespace: opts.defaultNamespace, AllowEmpty: opts.allowAfter}
	var before, after *model.State
	var err error
	switch {
	case opts.repo != "":
		before, err = input.Git(opts.repo, opts.base, opts.path, leftOpts)
		if err == nil {
			after, err = input.Git(opts.repo, opts.head, opts.path, rightOpts)
		}
	case opts.baseline != "":
		before, err = loadState(opts.baseline, leftOpts)
		if err == nil {
			after, err = input.ApplyChanges(before, opts.changes, rightOpts)
		}
	default:
		if opts.before == "" || opts.after == "" {
			return nil, nil, fmt.Errorf("both --before and --after are required")
		}
		before, err = loadState(opts.before, leftOpts)
		if err == nil {
			after, err = loadState(opts.after, rightOpts)
		}
	}
	return before, after, err
}

func loadState(path string, opts input.Options) (*model.State, error) {
	if _, err := os.Stat(filepath.Join(path, "metadata.json")); err == nil {
		return cluster.Load(path, opts)
	}
	return input.Load(path, opts)
}

func identitySet(path string, states ...*model.State) ([]model.Identity, error) {
	if path != "" {
		return input.LoadIdentities(path)
	}
	return input.InferIdentities(states...), nil
}

func query(opts options, stdout, stderr io.Writer) int {
	if opts.input == "" || opts.identity == "" {
		fmt.Fprintln(stderr, "--input and --identity are required")
		return 2
	}
	state, err := loadState(opts.input, input.Options{DefaultNamespace: opts.defaultNamespace})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	identities, err := identitySet(opts.identities, state)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	index := slices.IndexFunc(identities, func(identity model.Identity) bool { return identity.ID == opts.identity })
	if index < 0 {
		fmt.Fprintln(stderr, "identity not found:", opts.identity)
		return 2
	}
	if opts.catalog != "" {
		var catalog model.Catalog
		catalog, err = input.LoadCatalog(opts.catalog)
		if err == nil {
			state.Catalog, err = model.MergeCatalogs(state.Catalog, catalog)
		}
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
	}
	decision := evaluate.New(state).Prepare(identities[index]).Decide(opts.request)
	code := 0
	if err := evaluate.CheckRequest(opts.request, state.Catalog); err != nil {
		decision = model.Decision{Result: "unknown", Evidence: []model.Evidence{}, Diagnostics: []model.Diagnostic{{Code: "invalid_or_unknown_request", Message: err.Error()}}}
	}
	if decision.Result == "unknown" {
		code = 3
	}
	value := struct {
		SchemaVersion string         `json:"schemaVersion"`
		Subject       string         `json:"subject"`
		Request       model.Request  `json:"request"`
		Decision      model.Decision `json:"decision"`
	}{"1.0", identities[index].Username(), opts.request, decision}
	data, err := json.MarshalIndent(value, "", "  ")
	if opts.format == "text" {
		data = []byte(fmt.Sprintf("%s — %s\n%s\n", decision.Result, identities[index].Username(), opts.request.String()))
		for _, evidence := range decision.Evidence {
			data = append(data, []byte(fmt.Sprintf("%s → %s → %s\n", evidence.Subject, evidence.Binding, evidence.Role))...)
		}
		for _, diagnostic := range decision.Diagnostics {
			data = append(data, []byte(diagnostic.Code+" — "+diagnostic.Message+"\n")...)
		}
	}
	if opts.format == "markdown" {
		data = []byte("```json\n" + strings.ReplaceAll(string(data), "`", "\\u0060") + "\n```\n")
	}
	if opts.format == "json" {
		data = append(data, '\n')
	}
	if err == nil {
		err = report.Write(opts.output, data, stdout)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	return code
}

func catalog(opts options, stdout, stderr io.Writer) int {
	if opts.input == "" {
		fmt.Fprintln(stderr, "catalog requires --input with CRD manifests")
		return 2
	}
	state, err := input.Load(opts.input, input.Options{AllowEmpty: true})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if len(state.Catalog.Resources) == 0 {
		fmt.Fprintln(stderr, "no served CRD resources found in input")
		return 2
	}
	if opts.catalog != "" {
		var base model.Catalog
		base, err = input.LoadCatalog(opts.catalog)
		if err == nil {
			state.Catalog, err = model.MergeCatalogs(base, state.Catalog)
		}
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
	}
	data, err := json.MarshalIndent(state.Catalog, "", "  ")
	if err == nil {
		err = report.Write(opts.output, append(data, '\n'), stdout)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	return 0
}

func snapshot(opts options, stdout, stderr io.Writer) int {
	if opts.output == "" || opts.context == "" {
		fmt.Fprintln(stderr, "snapshot requires --context and --output")
		return 2
	}
	client, err := cluster.Connect(opts.kubeconfig, opts.context)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 4
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	state, metadata, data, err := cluster.Capture(ctx, client)
	if err == nil {
		err = cluster.Save(opts.output, state, metadata, data)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 4
	}
	fmt.Fprintf(stdout, "Snapshot %s\n%d RBAC objects\nDigest %s\n", opts.output, len(state.Objects), metadata.Digest)
	return 0
}
