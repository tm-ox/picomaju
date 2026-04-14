package sop

import "testing"

func sops(ss ...*SOP) []*SOP { return ss }

func mkSOP(id, category string, priority int) *SOP {
	return &SOP{
		ID:       id,
		Title:    id,
		Version:  1,
		Priority: priority,
		Trigger:  "on request",
		Category: category,
		Body:     "instruction for " + id,
	}
}

func TestCompile_EmptyResult(t *testing.T) {
	res, err := Compile("agent", []string{"tasks"}, nil, sops())
	if err != nil {
		t.Fatal(err)
	}
	if res.Role != "agent" {
		t.Errorf("role = %q, want %q", res.Role, "agent")
	}
	if len(res.Policies) != 0 {
		t.Errorf("expected 0 policies, got %d", len(res.Policies))
	}
}

func TestCompile_FiltersByCategory(t *testing.T) {
	all := sops(
		mkSOP("a", "tasks", 50),
		mkSOP("b", "communication", 50),
		mkSOP("c", "tasks", 30),
	)
	res, err := Compile("agent", []string{"tasks"}, nil, all)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Policies) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(res.Policies))
	}
}

func TestCompile_MultipleCategoriesIncluded(t *testing.T) {
	all := sops(
		mkSOP("a", "tasks", 50),
		mkSOP("b", "communication", 60),
		mkSOP("c", "escalation", 70),
	)
	res, err := Compile("agent", []string{"tasks", "communication"}, nil, all)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Policies) != 2 {
		t.Fatalf("expected 2 policies, got %d: %v", len(res.Policies), res.Policies)
	}
}

func TestCompile_IndividualSOPAdded(t *testing.T) {
	all := sops(
		mkSOP("a", "tasks", 50),
		mkSOP("b", "communication", 60),
	)
	// Include tasks category + individually add "b" from communication
	res, err := Compile("agent", []string{"tasks"}, []string{"b"}, all)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Policies) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(res.Policies))
	}
}

func TestCompile_DeduplicatesIndividualVsCategory(t *testing.T) {
	all := sops(mkSOP("a", "tasks", 50))
	// "a" is in tasks category AND in individual list — must appear once.
	res, err := Compile("agent", []string{"tasks"}, []string{"a"}, all)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Policies) != 1 {
		t.Errorf("expected 1 policy (deduped), got %d", len(res.Policies))
	}
}

func TestCompile_SortsByPriorityDesc(t *testing.T) {
	all := sops(
		mkSOP("low", "tasks", 10),
		mkSOP("high", "tasks", 90),
		mkSOP("mid", "tasks", 50),
	)
	res, err := Compile("agent", []string{"tasks"}, nil, all)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"high", "mid", "low"}
	for i, p := range res.Policies {
		if p.ID != want[i] {
			t.Errorf("policies[%d].ID = %q, want %q", i, p.ID, want[i])
		}
	}
}

func TestCompile_StableSortOnTie(t *testing.T) {
	all := sops(
		mkSOP("a", "tasks", 50),
		mkSOP("b", "tasks", 50),
		mkSOP("c", "tasks", 50),
	)
	res, err := Compile("agent", []string{"tasks"}, nil, all)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a", "b", "c"}
	for i, p := range res.Policies {
		if p.ID != want[i] {
			t.Errorf("policies[%d].ID = %q, want %q", i, p.ID, want[i])
		}
	}
}

func TestCompile_AbortOnInvalidSOP(t *testing.T) {
	bad := &SOP{
		ID:       "",
		Title:    "Bad",
		Category: "tasks",
	}
	_, err := Compile("agent", []string{"tasks"}, nil, sops(bad))
	if err == nil {
		t.Fatal("expected error for invalid SOP, got nil")
	}
}

func TestCompile_IndividualSOPNotFound(t *testing.T) {
	_, err := Compile("agent", nil, []string{"missing_id"}, sops())
	if err == nil {
		t.Fatal("expected error for missing individual SOP, got nil")
	}
}

func TestCompile_MapsInstructionFromBody(t *testing.T) {
	s := mkSOP("x", "tasks", 50)
	s.Body = "custom instruction text"
	res, err := Compile("agent", []string{"tasks"}, nil, sops(s))
	if err != nil {
		t.Fatal(err)
	}
	if res.Policies[0].Instruction != "custom instruction text" {
		t.Errorf("instruction = %q", res.Policies[0].Instruction)
	}
}

func TestCompile_NoCategoriesNorSOPs(t *testing.T) {
	all := sops(mkSOP("a", "tasks", 50))
	res, err := Compile("agent", nil, nil, all)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Policies) != 0 {
		t.Errorf("expected 0 policies, got %d", len(res.Policies))
	}
}
