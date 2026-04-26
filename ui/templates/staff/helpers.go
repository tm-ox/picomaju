package stafftpl

import (
	"picomaju/internal/staff"
	"picomaju/internal/task"
	"picomaju/ui/templates/shell"
	"strings"
	"unicode"
)

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

func staffActiveLabel(active bool) string {
	if active {
		return "Active"
	}
	return "Inactive"
}

func staffToolCount(m *staff.Staff, tasks []task.Task) int {
	seen := map[string]bool{}
	for _, t := range tasks {
		if shell.IncludesStr(m.Tasks, t.ID) {
			for _, toolID := range t.Tools {
				seen[toolID] = true
			}
		}
	}
	return len(seen)
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
