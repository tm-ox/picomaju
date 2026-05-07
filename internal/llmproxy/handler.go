package llmproxy

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"picomaju/internal/license"
)

const (
	anthropicBase    = "https://api.anthropic.com"
	anthropicVersion = "2023-06-01"
	rateWindow       = time.Minute
	rateLimit        = 60 // requests per minute per token
)

type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string][]time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{buckets: make(map[string][]time.Time)}
}

func (rl *rateLimiter) allow(token string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-rateWindow)
	times := rl.buckets[token]
	recent := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	if len(recent) >= rateLimit {
		rl.buckets[token] = recent
		return false
	}
	rl.buckets[token] = append(recent, time.Now())
	return true
}

// Handler proxies /proxy/v1/* → https://api.anthropic.com/v1/* with token auth and credit metering.
type Handler struct {
	license    *license.Store
	apiKey     string
	httpClient *http.Client
	limiter    *rateLimiter
}

func NewHandler(licenseStore *license.Store, apiKey string) *Handler {
	return &Handler{
		license:    licenseStore,
		apiKey:     apiKey,
		httpClient: &http.Client{},
		limiter:    newRateLimiter(),
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := extractToken(r)
	if token == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	lic, err := h.license.Load()
	if err != nil || !lic.IsActive() || lic.Token != token {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if !h.limiter.allow(token) {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	if h.apiKey == "" {
		http.Error(w, "proxy not configured", http.StatusServiceUnavailable)
		return
	}

	// Strip /proxy prefix → /v1/...
	upstreamPath := strings.TrimPrefix(r.URL.Path, "/proxy")
	target := anthropicBase + upstreamPath

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, target, r.Body)
	if err != nil {
		http.Error(w, "proxy error", http.StatusBadGateway)
		return
	}
	req.Header.Set("x-api-key", h.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("content-type", "application/json")
	if accept := r.Header.Get("accept"); accept != "" {
		req.Header.Set("accept", accept)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Meter: deduct one credit per successful request on credits plan.
	if resp.StatusCode == http.StatusOK && lic.Plan == license.PlanCredits {
		_, _ = h.license.DeductCredit()
	}

	for key, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(key, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Flush incrementally so SSE streaming works.
	if flusher, ok := w.(http.Flusher); ok {
		buf := make([]byte, 4096)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				_, _ = w.Write(buf[:n])
				flusher.Flush()
			}
			if readErr != nil {
				break
			}
		}
	} else {
		_, _ = io.Copy(w, resp.Body)
	}
}

func extractToken(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return r.Header.Get("X-Proxy-Token")
}
