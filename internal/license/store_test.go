package license

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── IsActive ────────────────────────────────────────────────────────────────

func TestIsActive_NilLicense(t *testing.T) {
	var l *License
	if l.IsActive() {
		t.Error("nil license should not be active")
	}
}

func TestIsActive_InactiveLicense(t *testing.T) {
	l := &License{Active: false}
	if l.IsActive() {
		t.Error("inactive license should not be active")
	}
}

func TestIsActive_ActiveNoExpiry(t *testing.T) {
	l := &License{Active: true, Plan: PlanStarter}
	if !l.IsActive() {
		t.Error("active license with no expiry should be active")
	}
}

func TestIsActive_NotYetExpired(t *testing.T) {
	l := &License{Active: true, Plan: PlanStarter, ExpiresAt: time.Now().Add(24 * time.Hour).Unix()}
	if !l.IsActive() {
		t.Error("license expiring in the future should be active")
	}
}

func TestIsActive_Expired(t *testing.T) {
	l := &License{Active: true, Plan: PlanStarter, ExpiresAt: time.Now().Add(-time.Second).Unix()}
	if l.IsActive() {
		t.Error("expired license should not be active")
	}
}

func TestIsActive_Credits_HasCredits(t *testing.T) {
	l := &License{Active: true, Plan: PlanCredits, CreditsRemaining: 1}
	if !l.IsActive() {
		t.Error("credits plan with credits remaining should be active")
	}
}

func TestIsActive_Credits_ZeroCredits(t *testing.T) {
	l := &License{Active: true, Plan: PlanCredits, CreditsRemaining: 0}
	if l.IsActive() {
		t.Error("credits plan with zero credits should not be active")
	}
}

// ── PlanLabel ───────────────────────────────────────────────────────────────

func TestPlanLabel(t *testing.T) {
	cases := []struct {
		plan string
		want string
	}{
		{PlanCredits, "Credits"},
		{PlanStarter, "Starter"},
		{PlanPro, "Pro"},
		{PlanFree, "Free"},
		{"unknown", "Free"},
	}
	for _, c := range cases {
		l := &License{Plan: c.plan}
		if got := l.PlanLabel(); got != c.want {
			t.Errorf("plan %q: got %q, want %q", c.plan, got, c.want)
		}
	}
}

// ── DeductCredit ────────────────────────────────────────────────────────────

func newTempStore(t *testing.T, l *License) *Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "license.json")
	s := NewStore(path)
	if l != nil {
		if err := s.Save(l); err != nil {
			t.Fatalf("setup Save: %v", err)
		}
	}
	return s
}

func TestDeductCredit_Success(t *testing.T) {
	s := newTempStore(t, &License{Active: true, Plan: PlanCredits, CreditsRemaining: 5})
	ok, err := s.DeductCredit()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected deduction to succeed")
	}
	l, _ := s.Load()
	if l.CreditsRemaining != 4 {
		t.Errorf("expected 4 credits remaining, got %d", l.CreditsRemaining)
	}
	if !l.Active {
		t.Error("license should still be active with credits remaining")
	}
}

func TestDeductCredit_LastCredit_DeactivatesLicense(t *testing.T) {
	s := newTempStore(t, &License{Active: true, Plan: PlanCredits, CreditsRemaining: 1})
	ok, err := s.DeductCredit()
	if err != nil || !ok {
		t.Fatalf("unexpected result: ok=%v err=%v", ok, err)
	}
	l, _ := s.Load()
	if l.CreditsRemaining != 0 {
		t.Errorf("expected 0 credits, got %d", l.CreditsRemaining)
	}
	if l.Active {
		t.Error("license should be deactivated when credits reach zero")
	}
}

func TestDeductCredit_NoCredits(t *testing.T) {
	s := newTempStore(t, &License{Active: true, Plan: PlanCredits, CreditsRemaining: 0})
	ok, err := s.DeductCredit()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("deduction should fail when no credits remain")
	}
}

func TestDeductCredit_WrongPlan(t *testing.T) {
	s := newTempStore(t, &License{Active: true, Plan: PlanStarter})
	ok, err := s.DeductCredit()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("deduction should fail on non-credits plan")
	}
}

func TestDeductCredit_NoLicenseFile(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(filepath.Join(dir, "license.json"))
	// No file written — store returns empty license.
	ok, err := s.DeductCredit()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("deduction should fail on empty license")
	}
}

func TestDeductCredit_PersistsAcrossReload(t *testing.T) {
	s := newTempStore(t, &License{Active: true, Plan: PlanCredits, CreditsRemaining: 10})
	for i := range 3 {
		ok, err := s.DeductCredit()
		if err != nil || !ok {
			t.Fatalf("deduction %d failed: ok=%v err=%v", i, ok, err)
		}
	}
	l, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if l.CreditsRemaining != 7 {
		t.Errorf("expected 7 credits after 3 deductions, got %d", l.CreditsRemaining)
	}
}

func TestStore_SaveAndLoad_RoundTrip(t *testing.T) {
	orig := &License{
		Active:           true,
		Plan:             PlanPro,
		CreditsRemaining: 0,
		Token:            "tok_abc",
		ExpiresAt:        12345678,
	}
	s := newTempStore(t, orig)
	l, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if l.Plan != orig.Plan || l.Token != orig.Token || l.ExpiresAt != orig.ExpiresAt {
		t.Errorf("round-trip mismatch: %+v", l)
	}
}

func TestStore_Load_MissingFile(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "nope.json"))
	l, err := s.Load()
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if l == nil {
		t.Fatal("expected empty license, got nil")
	}
	if l.Active {
		t.Error("empty license should not be active")
	}
}

func TestStore_Load_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "license.json")
	os.WriteFile(path, []byte("not json {{{"), 0644)
	s := NewStore(path)
	_, err := s.Load()
	if err == nil {
		t.Error("expected error loading corrupt JSON")
	}
}
