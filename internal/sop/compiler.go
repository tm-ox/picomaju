package sop

import (
	"fmt"
	"sort"
	"strings"
)

// Compile assembles a Role Manifest from a role definition's category and SOP selections.
//
// Pipeline:
//  1. Collect all SOPs whose category is in categoryIDs (bulk inclusion).
//  2. Add individually selected SOPs (individualSOPIDs), deduplicating against step 1.
//  3. Validate all collected SOPs — any failure aborts compilation.
//  4. Sort descending by priority (stable to preserve load order on ties).
//  5. Assemble policies[].
//
// v1: primary output is the in-memory CompileResult. JSON manifest write is v2.
func Compile(roleID string, categoryIDs []string, individualSOPIDs []string, allSOPs []*SOP) (*CompileResult, error) {
	// Build lookup for fast category membership test.
	catSet := make(map[string]bool, len(categoryIDs))
	for _, c := range categoryIDs {
		catSet[c] = true
	}

	// Build SOP index for individual lookups.
	sopIndex := make(map[string]*SOP, len(allSOPs))
	for _, s := range allSOPs {
		sopIndex[s.ID] = s
	}

	seen := make(map[string]bool)
	var filtered []*SOP

	// 1. Bulk inclusion from categories.
	for _, s := range allSOPs {
		if catSet[s.Category] && !seen[s.ID] {
			filtered = append(filtered, s)
			seen[s.ID] = true
		}
	}

	// 2. Individual SOPs — deduplicate against category-included set.
	for _, id := range individualSOPIDs {
		if seen[id] {
			continue
		}
		s, ok := sopIndex[id]
		if !ok {
			return nil, fmt.Errorf("individual SOP %q not found", id)
		}
		filtered = append(filtered, s)
		seen[id] = true
	}

	// 3. Validate (gate) — abort on first invalid SOP.
	var allWarnings []string
	for _, s := range filtered {
		res := Validate(s)
		if !res.Valid {
			var msgs []string
			for _, e := range res.Errors {
				msgs = append(msgs, fmt.Sprintf("%s: %s", e.Field, e.Message))
			}
			return nil, fmt.Errorf("SOP %q failed validation: %s", s.ID, strings.Join(msgs, "; "))
		}
		allWarnings = append(allWarnings, res.Warnings...)
	}

	// 4. Sort descending by priority (stable).
	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].Priority > filtered[j].Priority
	})

	// 5. Assemble.
	policies := make([]Policy, 0, len(filtered))
	for _, s := range filtered {
		policies = append(policies, Policy{
			ID:          s.ID,
			Version:     s.Version,
			Priority:    s.Priority,
			Trigger:     s.Trigger,
			Conditions:  s.Conditions,
			Instruction: s.Body,
		})
	}

	return &CompileResult{
		Role:     roleID,
		Policies: policies,
		Warnings: allWarnings,
	}, nil
}
