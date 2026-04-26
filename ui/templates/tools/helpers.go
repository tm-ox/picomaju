package toolstpl

import (
	"picomaju/internal/tool"
	"picomaju/ui/templates/shell"
)

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
			groups = append(groups, toolCatGroup{Label: shell.CatLabel(cat), Tools: items})
		}
	}
	return groups
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
