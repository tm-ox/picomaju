package value

import (
	"strings"
	"testing"
)

func validValue() *Value {
	return &Value{ID: "v1", Title: "Be Honest", Version: 1, Category: "core_values", Priority: 50}
}

func TestValidate_Valid(t *testing.T) {
	r := Validate(validValue())
	if !r.Valid {
		t.Errorf("expected valid, got errors: %v", r.Errors)
	}
	if len(r.Warnings) != 0 {
		t.Errorf("expected no warnings, got: %v", r.Warnings)
	}
}

func TestValidate_MissingID(t *testing.T) {
	v := validValue()
	v.ID = ""
	r := Validate(v)
	if r.Valid {
		t.Error("expected invalid with missing ID")
	}
	if !hasFieldError(r, "id") {
		t.Error("expected error on field 'id'")
	}
}

func TestValidate_MissingTitle(t *testing.T) {
	v := validValue()
	v.Title = ""
	r := Validate(v)
	if r.Valid {
		t.Error("expected invalid with missing title")
	}
	if !hasFieldError(r, "title") {
		t.Error("expected error on field 'title'")
	}
}

func TestValidate_MissingVersion(t *testing.T) {
	v := validValue()
	v.Version = 0
	r := Validate(v)
	if r.Valid {
		t.Error("expected invalid with version=0")
	}
	if !hasFieldError(r, "version") {
		t.Error("expected error on field 'version'")
	}
}

func TestValidate_MissingCategory(t *testing.T) {
	v := validValue()
	v.Category = ""
	r := Validate(v)
	if r.Valid {
		t.Error("expected invalid with missing category")
	}
	if !hasFieldError(r, "category") {
		t.Error("expected error on field 'category'")
	}
}

func TestValidate_AllFieldsMissing(t *testing.T) {
	r := Validate(&Value{})
	if r.Valid {
		t.Error("expected invalid")
	}
	if len(r.Errors) != 4 {
		t.Errorf("expected 4 errors, got %d: %v", len(r.Errors), r.Errors)
	}
}

func TestValidate_PriorityClampHigh(t *testing.T) {
	v := validValue()
	v.Priority = 150
	r := Validate(v)
	if !r.Valid {
		t.Error("expected valid after clamping")
	}
	if v.Priority != 100 {
		t.Errorf("expected priority clamped to 100, got %d", v.Priority)
	}
	if len(r.Warnings) == 0 {
		t.Error("expected a warning for out-of-range priority")
	}
	if !strings.Contains(r.Warnings[0], "150") {
		t.Errorf("warning should mention original value 150, got: %s", r.Warnings[0])
	}
}

func TestValidate_PriorityClampLow(t *testing.T) {
	v := validValue()
	v.Priority = -5
	r := Validate(v)
	if !r.Valid {
		t.Error("expected valid after clamping")
	}
	if v.Priority != 0 {
		t.Errorf("expected priority clamped to 0, got %d", v.Priority)
	}
	if len(r.Warnings) == 0 {
		t.Error("expected a warning for out-of-range priority")
	}
	if !strings.Contains(r.Warnings[0], "-5") {
		t.Errorf("warning should mention original value -5, got: %s", r.Warnings[0])
	}
}

func TestValidate_PriorityBoundaryValues(t *testing.T) {
	for _, p := range []int{0, 1, 99, 100} {
		v := validValue()
		v.Priority = p
		r := Validate(v)
		if !r.Valid {
			t.Errorf("priority %d should be valid", p)
		}
		if len(r.Warnings) != 0 {
			t.Errorf("priority %d should produce no warnings", p)
		}
	}
}

func hasFieldError(r ValidationResult, field string) bool {
	for _, e := range r.Errors {
		if e.Field == field {
			return true
		}
	}
	return false
}
