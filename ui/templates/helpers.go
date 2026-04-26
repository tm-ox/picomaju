package templates

import (
	"fmt"
	"picomaju/internal/staff"
	"picomaju/internal/task"
	"picomaju/internal/tool"
	"picomaju/internal/value"
	"strings"
	"unicode"
)

func includesStr(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func formTitle(noun string, isNew bool) string {
	if isNew {
		return "New " + noun
	}
	return "Edit " + noun
}

func formAction(base, id string, isNew bool) string {
	if isNew {
		return base
	}
	return base + "/" + id
}

func countWord(n int, singular, plural string) string {
	if n == 0 {
		return "—"
	}
	if n == 1 {
		return "1 " + singular
	}
	return fmt.Sprintf("%d %s", n, plural)
}

func categoryLabel(cats []value.Category, id string) string {
	for _, c := range cats {
		if c.ID == id {
			return c.Label
		}
	}
	return id
}

func configValue(cfg map[string]any, key string) string {
	if cfg == nil {
		return ""
	}
	v, ok := cfg[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func staffInitials(label string) string {
	fields := strings.Fields(strings.TrimSpace(label))
	var out []rune
	for _, f := range fields {
		for _, r := range f {
			if unicode.IsLetter(r) {
				out = append(out, unicode.ToUpper(r))
				break
			}
		}
		if len(out) >= 2 {
			break
		}
	}
	if len(out) == 0 {
		return "?"
	}
	return string(out)
}

func staffSectionTitle(section string) string {
	switch section {
	case "overview":
		return "Overview"
	case "profile":
		return "Profile"
	case "values":
		return "Values"
	case "tools":
		return "Tools"
	case "tasks":
		return "Tasks"
	default:
		return "Overview"
	}
}

func staffActiveLabel(active bool) string {
	if active {
		return "Active"
	}
	return "Inactive"
}

// staffToolCount returns the number of unique tool IDs reachable via the staff member's assigned tasks.
func staffToolCount(m *staff.Staff, tasks []task.Task) int {
	seen := map[string]bool{}
	for _, t := range tasks {
		if includesStr(m.Tasks, t.ID) {
			for _, toolID := range t.Tools {
				seen[toolID] = true
			}
		}
	}
	return len(seen)
}

// catLabel returns the display label for a tool catalog category, or "" if unknown.
func catLabel(cat string) string {
	switch cat {
	case "messaging":
		return "Messaging"
	case "commerce":
		return "Commerce"
	case "payments":
		return "Payments"
	case "utilities":
		return "Utilities"
	default:
		return ""
	}
}

type valCatGroup struct {
	Cat    value.Category
	Values []*value.Value
}

func groupValuesByCat(cats []value.Category, vals []*value.Value) []valCatGroup {
	index := make(map[string][]*value.Value)
	for _, v := range vals {
		index[v.Category] = append(index[v.Category], v)
	}
	var groups []valCatGroup
	for _, c := range cats {
		if items := index[c.ID]; len(items) > 0 {
			groups = append(groups, valCatGroup{Cat: c, Values: items})
		}
	}
	return groups
}

type toolCatGroup struct {
	Label string
	Tools []tool.Tool
}

func groupToolsByCat(tools []tool.Tool) []toolCatGroup {
	catIndex := tool.CatalogByType()
	order := []string{"messaging", "commerce", "payments", "utilities"}
	index := make(map[string][]tool.Tool)
	for _, t := range tools {
		if integ, ok := catIndex[t.Type]; ok {
			index[integ.Category] = append(index[integ.Category], t)
		}
	}
	var groups []toolCatGroup
	for _, cat := range order {
		if items := index[cat]; len(items) > 0 {
			groups = append(groups, toolCatGroup{Label: catLabel(cat), Tools: items})
		}
	}
	return groups
}

func staffIconOptions() []string {
	return []string{
		"user", "user-round", "bot", "headphones", "phone", "mail",
		"calendar", "clipboard", "briefcase", "calculator",
		"cpu", "database", "globe", "key", "megaphone", "message-circle",
		"monitor", "package", "pencil", "shield", "star",
		"truck", "wrench", "zap", "wallet", "award", "coffee", "flag",
	}
}
