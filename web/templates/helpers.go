package templates

import (
	"fmt"
	"picomaju/internal/staff"
	"picomaju/internal/task"
	"picomaju/internal/tool"
	"picomaju/internal/value"
)

// SidebarData is passed to every page that uses the sidebar layout.
type SidebarData struct {
	Categories    []value.Category
	Staff         []staff.Staff
	Tasks         []task.Task
	Tools         []tool.Tool    // catalog tools
	ValueCounts   map[string]int // keyed by category ID
	ActiveCat     string         // active category filter ("" = all)
	ActiveSection string         // "values" | "tools" | "tasks" | "staff"
	BusinessName  string
	HideSidebar   bool
}

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
