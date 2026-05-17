package payment

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ── Config ────────────────────────────────────────────────────────────────────

func TestStripeConfigured_True(t *testing.T) {
	c := Config{StripeSecretKey: "sk_test_x", StripeWebhookSecret: "whsec_x"}
	if !c.StripeConfigured() {
		t.Error("expected StripeConfigured() true")
	}
}

func TestStripeConfigured_MissingKey(t *testing.T) {
	c := Config{StripeWebhookSecret: "whsec_x"}
	if c.StripeConfigured() {
		t.Error("expected StripeConfigured() false when key missing")
	}
}

func TestStripeConfigured_MissingWebhook(t *testing.T) {
	c := Config{StripeSecretKey: "sk_test_x"}
	if c.StripeConfigured() {
		t.Error("expected StripeConfigured() false when webhook secret missing")
	}
}

func TestXenditConfigured_True(t *testing.T) {
	c := Config{XenditAPIKey: "xnd_x", XenditWebhookToken: "tok"}
	if !c.XenditConfigured() {
		t.Error("expected XenditConfigured() true")
	}
}

func TestXenditConfigured_Missing(t *testing.T) {
	c := Config{XenditAPIKey: "xnd_x"}
	if c.XenditConfigured() {
		t.Error("expected XenditConfigured() false when token missing")
	}
}

// ── StripeCheckoutURL validation (no network) ─────────────────────────────────

func TestStripeCheckoutURL_NeitherPackNorPlan(t *testing.T) {
	cfg := Config{StripeSecretKey: "sk_test_x", BaseURL: "https://example.com"}
	_, err := StripeCheckoutURL(cfg, "", "")
	if err == nil || !strings.Contains(err.Error(), "packID or planID required") {
		t.Errorf("expected packID/planID error, got %v", err)
	}
}

func TestStripeCheckoutURL_UnknownPack(t *testing.T) {
	cfg := Config{StripeSecretKey: "sk_test_x", BaseURL: "https://example.com"}
	_, err := StripeCheckoutURL(cfg, "credits_9999", "")
	if err == nil || !strings.Contains(err.Error(), "unknown credit pack") {
		t.Errorf("expected unknown pack error, got %v", err)
	}
}

func TestStripeCheckoutURL_UnknownPlan(t *testing.T) {
	cfg := Config{StripeSecretKey: "sk_test_x", BaseURL: "https://example.com"}
	_, err := StripeCheckoutURL(cfg, "", "enterprise")
	if err == nil || !strings.Contains(err.Error(), "unknown plan") {
		t.Errorf("expected unknown plan error, got %v", err)
	}
}

func TestStripeCheckoutURL_PlanNoPriceID(t *testing.T) {
	cfg := Config{StripeSecretKey: "sk_test_x", BaseURL: "https://example.com"}
	_, err := StripeCheckoutURL(cfg, "", "starter")
	if err == nil || !strings.Contains(err.Error(), "no Stripe price ID") {
		t.Errorf("expected no price ID error, got %v", err)
	}
}

// ── XenditCheckoutURL ─────────────────────────────────────────────────────────

func TestXenditCheckoutURL_NeitherPackNorPlan(t *testing.T) {
	cfg := Config{XenditAPIKey: "xnd_x", BaseURL: "https://example.com"}
	_, err := XenditCheckoutURL(cfg, "", "")
	if err == nil || !strings.Contains(err.Error(), "packID or planID required") {
		t.Errorf("expected packID/planID error, got %v", err)
	}
}

func TestXenditCheckoutURL_UnknownPack(t *testing.T) {
	cfg := Config{XenditAPIKey: "xnd_x", BaseURL: "https://example.com"}
	_, err := XenditCheckoutURL(cfg, "credits_9999", "")
	if err == nil || !strings.Contains(err.Error(), "unknown credit pack") {
		t.Errorf("expected unknown pack error, got %v", err)
	}
}

func TestXenditCheckoutURL_UnknownPlan(t *testing.T) {
	cfg := Config{XenditAPIKey: "xnd_x", BaseURL: "https://example.com"}
	_, err := XenditCheckoutURL(cfg, "", "enterprise")
	if err == nil || !strings.Contains(err.Error(), "unknown plan") {
		t.Errorf("expected unknown plan error, got %v", err)
	}
}

func TestXenditCheckoutURL_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	old := xenditHTTPClient
	xenditHTTPClient = srv.Client()
	defer func() { xenditHTTPClient = old }()

	// Also redirect xenditBaseURL → test server via the client transport.
	xenditHTTPClient = &http.Client{
		Transport: &xenditRedirect{target: srv.URL, inner: srv.Client().Transport},
	}

	cfg := Config{XenditAPIKey: "xnd_x", BaseURL: "https://example.com"}
	_, err := XenditCheckoutURL(cfg, "credits_100", "")
	if err == nil || !strings.Contains(err.Error(), "xendit error") {
		t.Errorf("expected xendit error on 4xx, got %v", err)
	}
}

func TestXenditCheckoutURL_Success(t *testing.T) {
	invoiceURL := "https://checkout.xendit.co/test-invoice"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify basic auth.
		user, _, _ := r.BasicAuth()
		if user != "xnd_key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		resp := map[string]string{
			"id":          fmt.Sprintf("inv_%d", time.Now().Unix()),
			"invoice_url": invoiceURL,
			"external_id": "pm-credits-credits_100-123",
			"status":      "PENDING",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	old := xenditHTTPClient
	xenditHTTPClient = &http.Client{
		Transport: &xenditRedirect{target: srv.URL, inner: srv.Client().Transport},
	}
	defer func() { xenditHTTPClient = old }()

	cfg := Config{XenditAPIKey: "xnd_key", BaseURL: "https://example.com"}
	got, err := XenditCheckoutURL(cfg, "credits_100", "")
	if err != nil {
		t.Fatalf("XenditCheckoutURL: %v", err)
	}
	if got != invoiceURL {
		t.Errorf("expected %q, got %q", invoiceURL, got)
	}
}

// xenditRedirect redirects all requests to a test server preserving method/headers/body.
type xenditRedirect struct {
	target string
	inner  http.RoundTripper
}

func (x *xenditRedirect) RoundTrip(r *http.Request) (*http.Response, error) {
	r2 := r.Clone(r.Context())
	r2.URL.Scheme = "http"
	r2.URL.Host = strings.TrimPrefix(x.target, "http://")
	r2.URL.Path = "/v2/invoices"
	t := x.inner
	if t == nil {
		t = http.DefaultTransport
	}
	return t.RoundTrip(r2)
}
