package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
	"picomaju/internal/license"
	"picomaju/internal/payment"
)

// stripeWebhook handles Stripe webhook events.
// Verifies the signature, then activates the license on checkout.session.completed.
func (h *uiHandler) stripeWebhook(w http.ResponseWriter, r *http.Request) {
	cfg := payment.LoadConfig()
	if !cfg.StripeConfigured() {
		http.Error(w, "stripe not configured", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), cfg.StripeWebhookSecret)
	if err != nil {
		http.Error(w, "signature verification failed", http.StatusBadRequest)
		return
	}

	if event.Type != "checkout.session.completed" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		http.Error(w, "parse error", http.StatusBadRequest)
		return
	}

	if err := h.activateFromStripe(session); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *uiHandler) activateFromStripe(session stripe.CheckoutSession) error {
	l, err := h.license.Load()
	if err != nil {
		return err
	}

	meta := session.Metadata
	switch meta["type"] {
	case "credits":
		credits, _ := strconv.Atoi(meta["credits"])
		l.Active = true
		l.Plan = license.PlanCredits
		l.CreditsRemaining += credits
		l.Token = session.ID
	case "subscription":
		planID := meta["plan_id"]
		if planID == "" {
			return fmt.Errorf("missing plan_id in session metadata")
		}
		l.Active = true
		l.Plan = planID
		l.CreditsRemaining = 0
		if session.Subscription != nil {
			l.Token = session.Subscription.ID
		}
		// Subscription expiry handled by Stripe; set 35-day local expiry as safety net.
		l.ExpiresAt = time.Now().AddDate(0, 0, 35).Unix()
	default:
		return fmt.Errorf("unknown payment type %q", meta["type"])
	}

	return h.license.Save(l)
}

// xenditWebhook handles Xendit webhook events (invoice.paid).
func (h *uiHandler) xenditWebhook(w http.ResponseWriter, r *http.Request) {
	cfg := payment.LoadConfig()
	if !cfg.XenditConfigured() {
		http.Error(w, "xendit not configured", http.StatusServiceUnavailable)
		return
	}

	// Xendit webhook verification: X-CALLBACK-TOKEN header must match our token.
	if r.Header.Get("X-CALLBACK-TOKEN") != cfg.XenditWebhookToken {
		http.Error(w, "invalid callback token", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	var event struct {
		Status   string            `json:"status"`
		Metadata map[string]string `json:"metadata"`
		ID       string            `json:"id"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "parse error", http.StatusBadRequest)
		return
	}

	if event.Status != "PAID" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := h.activateFromXendit(event.ID, event.Metadata); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *uiHandler) activateFromXendit(invoiceID string, meta map[string]string) error {
	l, err := h.license.Load()
	if err != nil {
		return err
	}

	switch meta["type"] {
	case "credits":
		credits, _ := strconv.Atoi(meta["credits"])
		l.Active = true
		l.Plan = license.PlanCredits
		l.CreditsRemaining += credits
		l.Token = invoiceID
	case "subscription":
		planID := meta["plan_id"]
		if planID == "" {
			return fmt.Errorf("missing plan_id in metadata")
		}
		l.Active = true
		l.Plan = planID
		l.CreditsRemaining = 0
		l.Token = invoiceID
		l.ExpiresAt = time.Now().AddDate(0, 0, 35).Unix()
	default:
		return fmt.Errorf("unknown payment type %q", meta["type"])
	}

	return h.license.Save(l)
}

// xenditWebhookHMAC is an alternative Xendit verification using HMAC-SHA256
// for endpoints that use the newer Xendit signing method.
func verifyXenditHMAC(body []byte, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
