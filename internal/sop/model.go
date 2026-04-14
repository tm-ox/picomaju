package sop

// SOP is the in-memory representation of a single Standard Operating Procedure.
// Frontmatter fields map 1:1 to YAML keys on disk; Body holds the raw NL instruction.
type SOP struct {
	ID         string                 `yaml:"id"         json:"id"`
	Title      string                 `yaml:"title"      json:"title"`
	Version    int                    `yaml:"version"    json:"version"`
	Priority   int                    `yaml:"priority"   json:"priority"`
	Trigger    string                 `yaml:"trigger"    json:"trigger"`
	Category   string                 `yaml:"category"   json:"category"`
	Conditions map[string]interface{} `yaml:"conditions,omitempty" json:"conditions,omitempty"`
	Body       string                 `yaml:"-"          json:"body,omitempty"`
}

// Policy is one entry in a compiled Role Manifest's policies array.
type Policy struct {
	ID          string                 `json:"id"`
	Version     int                    `json:"version"`
	Priority    int                    `json:"priority"`
	Trigger     string                 `json:"trigger"`
	Conditions  map[string]interface{} `json:"conditions,omitempty"`
	Instruction string                 `json:"instruction"`
}

// ValidationError describes a single field-level validation failure.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationResult is the output of a validation pass over one SOP.
type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []string          `json:"warnings,omitempty"`
}

// CompileResult is the assembled output for a single role.
type CompileResult struct {
	Role     string   `json:"role"`
	Policies []Policy `json:"policies"`
	Warnings []string `json:"warnings,omitempty"`
}
