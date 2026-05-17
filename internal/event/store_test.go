package event

import (
	"fmt"
	"testing"
	"time"
)

func TestAppend_And_ListByAgent(t *testing.T) {
	s := NewStore(t.TempDir())
	e := Event{ID: "e1", AgentID: "a1", Type: EventTypeAction, Timestamp: time.Now().Unix(), Summary: "Did a thing"}
	if err := s.Append(e); err != nil {
		t.Fatalf("Append: %v", err)
	}
	events, err := s.ListByAgent("a1")
	if err != nil {
		t.Fatalf("ListByAgent: %v", err)
	}
	if len(events) != 1 || events[0].ID != "e1" {
		t.Errorf("unexpected events: %+v", events)
	}
}

func TestListByAgent_Empty(t *testing.T) {
	s := NewStore(t.TempDir())
	events, err := s.ListByAgent("nobody")
	if err != nil {
		t.Fatalf("ListByAgent: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected empty, got %d", len(events))
	}
}

func TestPendingApprovals_NoPending(t *testing.T) {
	s := NewStore(t.TempDir())
	s.Append(Event{ID: "e1", AgentID: "a1", Type: EventTypeAction, Summary: "ok"})
	pending, err := s.PendingApprovals("a1")
	if err != nil {
		t.Fatalf("PendingApprovals: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("expected no pending, got %d", len(pending))
	}
}

func TestPendingApprovals_OneUnresolved(t *testing.T) {
	s := NewStore(t.TempDir())
	s.Append(Event{ID: "req1", AgentID: "a1", Type: EventTypeApprovalRequest, Summary: "send invoice?"})
	pending, err := s.PendingApprovals("a1")
	if err != nil {
		t.Fatalf("PendingApprovals: %v", err)
	}
	if len(pending) != 1 || pending[0].ID != "req1" {
		t.Errorf("expected req1, got %+v", pending)
	}
}

func TestPendingApprovals_ResolvedExcluded(t *testing.T) {
	s := NewStore(t.TempDir())
	s.Append(Event{ID: "req1", AgentID: "a1", Type: EventTypeApprovalRequest, Summary: "send invoice?"})
	s.Append(Event{ID: "res1", AgentID: "a1", Type: EventTypeApprovalResult, RefID: "req1", Decision: "approved"})
	pending, err := s.PendingApprovals("a1")
	if err != nil {
		t.Fatalf("PendingApprovals: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("expected 0 pending after resolution, got %d", len(pending))
	}
}

func TestPendingApprovals_ExpiredExcluded(t *testing.T) {
	s := NewStore(t.TempDir())
	s.Append(Event{
		ID: "req1", AgentID: "a1", Type: EventTypeApprovalRequest,
		Summary:   "old request",
		ExpiresAt: time.Now().Add(-time.Minute).Unix(),
	})
	pending, _ := s.PendingApprovals("a1")
	if len(pending) != 0 {
		t.Errorf("expected expired request excluded, got %d", len(pending))
	}
}

func TestRecentAll_AcrossAgents(t *testing.T) {
	s := NewStore(t.TempDir())
	s.Append(Event{ID: "e1", AgentID: "a1", Type: EventTypeAction, Timestamp: 100, Summary: "old"})
	s.Append(Event{ID: "e2", AgentID: "a2", Type: EventTypeAction, Timestamp: 200, Summary: "new"})
	s.Append(Event{ID: "e3", AgentID: "a1", Type: EventTypeAction, Timestamp: 150, Summary: "mid"})

	all, err := s.RecentAll(10)
	if err != nil {
		t.Fatalf("RecentAll: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 events, got %d", len(all))
	}
	if all[0].ID != "e2" {
		t.Errorf("expected newest first, got %q", all[0].ID)
	}
}

func TestRecentAll_Capped(t *testing.T) {
	s := NewStore(t.TempDir())
	for i := range 10 {
		s.Append(Event{ID: fmt.Sprintf("e%d", i), AgentID: "a1", Type: EventTypeAction, Timestamp: int64(i), Summary: "x"})
	}
	all, _ := s.RecentAll(5)
	if len(all) != 5 {
		t.Errorf("expected 5, got %d", len(all))
	}
}

func TestRecentAll_Empty(t *testing.T) {
	s := NewStore(t.TempDir())
	all, err := s.RecentAll(10)
	if err != nil {
		t.Fatalf("RecentAll: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected empty, got %d", len(all))
	}
}
