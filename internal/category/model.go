package category

const Uncategorized = "uncategorized"

// Category is a named grouping for SOPs.
// System categories are seeded on first run and cannot be deleted.
type Category struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	System bool   `json:"system"`
}

// Defaults is the starter set of categories seeded when no categories.json exists.
var Defaults = []Category{
	{ID: "business_objectives", Label: "Core Values",  System: true},
	{ID: "communication",       Label: "Communication", System: true},
	{ID: "tasks",               Label: "Skills",        System: true},
	{ID: "escalation",          Label: "Escalation",    System: true},
	{ID: Uncategorized,         Label: "Custom",        System: true},
}
