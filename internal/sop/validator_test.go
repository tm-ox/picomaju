package sop

import "testing"

func TestValidate(t *testing.T) {
	base := func() *SOP {
		return &SOP{
			ID:       "test_sop",
			Title:    "Test SOP",
			Version:  1,
			Priority: 50,
			Trigger:  "on request",
			Category: "tasks",
			Body:     "Do the thing.",
		}
	}

	tests := []struct {
		name     string
		mutate   func(*SOP)
		wantOK   bool
		wantErrs []string
		wantWarn bool
	}{
		{
			name:   "valid sop",
			mutate: func(s *SOP) {},
			wantOK: true,
		},
		{
			name:     "missing id",
			mutate:   func(s *SOP) { s.ID = "" },
			wantOK:   false,
			wantErrs: []string{"id"},
		},
		{
			name:     "missing title",
			mutate:   func(s *SOP) { s.Title = "" },
			wantOK:   false,
			wantErrs: []string{"title"},
		},
		{
			name:     "missing version",
			mutate:   func(s *SOP) { s.Version = 0 },
			wantOK:   false,
			wantErrs: []string{"version"},
		},
		{
			name:     "missing trigger",
			mutate:   func(s *SOP) { s.Trigger = "" },
			wantOK:   false,
			wantErrs: []string{"trigger"},
		},
		{
			name:     "missing category",
			mutate:   func(s *SOP) { s.Category = "" },
			wantOK:   false,
			wantErrs: []string{"category"},
		},
		{
			name:     "multiple missing fields",
			mutate:   func(s *SOP) { s.ID = ""; s.Title = ""; s.Trigger = "" },
			wantOK:   false,
			wantErrs: []string{"id", "title", "trigger"},
		},
		{
			name:     "priority below range clamps and warns",
			mutate:   func(s *SOP) { s.Priority = -10 },
			wantOK:   true,
			wantWarn: true,
		},
		{
			name:     "priority above range clamps and warns",
			mutate:   func(s *SOP) { s.Priority = 150 },
			wantOK:   true,
			wantWarn: true,
		},
		{
			name:   "priority at boundary 0",
			mutate: func(s *SOP) { s.Priority = 0 },
			wantOK: true,
		},
		{
			name:   "priority at boundary 100",
			mutate: func(s *SOP) { s.Priority = 100 },
			wantOK: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := base()
			tc.mutate(s)
			res := Validate(s)

			if res.Valid != tc.wantOK {
				t.Errorf("Valid = %v, want %v", res.Valid, tc.wantOK)
			}
			for _, field := range tc.wantErrs {
				found := false
				for _, e := range res.Errors {
					if e.Field == field {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error for field %q, got errors: %v", field, res.Errors)
				}
			}
			if tc.wantWarn && len(res.Warnings) == 0 {
				t.Error("expected warnings, got none")
			}
		})
	}
}

func TestValidate_PriorityClamping(t *testing.T) {
	s := &SOP{
		ID: "x", Title: "X", Version: 1, Priority: -5,
		Trigger: "t", Category: "tasks",
	}
	Validate(s)
	if s.Priority != 0 {
		t.Errorf("priority = %d, want 0", s.Priority)
	}

	s.Priority = 200
	Validate(s)
	if s.Priority != 100 {
		t.Errorf("priority = %d, want 100", s.Priority)
	}
}
