package sop

import "fmt"

// Validate checks required fields and clamps priority to [0, 100].
// Priority clamping mutates the SOP in place and emits a warning.
func Validate(s *SOP) ValidationResult {
	var result ValidationResult

	if s.ID == "" {
		result.Errors = append(result.Errors, ValidationError{Field: "id", Message: "required"})
	}
	if s.Title == "" {
		result.Errors = append(result.Errors, ValidationError{Field: "title", Message: "required"})
	}
	if s.Version == 0 {
		result.Errors = append(result.Errors, ValidationError{Field: "version", Message: "required"})
	}
	if s.Trigger == "" {
		result.Errors = append(result.Errors, ValidationError{Field: "trigger", Message: "required"})
	}
	if s.Category == "" {
		result.Errors = append(result.Errors, ValidationError{Field: "category", Message: "required"})
	}

	if s.Priority < 0 || s.Priority > 100 {
		orig := s.Priority
		if s.Priority < 0 {
			s.Priority = 0
		} else {
			s.Priority = 100
		}
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"priority %d out of range [0, 100]; clamped to %d", orig, s.Priority,
		))
	}

	result.Valid = len(result.Errors) == 0
	return result
}
