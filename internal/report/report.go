package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/jintilm/GrantDiff/internal/diff"
	"github.com/jintilm/GrantDiff/internal/model"
)

func Render(report diff.Report, format string) ([]byte, error) {
	if format == "json" {
		data, err := json.MarshalIndent(report, "", "  ")
		return append(data, '\n'), err
	}
	return renderReadable(report, format, false)
}

func RenderSummary(report diff.Report, format string) ([]byte, error) {
	return renderReadable(report, format, true)
}

func renderReadable(report diff.Report, format string, summaryOnly bool) ([]byte, error) {
	if format != "text" && format != "markdown" {
		return nil, fmt.Errorf("format must be text, markdown, or json")
	}
	var out bytes.Buffer
	writeSummary(&out, report, format)
	if !summaryOnly {
		for _, change := range report.Changes {
			var detail bytes.Buffer
			if format == "markdown" {
				fmt.Fprintf(&out, "<details>\n<summary>%s · %s · %s</summary>\n\n", html.EscapeString(escape(change.Type)), html.EscapeString(escape(change.Subject)), change.ID)
			}
			writeChange(&detail, change)
			if format == "markdown" {
				for _, line := range strings.Split(strings.TrimSuffix(detail.String(), "\n"), "\n") {
					if line == "" {
						fmt.Fprintln(&out)
					} else {
						fmt.Fprintf(&out, "    %s\n", line)
					}
				}
				fmt.Fprint(&out, "\n</details>\n\n")
			} else {
				out.Write(detail.Bytes())
			}
		}
	}
	for _, diagnostic := range report.Diagnostics {
		line := fmt.Sprintf("%s — %s %s", diagnostic.Code, diagnostic.Message, diagnostic.Object)
		if format == "markdown" {
			fmt.Fprintf(&out, "    %s\n", escape(line))
		} else {
			fmt.Fprintln(&out, escape(line))
		}
	}
	return append(bytes.TrimRight(out.Bytes(), "\n"), '\n'), nil
}

func writeChange(out *bytes.Buffer, change diff.Change) {
	fmt.Fprintf(out, "%s  %s  %s\n", change.Type, escape(change.Subject), change.TargetType)
	fmt.Fprintf(out, "  %s → %s\n  %s\n", change.Before, change.After, escape(regionString(change.Range)))
	for _, side := range []struct {
		label string
		paths []model.Evidence
	}{{"Before", change.BeforeEvidence}, {"After", change.AfterEvidence}} {
		for _, evidence := range side.paths {
			fmt.Fprintf(out, "  %s  %s → %s → %s (%s:%d)\n", side.label, escape(evidence.Subject), escape(evidence.Binding), escape(evidence.Role), escape(evidence.RuleSource.File), evidence.RuleSource.Line)
			if len(evidence.AggregationPath) > 0 {
				fmt.Fprintf(out, "    Aggregation %s\n", escape(strings.Join(evidence.AggregationPath, " → ")))
			}
		}
	}
	for _, risk := range change.Risk {
		fmt.Fprintf(out, "  %s %s — %s\n", risk.Severity, risk.ID, risk.Message)
	}
	fmt.Fprintln(out)
}

type subjectSummary struct {
	subject    string
	targetType string
	counts     map[string]int
	high       int
	sources    map[string]bool
}

