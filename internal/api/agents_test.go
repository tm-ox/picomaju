package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"picomaju/internal/event"
	"picomaju/internal/license"
)

func newAgentHandlerForTest(t *testing.T) (*agentHandler, *event.Store, *license.Store) {
	t.Helper()
	dir := t.TempDir()
	ev := event.NewStore(filepath.Join(dir, "events"))
	lic := license.NewStore(filepath.Join(dir, "license.json"))
	return newAgentHandler(ev, lic), ev, lic
}

func agentRequest(method, path string, params map[string]string, body []byte, token string) *http.Request {
	var r *http.Request
	if len(body) > 0 {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func formRequest(method, path string, params map[string]string, form string) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(form))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// ── ingestEvent ───────────────────────────────────────────────────────────────

func TestIngestEvent_NoToken(t *testing.T) {
	ah, _, _ := newAgentHandlerForTest(t)
	r := agentRequest(http.MethodPost, "/agents/a1/events", map[string]string{"id": "a1"}, []byte(`{}`), "")
	w := httptest.NewRecorder()
	ah.ingestEvent(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestIngestEvent_WrongToken(t *testing.T) {
	ah, _, lic := newAgentHandlerForTest(t)
	lic.Save(&license.License{Active: true, Token: "correct"})
	r := agentRequest(http.MethodPost, "/agents/a1/events", map[string]string{"id": "a1"}, []byte(`{}`), "wrong")
	w := httptest.NewRecorder()
	ah.ingestEvent(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestIngestEvent_Success(t *testing.T) {
	ah, ev, lic := newAgentHandlerForTest(t)
	lic.Save(&license.License{Active: true, Token: "tok"})

	body, _ := json.Marshal(map[string]string{
		"type":    "action",
		"summary": "Sent WhatsApp reply",
		"channel": "whatsapp",
	})
	r := agentRequest(http.MethodPost, "/agents/a1/events", map[string]string{"id": "a1"}, body, "tok")
	w := httptest.NewRecorder()
	ah.ingestEvent(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	events, _ := ev.ListByAgent("a1")
	if len(events) != 1 || events[0].Summary != "Sent WhatsApp reply" {
		t.Errorf("event not stored correctly: %+v", events)
	}
}

func TestIngestEvent_AgentIDOverriddenFromPath(t *testing.T) {
	ah, ev, lic := newAgentHandlerForTest(t)
	lic.Save(&license.License{Active: true, Token: "tok"})

	body, _ := json.Marshal(map[string]string{"agent_id": "injected", "type": "action", "summary": "x"})
	r := agentRequest(http.MethodPost, "/agents/a1/events", map[string]string{"id": "a1"}, body, "tok")
	w := httptest.NewRecorder()
	ah.ingestEvent(w, r)

	events, _ := ev.ListByAgent("a1")
	if len(events) == 0 || events[0].AgentID != "a1" {
		t.Errorf("agent_id should be set from path, got %+v", events)
	}
}

func TestIngestEvent_SetsTimestampIfMissing(t *testing.T) {
	ah, ev, lic := newAgentHandlerForTest(t)
	lic.Save(&license.License{Active: true, Token: "tok"})

	body, _ := json.Marshal(map[string]string{"type": "action", "summary": "x"})
	r := agentRequest(http.MethodPost, "/agents/a1/events", map[string]string{"id": "a1"}, body, "tok")
	ah.ingestEvent(httptest.NewRecorder(), r)

	events, _ := ev.ListByAgent("a1")
	if len(events) == 0 || events[0].Timestamp == 0 {
		t.Error("timestamp should be set automatically")
	}
}

// ── approvalBroker ────────────────────────────────────────────────────────────

func TestApprovalBroker_ResolveWithNoWaiter(t *testing.T) {
	ab := newApprovalBroker()
	if ab.resolve("evt1", "approved") {
		t.Error("resolve should return false when no waiter exists")
	}
}

func TestApprovalBroker_WaitAndResolve(t *testing.T) {
	ab := newApprovalBroker()
	done := make(chan string, 1)
	go func() {
		d, ok := ab.wait("evt1", 2*time.Second)
		if !ok {
			done <- "timeout"
			return
		}
		done <- d
	}()
	time.Sleep(10 * time.Millisecond)
	ab.resolve("evt1", "approved")
	select {
	case d := <-done:
		if d != "approved" {
			t.Errorf("expected approved, got %q", d)
		}
	case <-time.After(3 * time.Second):
		t.Error("wait did not resolve in time")
	}
}

func TestApprovalBroker_Timeout(t *testing.T) {
	ab := newApprovalBroker()
	_, ok := ab.wait("evt1", 30*time.Millisecond)
	if ok {
		t.Error("expected timeout, got resolved")
	}
}

// ── resolveApproval ───────────────────────────────────────────────────────────

func TestResolveApproval_InvalidDecision(t *testing.T) {
	ah, _, _ := newAgentHandlerForTest(t)
	r := formRequest(http.MethodPost, "/agents/a1/approvals/e1",
		map[string]string{"id": "a1", "eventId": "e1"}, "decision=maybe")
	w := httptest.NewRecorder()
	ah.resolveApproval(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestResolveApproval_StoresResultAndRedirects(t *testing.T) {
	ah, ev, _ := newAgentHandlerForTest(t)
	r := formRequest(http.MethodPost, "/agents/a1/approvals/e1",
		map[string]string{"id": "a1", "eventId": "e1"}, "decision=approved")
	w := httptest.NewRecorder()
	ah.resolveApproval(w, r)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected redirect, got %d", w.Code)
	}
	events, _ := ev.ListByAgent("a1")
	if len(events) == 0 {
		t.Fatal("expected approval result event stored")
	}
	last := events[len(events)-1]
	if last.Type != event.EventTypeApprovalResult || last.Decision != "approved" || last.RefID != "e1" {
		t.Errorf("unexpected result event: %+v", last)
	}
}

func TestResolveApproval_NotifiesWaiter(t *testing.T) {
	ah, _, _ := newAgentHandlerForTest(t)
	done := make(chan string, 1)
	go func() {
		d, ok := ah.approvals.wait("e1", 2*time.Second)
		if !ok {
			done <- "timeout"
			return
		}
		done <- d
	}()
	time.Sleep(10 * time.Millisecond)

	r := formRequest(http.MethodPost, "/agents/a1/approvals/e1",
		map[string]string{"id": "a1", "eventId": "e1"}, "decision=denied")
	ah.resolveApproval(httptest.NewRecorder(), r)

	select {
	case d := <-done:
		if d != "denied" {
			t.Errorf("expected denied, got %q", d)
		}
	case <-time.After(3 * time.Second):
		t.Error("waiter not notified")
	}
}

// ── broadcaster ───────────────────────────────────────────────────────────────

func TestBroadcaster_PublishReceive(t *testing.T) {
	bc := newBroadcaster()
	ch, unsub := bc.subscribe("a1")
	defer unsub()

	e := event.Event{ID: "e1", AgentID: "a1", Type: event.EventTypeAction, Summary: "did thing"}
	bc.publish(e)

	select {
	case got := <-ch:
		if got.ID != "e1" {
			t.Errorf("expected e1, got %q", got.ID)
		}
	case <-time.After(time.Second):
		t.Error("did not receive event within timeout")
	}
}

func TestBroadcaster_UnsubscribeCloses(t *testing.T) {
	bc := newBroadcaster()
	ch, unsub := bc.subscribe("a1")
	unsub()
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected closed channel after unsubscribe")
		}
	case <-time.After(time.Second):
		t.Error("channel not closed after unsubscribe")
	}
}

func TestBroadcaster_IsolatedByAgent(t *testing.T) {
	bc := newBroadcaster()
	ch, unsub := bc.subscribe("a1")
	defer unsub()
	bc.publish(event.Event{ID: "e1", AgentID: "a2", Summary: "other"})
	select {
	case <-ch:
		t.Error("should not receive event for a different agent")
	case <-time.After(50 * time.Millisecond):
		// correct — nothing received
	}
}
