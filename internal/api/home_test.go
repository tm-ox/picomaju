package api

import (
	"path/filepath"
	"testing"
	"time"

	"picomaju/internal/chat"
	"picomaju/internal/license"
	"picomaju/internal/picoclaw"
	"picomaju/internal/session"
	"picomaju/internal/staff"
)

func newHomeHandler(t *testing.T) *uiHandler {
	t.Helper()
	dir := t.TempDir()
	h := &uiHandler{
		staff:    staff.NewStore(filepath.Join(dir, "staff.json")),
		chats:    chat.NewStore(filepath.Join(dir, "chats.json")),
		license:  license.NewStore(filepath.Join(dir, "license.json")),
		sessions: session.NewStore(),
		picoclaw: picoclaw.NewManager(),
	}
	return h
}

func TestHomeData_Empty(t *testing.T) {
	h := newHomeHandler(t)
	d := h.homeData()
	if d.AgentCount != 0 {
		t.Errorf("expected 0 agents, got %d", d.AgentCount)
	}
	if d.ActiveCount != 0 {
		t.Errorf("expected 0 active, got %d", d.ActiveCount)
	}
	if d.MessagesToday != 0 {
		t.Errorf("expected 0 messages, got %d", d.MessagesToday)
	}
	if d.Credits != "Free" {
		t.Errorf("expected Free, got %q", d.Credits)
	}
}

func TestHomeData_AgentCount(t *testing.T) {
	h := newHomeHandler(t)
	h.staff.Create(&staff.Staff{ID: "s1", Label: "Support"})
	h.staff.Create(&staff.Staff{ID: "s2", Label: "Sales"})
	d := h.homeData()
	if d.AgentCount != 2 {
		t.Errorf("expected 2 agents, got %d", d.AgentCount)
	}
	if len(d.Agents) != 2 {
		t.Errorf("expected 2 agent status entries, got %d", len(d.Agents))
	}
}

func TestHomeData_MessagesToday(t *testing.T) {
	h := newHomeHandler(t)
	h.staff.Create(&staff.Staff{ID: "s1", Label: "Support"})
	now := time.Now().Unix()
	yesterday := time.Now().Add(-25 * time.Hour).Unix()
	h.chats.Create(&chat.Chat{
		ID:      "c1",
		StaffID: "s1",
		Title:   "Chat 1",
		Messages: []chat.Message{
			{Role: "user", Content: "hello", Timestamp: now},
			{Role: "assistant", Content: "hi", Timestamp: now},
			{Role: "user", Content: "old msg", Timestamp: yesterday},
		},
	})
	d := h.homeData()
	if d.MessagesToday != 2 {
		t.Errorf("expected 2 messages today, got %d", d.MessagesToday)
	}
}

func TestHomeData_Credits_Creditsплан(t *testing.T) {
	h := newHomeHandler(t)
	h.license.Save(&license.License{Active: true, Plan: license.PlanCredits, CreditsRemaining: 42})
	d := h.homeData()
	if d.Credits != "42" {
		t.Errorf("expected \"42\", got %q", d.Credits)
	}
}

func TestHomeData_Credits_SubscriptionPlan(t *testing.T) {
	h := newHomeHandler(t)
	h.license.Save(&license.License{Active: true, Plan: license.PlanStarter, ExpiresAt: time.Now().Add(24 * time.Hour).Unix()})
	d := h.homeData()
	if d.Credits != "Starter" {
		t.Errorf("expected \"Starter\", got %q", d.Credits)
	}
}

func TestHomeData_RecentChats_SortedNewestFirst(t *testing.T) {
	h := newHomeHandler(t)
	h.staff.Create(&staff.Staff{ID: "s1", Label: "Support"})
	h.chats.Create(&chat.Chat{ID: "c1", StaffID: "s1", Title: "Old", CreatedAt: 1000})
	h.chats.Create(&chat.Chat{ID: "c2", StaffID: "s1", Title: "New", CreatedAt: 2000})
	d := h.homeData()
	if len(d.RecentChats) != 2 {
		t.Fatalf("expected 2 recent chats, got %d", len(d.RecentChats))
	}
	if d.RecentChats[0].ID != "c2" {
		t.Errorf("expected newest chat first, got %q", d.RecentChats[0].ID)
	}
}

func TestHomeData_RecentChats_StaffLabel(t *testing.T) {
	h := newHomeHandler(t)
	h.staff.Create(&staff.Staff{ID: "s1", Label: "Support Bot"})
	h.chats.Create(&chat.Chat{ID: "c1", StaffID: "s1", Title: "Hello"})
	d := h.homeData()
	if len(d.RecentChats) == 0 {
		t.Fatal("expected recent chats")
	}
	if d.RecentChats[0].StaffLabel != "Support Bot" {
		t.Errorf("expected staff label, got %q", d.RecentChats[0].StaffLabel)
	}
}

func TestHomeData_RecentChats_LastMsgTruncated(t *testing.T) {
	h := newHomeHandler(t)
	h.staff.Create(&staff.Staff{ID: "s1", Label: "Agent"})
	long := "This is a very long message that exceeds the eighty character truncation limit set in the homeData function."
	h.chats.Create(&chat.Chat{
		ID: "c1", StaffID: "s1", Title: "T",
		Messages: []chat.Message{{Role: "user", Content: long, Timestamp: time.Now().Unix()}},
	})
	d := h.homeData()
	if len(d.RecentChats[0].LastMsg) > 83 { // 80 + "…"
		t.Errorf("last message not truncated, len=%d", len(d.RecentChats[0].LastMsg))
	}
}

func TestHomeData_RecentChats_CappedAt10(t *testing.T) {
	h := newHomeHandler(t)
	h.staff.Create(&staff.Staff{ID: "s1", Label: "Agent"})
	for i := range 15 {
		h.chats.Create(&chat.Chat{
			ID: string(rune('a' + i)), StaffID: "s1", Title: "Chat",
			CreatedAt: int64(i),
		})
	}
	d := h.homeData()
	if len(d.RecentChats) != 10 {
		t.Errorf("expected 10 recent chats, got %d", len(d.RecentChats))
	}
}

func TestHomeData_AgentStatus_Running(t *testing.T) {
	h := newHomeHandler(t)
	h.staff.Create(&staff.Staff{ID: "s1", Label: "Agent", Active: true})
	// No picoclaw process started — IsRunning should be false.
	d := h.homeData()
	if d.ActiveCount != 0 {
		t.Errorf("expected 0 active, got %d", d.ActiveCount)
	}
	if d.Agents[0].Running {
		t.Error("agent should not be running without a picoclaw process")
	}
}
