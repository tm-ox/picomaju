package chat

import (
	"path/filepath"
	"testing"
)

func newChatStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(filepath.Join(t.TempDir(), "chats.json"))
}

func TestChatStore_Empty(t *testing.T) {
	s := newChatStore(t)
	chats, err := s.ListByStaff("s1")
	if err != nil || len(chats) != 0 {
		t.Errorf("expected empty list, got %v err=%v", chats, err)
	}
}

func TestChatStore_CreateAndGet(t *testing.T) {
	s := newChatStore(t)
	c := &Chat{ID: "c1", StaffID: "s1", Title: "Hello"}
	if err := s.Create(c); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.CreatedAt == 0 {
		t.Error("CreatedAt should be set on create")
	}
	got, err := s.Get("c1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Hello" || got.StaffID != "s1" {
		t.Errorf("unexpected chat: %+v", got)
	}
}

func TestChatStore_Create_PreservesTimestamp(t *testing.T) {
	s := newChatStore(t)
	c := &Chat{ID: "c1", StaffID: "s1", CreatedAt: 12345}
	s.Create(c)
	got, _ := s.Get("c1")
	if got.CreatedAt != 12345 {
		t.Errorf("expected preserved timestamp 12345, got %d", got.CreatedAt)
	}
}

func TestChatStore_Get_NotFound(t *testing.T) {
	s := newChatStore(t)
	if _, err := s.Get("nope"); err == nil {
		t.Error("expected error for missing chat")
	}
}

func TestChatStore_ListByStaff_Filters(t *testing.T) {
	s := newChatStore(t)
	s.Create(&Chat{ID: "c1", StaffID: "s1", Title: "A"})
	s.Create(&Chat{ID: "c2", StaffID: "s1", Title: "B"})
	s.Create(&Chat{ID: "c3", StaffID: "s2", Title: "C"})
	chats, err := s.ListByStaff("s1")
	if err != nil || len(chats) != 2 {
		t.Errorf("expected 2 chats for s1, got %d err=%v", len(chats), err)
	}
}

func TestChatStore_Update(t *testing.T) {
	s := newChatStore(t)
	s.Create(&Chat{ID: "c1", StaffID: "s1", Title: "Old"})
	c := &Chat{ID: "c1", StaffID: "s1", Title: "New", Messages: []Message{
		{Role: "user", Content: "hi", Timestamp: 1},
	}}
	if err := s.Update(c); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := s.Get("c1")
	if got.Title != "New" || len(got.Messages) != 1 {
		t.Errorf("update not persisted: %+v", got)
	}
}

func TestChatStore_Update_NotFound(t *testing.T) {
	s := newChatStore(t)
	if err := s.Update(&Chat{ID: "nope"}); err == nil {
		t.Error("expected error updating non-existent chat")
	}
}

func TestChatStore_Delete(t *testing.T) {
	s := newChatStore(t)
	s.Create(&Chat{ID: "c1", StaffID: "s1", Title: "A"})
	if err := s.Delete("c1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get("c1"); err == nil {
		t.Error("expected chat to be gone")
	}
}

func TestChatStore_Delete_NotFound(t *testing.T) {
	s := newChatStore(t)
	if err := s.Delete("nope"); err == nil {
		t.Error("expected error deleting non-existent chat")
	}
}

func TestChatStore_List(t *testing.T) {
	s := newChatStore(t)
	s.Create(&Chat{ID: "c1", StaffID: "s1", Title: "A"})
	s.Create(&Chat{ID: "c2", StaffID: "s2", Title: "B"})
	all, err := s.List()
	if err != nil || len(all) != 2 {
		t.Errorf("expected 2 chats, got %d err=%v", len(all), err)
	}
}

func TestChatStore_List_Empty(t *testing.T) {
	s := newChatStore(t)
	all, err := s.List()
	if err != nil || len(all) != 0 {
		t.Errorf("expected empty list, got %d err=%v", len(all), err)
	}
}
