package llmproxy

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"picomaju/internal/license"
)

func newLicenseStore(t *testing.T, l *license.License) *license.Store {
	t.Helper()
	s := license.NewStore(filepath.Join(t.TempDir(), "license.json"))
	if l != nil {
		if err := s.Save(l); err != nil {
			t.Fatalf("save license: %v", err)
		}
	}
	return s
}

func newHandler(t *testing.T, lic *license.License, apiKey string) *Handler {
	t.Helper()
	return NewHandler(newLicenseStore(t, lic), apiKey)
}

func post(h *Handler, path, token string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{}"))
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// ── extractToken ─────────────────────────────────────────────────────────────

func TestExtractToken_Bearer(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("Authorization", "Bearer mytoken")
	if got := extractToken(r); got != "mytoken" {
		t.Errorf("got %q", got)
	}
}

func TestExtractToken_XProxyToken(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("X-Proxy-Token", "alttoken")
	if got := extractToken(r); got != "alttoken" {
		t.Errorf("got %q", got)
	}
}

func TestExtractToken_Missing(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	if got := extractToken(r); got != "" {
		t.Errorf("expected empty token, got %q", got)
	}
}

// ── ServeHTTP gates ───────────────────────────────────────────────────────────

func TestProxy_MethodNotAllowed(t *testing.T) {
	h := newHandler(t, nil, "")
	r := httptest.NewRequest(http.MethodGet, "/proxy/v1/messages", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestProxy_NoToken_Unauthorized(t *testing.T) {
	h := newHandler(t, nil, "")
	r := httptest.NewRequest(http.MethodPost, "/proxy/v1/messages", strings.NewReader("{}"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestProxy_NoLicense_Forbidden(t *testing.T) {
	// No license file → empty license → not active.
	h := newHandler(t, nil, "key")
	w := post(h, "/proxy/v1/messages", "some-token")
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestProxy_WrongToken_Forbidden(t *testing.T) {
	lic := &license.License{Active: true, Plan: license.PlanStarter, Token: "correct-token"}
	h := newHandler(t, lic, "key")
	w := post(h, "/proxy/v1/messages", "wrong-token")
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestProxy_NoAPIKey_ServiceUnavailable(t *testing.T) {
	lic := &license.License{Active: true, Plan: license.PlanStarter, Token: "tok"}
	h := newHandler(t, lic, "") // no API key
	w := post(h, "/proxy/v1/messages", "tok")
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestProxy_RateLimitExceeded(t *testing.T) {
	lic := &license.License{Active: true, Plan: license.PlanStarter, Token: "tok"}
	// Use a real upstream so we bypass the API key check and hit the rate limiter.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	h := newHandler(t, lic, "real-key")
	// Exhaust the rate limiter directly.
	for range rateLimit {
		h.limiter.allow("tok")
	}
	w := post(h, "/proxy/v1/messages", "tok")
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
}

// ── rateLimiter ───────────────────────────────────────────────────────────────

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	rl := newRateLimiter()
	for range rateLimit {
		if !rl.allow("tok") {
			t.Fatal("expected allow before limit reached")
		}
	}
}

func TestRateLimiter_BlocksAtLimit(t *testing.T) {
	rl := newRateLimiter()
	for range rateLimit {
		rl.allow("tok")
	}
	if rl.allow("tok") {
		t.Error("expected block at rate limit")
	}
}

func TestRateLimiter_ResetsAfterWindow(t *testing.T) {
	rl := newRateLimiter()
	// Inject old timestamps directly to simulate an expired window.
	past := time.Now().Add(-(rateWindow + time.Second))
	rl.mu.Lock()
	rl.buckets["tok"] = make([]time.Time, rateLimit)
	for i := range rateLimit {
		rl.buckets["tok"][i] = past
	}
	rl.mu.Unlock()
	if !rl.allow("tok") {
		t.Error("expected allow after window reset")
	}
}

func TestRateLimiter_IndependentTokens(t *testing.T) {
	rl := newRateLimiter()
	for range rateLimit {
		rl.allow("tok-a")
	}
	// tok-b should not be affected by tok-a's exhaustion.
	if !rl.allow("tok-b") {
		t.Error("rate limit should be per-token")
	}
}
