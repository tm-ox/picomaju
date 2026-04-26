package shell

import (
	"fmt"
	"picomaju/internal/value"
)

func IncludesStr(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func FormTitle(noun string, isNew bool) string {
	if isNew {
		return "New " + noun
	}
	return "Edit " + noun
}

func FormAction(base, id string, isNew bool) string {
	if isNew {
		return base
	}
	return base + "/" + id
}

func CountWord(n int, singular, plural string) string {
	if n == 0 {
		return "—"
	}
	if n == 1 {
		return "1 " + singular
	}
	return fmt.Sprintf("%d %s", n, plural)
}

func CategoryLabel(cats []value.Category, id string) string {
	for _, c := range cats {
		if c.ID == id {
			return c.Label
		}
	}
	return id
}

func CatLabel(cat string) string {
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
