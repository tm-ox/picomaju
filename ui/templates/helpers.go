package templates

import (
	"fmt"
	"picomaju/internal/value"
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
