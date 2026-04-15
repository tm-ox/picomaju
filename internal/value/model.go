package value

// Value is the in-memory representation of a single org-level directive.
// Frontmatter fields map 1:1 to YAML keys on disk; Body holds the raw instruction text.
type Value struct {
	ID       string `yaml:"id"       json:"id"`
	Title    string `yaml:"title"    json:"title"`
	Version  int    `yaml:"version"  json:"version"`
	Priority int    `yaml:"priority" json:"priority"`
	Category string `yaml:"category" json:"category"`
	Body     string `yaml:"-"        json:"body,omitempty"`
}

// DirectiveEntry is one entry in a compiled staff directive.
// Compiler output format is TBD (AGENTS.md, SOUL.md, etc.) — this is a placeholder.
type DirectiveEntry struct {
	ID          string `json:"id"`
	Version     int    `json:"version"`
	Priority    int    `json:"priority"`
	Instruction string `json:"instruction"`
}

// ValidationError describes a single field-level validation failure.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationResult is the output of a validation pass over one Value.
type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []string          `json:"warnings,omitempty"`
}
