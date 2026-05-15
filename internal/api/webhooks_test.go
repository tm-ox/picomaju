package api

import (
	"path/filepath"
	"testing"

	"github.com/stripe/stripe-go/v82"
	"picomaju/internal/license"
)

func newWebhookHandler(t *testing.T, initial *license.License) *uiHandler {
	t.Helper()
	path := filepath.Join(t.TempDir(), "license.json")
	s := license.NewStore(path)
	if initial != nil {
		if err := s.Save(initial); err != nil {
			t.Fatalf("setup Save: %v", err)
		}
	}
	return &uiHandler{license: s}
}

func loadLicense(t *testing.T, h *uiHandler) *license.License {
	t.Helper()
	l, err := h.license.Load()
	if err != nil {
		t.Fatalf("load license: %v", err)
	}
	return l
}

// ── activateFromStripe ───────────────────────────────────────────────────────

func TestActivateFromStripe_Credits(t *testing.T) {
	h := newWebhookHandler(t, &license.License{})
	session := stripe.CheckoutSession{
		ID:       "cs_test_123",
		Metadata: map[string]string{"type": "credits", "credits": "100"},
	}
	if err := h.activateFromStripe(session); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l := loadLicense(t, h)
	if !l.Active {
		t.Error("license should be active")
	}
	if l.Plan != license.PlanCredits {
		t.Errorf("expected plan %q, got %q", license.PlanCredits, l.Plan)
	}
	if l.CreditsRemaining != 100 {
		t.Errorf("expected 100 credits, got %d", l.CreditsRemaining)
	}
	if l.Token != "cs_test_123" {
		t.Errorf("expected token cs_test_123, got %q", l.Token)
	}
}

func TestActivateFromStripe_Credits_Accumulate(t *testing.T) {
	h := newWebhookHandler(t, &license.License{Active: true, Plan: license.PlanCredits, CreditsRemaining: 50})
	session := stripe.CheckoutSession{
		ID:       "cs_test_456",
		Metadata: map[string]string{"type": "credits", "credits": "100"},
	}
	if err := h.activateFromStripe(session); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l := loadLicense(t, h)
	if l.CreditsRemaining != 150 {
		t.Errorf("expected 150 credits after top-up, got %d", l.CreditsRemaining)
	}
}

func TestActivateFromStripe_Credits_InvalidAmount(t *testing.T) {
	h := newWebhookHandler(t, &license.License{})
	cases := []string{"", "abc", "0", "-5"}
	for _, val := range cases {
		session := stripe.CheckoutSession{
			Metadata: map[string]string{"type": "credits", "credits": val},
		}
		if err := h.activateFromStripe(session); err == nil {
			t.Errorf("credits=%q: expected error, got nil", val)
		}
	}
}

func TestActivateFromStripe_Subscription(t *testing.T) {
	h := newWebhookHandler(t, &license.License{})
	sub := &stripe.Subscription{ID: "sub_abc"}
	session := stripe.CheckoutSession{
		ID:           "cs_test_sub",
		Metadata:     map[string]string{"type": "subscription", "plan_id": "starter"},
		Subscription: sub,
	}
	if err := h.activateFromStripe(session); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l := loadLicense(t, h)
	if !l.Active {
		t.Error("license should be active")
	}
	if l.Plan != "starter" {
		t.Errorf("expected plan starter, got %q", l.Plan)
	}
	if l.Token != "sub_abc" {
		t.Errorf("expected token sub_abc, got %q", l.Token)
	}
	if l.ExpiresAt == 0 {
		t.Error("expected non-zero ExpiresAt for subscription")
	}
	if l.CreditsRemaining != 0 {
		t.Errorf("expected 0 credits for subscription plan, got %d", l.CreditsRemaining)
	}
}

func TestActivateFromStripe_Subscription_MissingPlanID(t *testing.T) {
	h := newWebhookHandler(t, &license.License{})
	session := stripe.CheckoutSession{
		Metadata: map[string]string{"type": "subscription"},
	}
	if err := h.activateFromStripe(session); err == nil {
		t.Error("expected error for missing plan_id")
	}
}

func TestActivateFromStripe_UnknownType(t *testing.T) {
	h := newWebhookHandler(t, &license.License{})
	session := stripe.CheckoutSession{
		Metadata: map[string]string{"type": "gift_card"},
	}
	if err := h.activateFromStripe(session); err == nil {
		t.Error("expected error for unknown payment type")
	}
}

// ── activateFromXendit ───────────────────────────────────────────────────────

func TestActivateFromXendit_Credits(t *testing.T) {
	h := newWebhookHandler(t, &license.License{})
	meta := map[string]string{"type": "credits", "credits": "500"}
	if err := h.activateFromXendit("inv_xyz", meta); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l := loadLicense(t, h)
	if !l.Active {
		t.Error("license should be active")
	}
	if l.Plan != license.PlanCredits {
		t.Errorf("expected credits plan, got %q", l.Plan)
	}
	if l.CreditsRemaining != 500 {
		t.Errorf("expected 500 credits, got %d", l.CreditsRemaining)
	}
	if l.Token != "inv_xyz" {
		t.Errorf("expected token inv_xyz, got %q", l.Token)
	}
}

func TestActivateFromXendit_Credits_Accumulate(t *testing.T) {
	h := newWebhookHandler(t, &license.License{Active: true, Plan: license.PlanCredits, CreditsRemaining: 200})
	if err := h.activateFromXendit("inv_2", map[string]string{"type": "credits", "credits": "300"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l := loadLicense(t, h)
	if l.CreditsRemaining != 500 {
		t.Errorf("expected 500 after top-up, got %d", l.CreditsRemaining)
	}
}

func TestActivateFromXendit_Credits_InvalidAmount(t *testing.T) {
	h := newWebhookHandler(t, &license.License{})
	cases := []string{"", "bad", "0", "-1"}
	for _, val := range cases {
		if err := h.activateFromXendit("inv", map[string]string{"type": "credits", "credits": val}); err == nil {
			t.Errorf("credits=%q: expected error, got nil", val)
		}
	}
}

func TestActivateFromXendit_Subscription(t *testing.T) {
	h := newWebhookHandler(t, &license.License{})
	meta := map[string]string{"type": "subscription", "plan_id": "pro"}
	if err := h.activateFromXendit("inv_pro", meta); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l := loadLicense(t, h)
	if !l.Active {
		t.Error("license should be active")
	}
	if l.Plan != "pro" {
		t.Errorf("expected plan pro, got %q", l.Plan)
	}
	if l.Token != "inv_pro" {
		t.Errorf("expected token inv_pro, got %q", l.Token)
	}
	if l.ExpiresAt == 0 {
		t.Error("expected non-zero ExpiresAt for subscription")
	}
}

func TestActivateFromXendit_Subscription_MissingPlanID(t *testing.T) {
	h := newWebhookHandler(t, &license.License{})
	if err := h.activateFromXendit("inv", map[string]string{"type": "subscription"}); err == nil {
		t.Error("expected error for missing plan_id")
	}
}

func TestActivateFromXendit_UnknownType(t *testing.T) {
	h := newWebhookHandler(t, &license.License{})
	if err := h.activateFromXendit("inv", map[string]string{"type": "refund"}); err == nil {
		t.Error("expected error for unknown payment type")
	}
}
