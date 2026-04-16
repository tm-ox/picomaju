package templates

import (
	"picomaju/internal/role"
	"picomaju/internal/staff"
	"picomaju/internal/tool"
	"picomaju/internal/value"
)

// SidebarData is passed to every page that uses the sidebar layout.
type SidebarData struct {
	Categories    []value.Category
	Staff         []staff.Staff
	Roles         []role.Role
	Integrations  []tool.Tool    // catalog-type tools
	Skills        []tool.Tool    // skill-type tools
	ValueCounts   map[string]int // keyed by category ID
	ActiveCat     string         // active category filter ("" = all)
	ActiveSection string         // "values" | "tools" | "roles" | "staff"
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
