package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"picomaju/internal/event"
	"picomaju/internal/license"
)

// ── broadcaster ──────────────────────────────────────────────────────────────

type broadcaster struct {
	mu   sync.Mutex
	subs map[string][]chan event.Event
}

func newBroadcaster() *broadcaster {
	return &broadcaster{subs: make(map[string][]chan event.Event)}
}

func (b *broadcaster) subscribe(agentID string) (chan event.Event, func()) {
	ch := make(chan event.Event, 16)
	b.mu.Lock()
	b.subs[agentID] = append(b.subs[agentID], ch)
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		for i, s := range b.subs[agentID] {
			if s == ch {
				b.subs[agentID] = append(b.subs[agentID][:i], b.subs[agentID][i+1:]...)
				break
			}
		}
		close(ch)
	}
}

func (b *broadcaster) publish(e event.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.subs[e.AgentID] {
		select {
		case ch <- e:
		default:
		}
	}
}

// ── approvalBroker ───────────────────────────────────────────────────────────

type approvalBroker struct {
	mu      sync.Mutex
	waiters map[string]chan string
}

func newApprovalBroker() *approvalBroker {
	return &approvalBroker{waiters: make(map[string]chan string)}
}

// wait blocks until a decision is made or the timeout elapses.
func (ab *approvalBroker) wait(eventID string, timeout time.Duration) (decision string, resolved bool) {
	ch := make(chan string, 1)
	ab.mu.Lock()
	ab.waiters[eventID] = ch
	ab.mu.Unlock()
	defer func() {
		ab.mu.Lock()
		delete(ab.waiters, eventID)
		ab.mu.Unlock()
	}()
	select {
	case d := <-ch:
		return d, true
	case <-time.After(timeout):
		return "", false
	}
}

// resolve sends a decision to a waiting picoclaw goroutine.
func (ab *approvalBroker) resolve(eventID, decision string) bool {
	ab.mu.Lock()
	ch, ok := ab.waiters[eventID]
	ab.mu.Unlock()
	if !ok {
		return false
	}
	select {
	case ch <- decision:
		return true
	default:
		return false
	}
}

// ── agentHandler ─────────────────────────────────────────────────────────────

type agentHandler struct {
	events    *event.Store
	license   *license.Store
	bc        *broadcaster
	approvals *approvalBroker
}

func newAgentHandler(events *event.Store, lic *license.Store) *agentHandler {
	return &agentHandler{
		events:    events,
		license:   lic,
		bc:        newBroadcaster(),
		approvals: newApprovalBroker(),
	}
}

func (h *agentHandler) verifyToken(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	tok := strings.TrimPrefix(auth, "Bearer ")
	if tok == "" || tok == auth {
		return false
	}
	lic, err := h.license.Load()
	if err != nil {
		return false
	}
	return lic.IsActive() && lic.Token == tok
}

// ingestEvent — POST /agents/{id}/events
// Called by picoclaw to report an action or request approval.
func (h *agentHandler) ingestEvent(w http.ResponseWriter, r *http.Request) {
	if !h.verifyToken(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	agentID := chi.URLParam(r, "id")
	var e event.Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	e.AgentID = agentID
	if e.ID == "" {
		e.ID = fmt.Sprintf("%x", time.Now().UnixNano())
	}
	if e.Timestamp == 0 {
		e.Timestamp = time.Now().Unix()
	}
	if err := h.events.Append(e); err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}
	h.bc.publish(e)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": e.ID})
}

// streamEvents — GET /agents/{id}/events/stream
// Browser subscribes via SSE to receive real-time events for one agent.
func (h *agentHandler) streamEvents(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "id")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch, unsub := h.bc.subscribe(agentID)
	defer unsub()

	for {
		select {
		case <-r.Context().Done():
			return
		case e, ok := <-ch:
			if !ok {
				return
			}
			b, err := json.Marshal(e)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		}
	}
}

// waitApproval — GET /agents/{id}/approvals/{eventId}
// picoclaw long-polls here after posting an approval_request event.
// Returns 408 on timeout so picoclaw knows to retry.
func (h *agentHandler) waitApproval(w http.ResponseWriter, r *http.Request) {
	if !h.verifyToken(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	eventID := chi.URLParam(r, "eventId")
	decision, resolved := h.approvals.wait(eventID, 25*time.Second)
	if !resolved {
		w.WriteHeader(http.StatusRequestTimeout)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"decision": decision})
}

// resolveApproval — POST /agents/{id}/approvals/{eventId}
// Human approves or denies a pending approval request via the browser UI.
func (h *agentHandler) resolveApproval(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "id")
	eventID := chi.URLParam(r, "eventId")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	decision := r.FormValue("decision")
	if decision != "approved" && decision != "denied" {
		http.Error(w, "decision must be approved or denied", http.StatusBadRequest)
		return
	}
	result := event.Event{
		ID:        fmt.Sprintf("%x", time.Now().UnixNano()),
		AgentID:   agentID,
		Type:      event.EventTypeApprovalResult,
		Timestamp: time.Now().Unix(),
		RefID:     eventID,
		Decision:  decision,
		Summary:   "Approval " + decision,
	}
	_ = h.events.Append(result)
	h.bc.publish(result)
	h.approvals.resolve(eventID, decision)
	http.Redirect(w, r, "/staff/"+agentID, http.StatusSeeOther)
}
