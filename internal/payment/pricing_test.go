package payment

import "testing"

func TestCreditPackByID_Found(t *testing.T) {
	for _, p := range CreditPacks {
		got, ok := CreditPackByID(p.ID)
		if !ok {
			t.Errorf("CreditPackByID(%q): not found", p.ID)
		}
		if got.ID != p.ID {
			t.Errorf("CreditPackByID(%q): got ID %q", p.ID, got.ID)
		}
	}
}

func TestCreditPackByID_NotFound(t *testing.T) {
	_, ok := CreditPackByID("credits_9999")
	if ok {
		t.Error("expected not found for unknown credit pack ID")
	}
}

func TestCreditPackByID_Empty(t *testing.T) {
	_, ok := CreditPackByID("")
	if ok {
		t.Error("expected not found for empty ID")
	}
}

func TestPlanByID_Found(t *testing.T) {
	for _, p := range Plans {
		got, ok := PlanByID(p.ID)
		if !ok {
			t.Errorf("PlanByID(%q): not found", p.ID)
		}
		if got.ID != p.ID {
			t.Errorf("PlanByID(%q): got ID %q", p.ID, got.ID)
		}
	}
}

func TestPlanByID_NotFound(t *testing.T) {
	_, ok := PlanByID("enterprise")
	if ok {
		t.Error("expected not found for unknown plan ID")
	}
}

func TestPlanByID_Empty(t *testing.T) {
	_, ok := PlanByID("")
	if ok {
		t.Error("expected not found for empty ID")
	}
}

func TestCreditPacks_Invariants(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range CreditPacks {
		if p.ID == "" {
			t.Error("credit pack has empty ID")
		}
		if seen[p.ID] {
			t.Errorf("duplicate credit pack ID: %q", p.ID)
		}
		seen[p.ID] = true
		if p.Credits <= 0 {
			t.Errorf("pack %q: Credits must be > 0", p.ID)
		}
		if p.USDCents <= 0 {
			t.Errorf("pack %q: USDCents must be > 0", p.ID)
		}
		if p.IDRAmount <= 0 {
			t.Errorf("pack %q: IDRAmount must be > 0", p.ID)
		}
	}
}

func TestPlans_Invariants(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range Plans {
		if p.ID == "" {
			t.Error("plan has empty ID")
		}
		if seen[p.ID] {
			t.Errorf("duplicate plan ID: %q", p.ID)
		}
		seen[p.ID] = true
		if p.USDCents <= 0 {
			t.Errorf("plan %q: USDCents must be > 0", p.ID)
		}
		if p.IDRAmount <= 0 {
			t.Errorf("plan %q: IDRAmount must be > 0", p.ID)
		}
	}
}
