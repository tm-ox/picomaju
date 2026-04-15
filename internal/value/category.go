package value

// Category is a grouping label for Values.
type Category struct {
	ID    string
	Label string
}

// DefaultCategories are the built-in value categories.
var DefaultCategories = []Category{
	{ID: "core_values", Label: "Core Values"},
	{ID: "communication", Label: "Communication"},
	{ID: "skills", Label: "Skills"},
	{ID: "escalation", Label: "Escalation"},
	{ID: "custom", Label: "Custom"},
}
