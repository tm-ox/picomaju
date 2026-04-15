package value

import "fmt"

// Validate checks required fields and clamps priority to [0, 100].
// Priority clamping mutates the Value in place and emits a warning.
func Validate(v *Value) ValidationResult {
	var result ValidationResult

	if v.ID == "" {
		result.Errors = append(result.Errors, ValidationError{Field: "id", Message: "required"})
	}
	if v.Title == "" {
		result.Errors = append(result.Errors, ValidationError{Field: "title", Message: "required"})
	}
	if v.Version == 0 {
		result.Errors = append(result.Errors, ValidationError{Field: "version", Message: "required"})
	}
	if v.Category == "" {
		result.Errors = append(result.Errors, ValidationError{Field: "category", Message: "required"})
	}

	if v.Priority < 0 || v.Priority > 100 {
		orig := v.Priority
		if v.Priority < 0 {
			v.Priority = 0
		} else {
			v.Priority = 100
		}
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"priority %d out of range [0, 100]; clamped to %d", orig, v.Priority,
		))
	}

	result.Valid = len(result.Errors) == 0
	return result
}
