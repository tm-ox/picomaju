package templates

import (
	"picomaju/internal/role"
	"picomaju/internal/staff"
	"picomaju/internal/value"
)

// SidebarData is passed to every page that uses the sidebar layout.
type SidebarData struct {
	Categories    []value.Category
	Staff         []staff.Staff
	Roles         []role.Role
	ValueCounts   map[string]int // keyed by category ID
	ActiveCat     string         // active category filter ("" = all)
	ActiveSection string         // "values" | "tools" | "roles" | "staff"
	BusinessName  string
}

func includesStr(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
