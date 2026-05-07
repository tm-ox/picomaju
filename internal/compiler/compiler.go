package compiler

import (
	"fmt"
	"sort"
	"strings"

	"picomaju/internal/settings"
	"picomaju/internal/staff"
	"picomaju/internal/task"
	"picomaju/internal/tool"
	"picomaju/internal/value"
)

// Input is all resolved data needed to compile one agent workspace.
type Input struct {
	Staff    *staff.Staff
	Tasks    []task.Task
	Tools    []tool.Tool
	Values   []*value.Value
	Settings *settings.Settings
}

// Output holds the three generated workspace files as strings.
type Output struct {
	AgentMD string
	SoulMD  string
	UserMD  string
}

// Compile generates AGENT.md, SOUL.md, and USER.md from resolved staff data.
func Compile(in Input) Output {
	return Output{
		AgentMD: buildAgent(in),
		SoulMD:  buildSoul(in),
		UserMD:  buildUser(in),
	}
}

func buildAgent(in Input) string {
	var b strings.Builder

	b.WriteString("---\n")
	fmt.Fprintf(&b, "name: %s\n", in.Staff.Label)
	if in.Staff.Description != "" {
		fmt.Fprintf(&b, "description: >\n  %s\n", in.Staff.Description)
	}
	b.WriteString("---\n\n")

	b.WriteString("# Role\n\n")
	if in.Staff.Description != "" {
		b.WriteString(in.Staff.Description + "\n\n")
	} else if in.Settings != nil && in.Settings.BusinessName != "" {
		fmt.Fprintf(&b, "A staff agent for %s.\n\n", in.Settings.BusinessName)
	}

	if len(in.Tasks) > 0 {
		toolIndex := make(map[string]tool.Tool, len(in.Tools))
		for _, tl := range in.Tools {
			toolIndex[tl.ID] = tl
		}

		b.WriteString("# Tasks\n\n")
		for _, t := range in.Tasks {
			fmt.Fprintf(&b, "## %s\n\n", t.Label)
			if t.Description != "" {
				b.WriteString(t.Description + "\n\n")
			}
			var labels []string
			for _, tid := range t.Tools {
				if tl, ok := toolIndex[tid]; ok {
					labels = append(labels, tl.Label)
				}
			}
			if len(labels) > 0 {
				fmt.Fprintf(&b, "**Tools:** %s\n\n", strings.Join(labels, ", "))
			}
		}
	}

	if len(in.Tools) > 0 {
		catIndex := tool.CatalogByType()
		b.WriteString("# Integrations\n\n")
		for _, tl := range in.Tools {
			if cat, ok := catIndex[tl.Type]; ok {
				fmt.Fprintf(&b, "- **%s** (%s): %s\n", tl.Label, cat.Category, cat.Description)
			} else {
				fmt.Fprintf(&b, "- **%s**\n", tl.Label)
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

func buildSoul(in Input) string {
	catOrder := []string{"core_values", "communication", "skills", "escalation", "custom"}
	catLabels := map[string]string{
		"core_values":   "Core Values",
		"communication": "Communication",
		"skills":        "Skills",
		"escalation":    "Escalation",
		"custom":        "Custom",
	}

	grouped := make(map[string][]*value.Value)
	for _, v := range in.Values {
		grouped[v.Category] = append(grouped[v.Category], v)
	}
	for cat := range grouped {
		sort.Slice(grouped[cat], func(i, j int) bool {
			return grouped[cat][i].Priority > grouped[cat][j].Priority
		})
	}

	var b strings.Builder
	hasAny := false

	for _, cat := range catOrder {
		vals := grouped[cat]
		if len(vals) == 0 {
			continue
		}
		hasAny = true
		fmt.Fprintf(&b, "# %s\n\n", catLabels[cat])
		for _, v := range vals {
			fmt.Fprintf(&b, "## %s\n\n", v.Title)
			if v.Body != "" {
				b.WriteString(strings.TrimSpace(v.Body) + "\n\n")
			}
		}
	}

	if !hasAny {
		b.WriteString("# Identity\n\nThis agent has no configured values.\n")
	}

	return b.String()
}

func buildUser(in Input) string {
	var b strings.Builder
	s := in.Settings

	b.WriteString("# Business Context\n\n")
	if s == nil {
		return b.String()
	}
	if s.BusinessName != "" {
		fmt.Fprintf(&b, "**Name:** %s\n\n", s.BusinessName)
	}
	if s.BusinessDetails != "" {
		fmt.Fprintf(&b, "**About:** %s\n\n", s.BusinessDetails)
	}
	if s.Timezone != "" {
		fmt.Fprintf(&b, "**Timezone:** %s\n\n", s.Timezone)
	}
	if s.Hours != "" {
		fmt.Fprintf(&b, "**Hours:** %s\n\n", s.Hours)
	}
	if len(s.Languages) > 0 {
		fmt.Fprintf(&b, "**Languages:** %s\n\n", strings.Join(s.Languages, ", "))
	}

	return b.String()
}
