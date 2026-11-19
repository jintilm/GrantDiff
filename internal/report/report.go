package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	if format != "text" && format != "markdown" {
		return nil, fmt.Errorf("format must be text, markdown, or json")
	}
	var out bytes.Buffer
	fmt.Fprintf(&out, "GrantDiff — %s\n\n", report.Scope.Calculation)
	fmt.Fprintf(&out, "Scope %s · %d identities · %d compared classes · %d changes\n\n", report.Scope.Source, report.Identities, report.Compared, len(report.Changes))
	for _, change := range report.Changes {
		fmt.Fprintf(&out, "%s  %s  %s\n", change.Type, escape(change.Subject), change.TargetType)
		fmt.Fprintf(&out, "  %s → %s\n  %s\n", change.Before, change.After, escape(regionString(change.Range)))
		for _, side := range []struct {
			label string
			paths []model.Evidence
		}{{"Before", change.BeforeEvidence}, {"After", change.AfterEvidence}} {
			for _, evidence := range side.paths {
				fmt.Fprintf(&out, "  %s  %s → %s → %s (%s:%d)\n", side.label, escape(evidence.Subject), escape(evidence.Binding), escape(evidence.Role), escape(evidence.RuleSource.File), evidence.RuleSource.Line)
				if len(evidence.AggregationPath) > 0 {
					fmt.Fprintf(&out, "    Aggregation %s\n", escape(strings.Join(evidence.AggregationPath, " → ")))
				}
			}
		}
		for _, risk := range change.Risk {
			fmt.Fprintf(&out, "  %s %s — %s\n", risk.Severity, risk.ID, risk.Message)
		}
		fmt.Fprintln(&out)
	}
	for _, diagnostic := range report.Diagnostics {
		fmt.Fprintf(&out, "%s — %s %s\n", diagnostic.Code, escape(diagnostic.Message), escape(diagnostic.Object))
	}
	if format == "markdown" {
		// Indented blocks keep all untrusted object names as literal text.
		var markdown bytes.Buffer
		for _, line := range strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n") {
			fmt.Fprintf(&markdown, "    %s\n", line)
		}
		return markdown.Bytes(), nil
	}
	return out.Bytes(), nil
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
