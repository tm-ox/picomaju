package valuestpl

import "picomaju/internal/value"

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
