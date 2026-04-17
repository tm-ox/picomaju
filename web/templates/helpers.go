package templates

import (
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