func writeSummary(out *bytes.Buffer, report diff.Report, format string) {
	rows := map[string]*subjectSummary{}
	totals := map[string]int{}
	high := 0
	for _, change := range report.Changes {
		key := change.TargetType + "/" + change.Subject
		if rows[key] == nil {
			rows[key] = &subjectSummary{subject: change.Subject, targetType: change.TargetType, counts: map[string]int{}, sources: map[string]bool{}}
		}
		row := rows[key]
		row.counts[change.Type]++
		totals[change.Type]++
		if slices.ContainsFunc(change.Risk, func(risk diff.Risk) bool { return risk.Severity == "high" }) {
			row.high++
			high++
		}
		if change.Type == "added" {
			for _, evidence := range change.AfterEvidence {
				row.sources[evidence.Binding+" → "+evidence.Role] = true
			}
		}
	}
	keys := []string{}
	for key := range rows {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	if format == "markdown" {
		fmt.Fprintf(out, "**GrantDiff — %s**\n\n", markdownText(report.Scope.Calculation))
	} else {
		fmt.Fprintf(out, "GrantDiff — %s\n\n", escape(report.Scope.Calculation))
	}
	fmt.Fprintf(out, "Reviewed %d identities · %d affected identities or groups · %d compared request classes\n\n", report.Identities, len(rows), report.Compared)
	fmt.Fprintf(out, "Added %d · Removed %d · Retained via other path %d · Path changed %d · Indeterminate %d · High-risk ranges %d\n\n", totals["added"], totals["removed"], totals["retained_via_other_path"], totals["path_changed"], totals["indeterminate"], high)
	fmt.Fprintln(out, "Counts describe changed request ranges, including ranges that cover many object names.")
	fmt.Fprintln(out)
	if format == "markdown" {
		fmt.Fprintf(out, "Scope %s · Policies %s · Identity groups %s\n\n", markdownText(report.Scope.Source), markdownText(report.Scope.PolicyCompleteness), markdownText(report.Scope.IdentityCompleteness))
		if len(rows) > 0 {
			fmt.Fprintln(out, "| Identity or group | Result type | Added | Removed | Retained | Path changed | Unknown | High-risk |\n| --- | --- | --- | --- | --- | --- | --- | --- |")
		}
	} else {
		fmt.Fprintf(out, "Scope %s · Policies %s · Identity groups %s\n\n", escape(report.Scope.Source), escape(report.Scope.PolicyCompleteness), escape(report.Scope.IdentityCompleteness))
	}
	for _, key := range keys {
		row := rows[key]
		if format == "markdown" {
			fmt.Fprintf(out, "| %s | %s | %d | %d | %d | %d | %d | %d |\n", markdownText(row.subject), row.targetType, row.counts["added"], row.counts["removed"], row.counts["retained_via_other_path"], row.counts["path_changed"], row.counts["indeterminate"], row.high)
		} else {
			fmt.Fprintf(out, "%s (%s) — added %d, removed %d, retained %d, path changed %d, unknown %d, high-risk %d\n", escape(row.subject), row.targetType, row.counts["added"], row.counts["removed"], row.counts["retained_via_other_path"], row.counts["path_changed"], row.counts["indeterminate"], row.high)
		}
	}
	fmt.Fprintln(out)
	for _, key := range keys {
		row := rows[key]
		sources := []string{}
		for source := range row.sources {
			sources = append(sources, source)
		}
		slices.Sort(sources)
		for _, source := range sources {
			if format == "markdown" {
				fmt.Fprintf(out, "    Added access for %s through %s\n", escape(row.subject), escape(source))
			} else {
				fmt.Fprintf(out, "Added access for %s through %s\n", escape(row.subject), escape(source))
			}
		}
	}
	fmt.Fprintln(out)
}

func markdownText(value string) string {
	value = html.EscapeString(escape(value))
	return strings.NewReplacer("\\", "\\\\", "|", "\\|", "`", "\\`", "[", "\\[", "]", "\\]", "*", "\\*", "_", "\\_").Replace(value)
}
func regionString(region diff.Region) string {
	keys := []string{}
	for key := range region {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := []string{}
	for _, key := range keys {
		selector := region[key]
		if selector.Value != nil {
			parts = append(parts, fmt.Sprintf("%s=%q", key, *selector.Value))
			continue
		}
		value := "all"
		if selector.Prefix != nil {
			value = fmt.Sprintf("prefix(%q)", *selector.Prefix)
		}
		if len(selector.Except) > 0 {
			value += fmt.Sprintf(" except %q", selector.Except)
		}
		if len(selector.ExceptPrefixes) > 0 {
			value += fmt.Sprintf(" except prefixes %q", selector.ExceptPrefixes)
		}
		parts = append(parts, key+"="+value)
	}
	return strings.Join(parts, " ")
}
func escape(text string) string {
	return strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return ' '
		}
		return r
	}, text)
}

func Write(path string, data []byte, stdout io.Writer) error {
	if path == "" {
		_, err := stdout.Write(data)
		return err
	}
	directory := filepath.Dir(path)
	file, err := os.CreateTemp(directory, ".grantdiff-*")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
