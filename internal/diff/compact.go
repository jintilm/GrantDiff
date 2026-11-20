package diff

import (
	"maps"
	"slices"
	"strings"

	"github.com/jintilm/GrantDiff/internal/model"
)

type compactChange struct {
	change    Change
	signature string
}

// Compact joins exact cells with their complement when every other dimension,
// decision, risk and evidence location agrees. The union remains expressible in
// schema 1.0. Each representative request stays inside its original cell.
func Compact(report Report) Report {
	changes := make([]compactChange, 0, len(report.Changes))
	for _, change := range report.Changes {
		semantics := change
		semantics.ID, semantics.Range, semantics.Request = "", nil, model.Request{}
		changes = append(changes, compactChange{change: change, signature: model.Hash(semantics)})
	}
	slices.SortFunc(changes, func(a, b compactChange) int {
		return strings.Compare(a.change.Subject+"/"+a.change.ID, b.change.Subject+"/"+b.change.ID)
	})
	for {
		previous := len(changes)
		for _, dimension := range []string{"name", "namespace", "subresource", "resource", "apiGroup", "verb", "url"} {
			changes = compactDimension(changes, dimension)
		}
		if len(changes) == previous {
			break
		}
	}
	report.Changes = make([]Change, 0, len(changes))
	for _, item := range changes {
		change := item.change
		change.ID = model.Hash([]any{change.TargetType, change.Subject, change.Type, change.Range})[:20]
		report.Changes = append(report.Changes, change)
	}
	slices.SortFunc(report.Changes, func(a, b Change) int { return strings.Compare(a.Subject+"/"+a.ID, b.Subject+"/"+b.ID) })
	return report
}

func compactDimension(changes []compactChange, dimension string) []compactChange {
	groups := map[string][]compactChange{}
	keys := []string{}
	result := make([]compactChange, 0, len(changes))
	for _, item := range changes {
		if _, ok := item.change.Range[dimension]; !ok {
			result = append(result, item)
			continue
		}
		rest := maps.Clone(item.change.Range)
		delete(rest, dimension)
		key := item.signature + model.Hash(rest)
		if _, exists := groups[key]; !exists {
			keys = append(keys, key)
		}
		groups[key] = append(groups[key], item)
	}
	for _, key := range keys {
		group := groups[key]
		exacts := map[string]int{}
		for index, item := range group {
			if value := item.change.Range[dimension].Value; value != nil {
				exacts[*value] = index
			}
		}
		absorbed := map[int]bool{}
		for index, item := range group {
			selector := item.change.Range[dimension]
			if selector.Value != nil || len(selector.Except) == 0 {
				continue
			}
			remaining := []string{}
			for _, value := range selector.Except {
				exactIndex, exists := exacts[value]
				inside := selector.Prefix == nil || strings.HasPrefix(value, *selector.Prefix)
				for _, prefix := range selector.ExceptPrefixes {
					inside = inside && !strings.HasPrefix(value, prefix)
				}
				if exists && inside && !absorbed[exactIndex] {
					absorbed[exactIndex] = true
				} else {
					remaining = append(remaining, value)
				}
			}
			if len(remaining) != len(selector.Except) {
				selector.Except = remaining
				item.change.Range = maps.Clone(item.change.Range)
				item.change.Range[dimension] = selector
				group[index] = item
			}
		}
		for index, item := range group {
			if !absorbed[index] {
				result = append(result, item)
			}
		}
	}
	return result
}
