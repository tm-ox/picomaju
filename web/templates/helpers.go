package templates

import (
	"encoding/json"
	"picomaju/internal/category"
	"picomaju/internal/role"
	"picomaju/internal/sop"
)

// SidebarData is passed to every page that uses the sidebar layout.
type SidebarData struct {
	Categories   []category.Category
	Roles        []role.Role
	SOPCounts    map[string]int // keyed by category ID
	ActiveCat    string         // active category filter ("" = all)
	BusinessName string
}

func compileResultJSON(res *sop.CompileResult) string {
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}

func includesStr(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
