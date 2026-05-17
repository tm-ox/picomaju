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

// ── proxy path ────────────────────────────────────────────────────────────────

func upstreamHandler(status int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
}

func newHandlerWithUpstream(t *testing.T, lic *license.License, srv *httptest.Server) *Handler {
	t.Helper()
	h := newHandler(t, lic, "test-api-key")
	h.httpClient = srv.Client()
	// Redirect all upstream calls to the test server.
	h.httpClient.Transport = &proxyRedirect{base: srv.URL, inner: srv.Client().Transport}
	return h
}

type proxyRedirect struct {
	base  string
	inner http.RoundTripper
}

func (p *proxyRedirect) RoundTrip(r *http.Request) (*http.Response, error) {
	r2 := r.Clone(r.Context())
	r2.URL.Scheme = "http"
	r2.URL.Host = strings.TrimPrefix(p.base, "http://")
	if p.inner != nil {
		return p.inner.RoundTrip(r2)
	}
	return http.DefaultTransport.RoundTrip(r2)
}

func TestProxy_Success_200Forwarded(t *testing.T) {
	lic := &license.License{Active: true, Plan: license.PlanStarter, Token: "tok"}
	upstream := upstreamHandler(http.StatusOK, `{"id":"msg_1"}`)
	defer upstream.Close()

	h := newHandlerWithUpstream(t, lic, upstream)
	w := post(h, "/proxy/v1/messages", "tok")

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "msg_1") {
		t.Errorf("expected upstream body, got: %s", w.Body.String())
	}
}

func TestProxy_Upstream_Non200_Forwarded(t *testing.T) {
	lic := &license.License{Active: true, Plan: license.PlanStarter, Token: "tok"}
	upstream := upstreamHandler(http.StatusBadRequest, `{"error":"bad"}`)
	defer upstream.Close()

	h := newHandlerWithUpstream(t, lic, upstream)
	w := post(h, "/proxy/v1/messages", "tok")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 forwarded from upstream, got %d", w.Code)
	}
}

func TestProxy_CreditDeducted_OnSuccess(t *testing.T) {
	ls := newLicenseStore(t, &license.License{
		Active: true, Plan: license.PlanCredits, Token: "tok", CreditsRemaining: 10,
	})
	upstream := upstreamHandler(http.StatusOK, `{}`)
	defer upstream.Close()

	h := NewHandler(ls, "test-api-key")
	h.httpClient = &http.Client{
		Transport: &proxyRedirect{base: upstream.URL},
	}

	r := httptest.NewRequest(http.MethodPost, "/proxy/v1/messages", strings.NewReader("{}"))
	r.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	lic, _ := ls.Load()
	if lic.CreditsRemaining != 9 {
		t.Errorf("expected 9 credits remaining, got %d", lic.CreditsRemaining)
	}
}

func TestProxy_CreditNotDeducted_OnNon200(t *testing.T) {
	ls := newLicenseStore(t, &license.License{
		Active: true, Plan: license.PlanCredits, Token: "tok", CreditsRemaining: 10,
	})
	upstream := upstreamHandler(http.StatusInternalServerError, `{}`)
	defer upstream.Close()

	h := NewHandler(ls, "test-api-key")
	h.httpClient = &http.Client{
		Transport: &proxyRedirect{base: upstream.URL},
	}

	r := httptest.NewRequest(http.MethodPost, "/proxy/v1/messages", strings.NewReader("{}"))
	r.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	lic, _ := ls.Load()
	if lic.CreditsRemaining != 10 {
		t.Errorf("expected credits unchanged at 10, got %d", lic.CreditsRemaining)
	}
}

func TestProxy_SubscriptionPlan_NoCreditDeduction(t *testing.T) {
	ls := newLicenseStore(t, &license.License{
		Active: true, Plan: license.PlanStarter, Token: "tok", CreditsRemaining: 5,
	})
	upstream := upstreamHandler(http.StatusOK, `{}`)
	defer upstream.Close()

	h := NewHandler(ls, "test-api-key")
	h.httpClient = &http.Client{
		Transport: &proxyRedirect{base: upstream.URL},
	}

	r := httptest.NewRequest(http.MethodPost, "/proxy/v1/messages", strings.NewReader("{}"))
	r.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	lic, _ := ls.Load()
	if lic.CreditsRemaining != 5 {
		t.Errorf("expected credits unchanged on subscription plan, got %d", lic.CreditsRemaining)
	}
}
