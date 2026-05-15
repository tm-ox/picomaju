package task

import (
	"path/filepath"
	"testing"
)

// ── validID ───────────────────────────────────────────────────────────────────

func TestValidID(t *testing.T) {
	valid := []string{"abc", "ABC", "a-b", "a_b", "a1", "A1Z", "task-001"}
	for _, id := range valid {
		if !validID(id) {
			t.Errorf("expected %q to be valid", id)
		}
	}
	invalid := []string{"", "a b", "a.b", "a/b", "a@b", "../evil"}
	for _, id := range invalid {
		if validID(id) {
			t.Errorf("expected %q to be invalid", id)
		}
	}
}

// ── Store ─────────────────────────────────────────────────────────────────────

func newTaskStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(filepath.Join(t.TempDir(), "tasks.json"))
}

func TestTaskStore_Empty(t *testing.T) {
	s := newTaskStore(t)
	tasks, err := s.List()
	if err != nil || len(tasks) != 0 {
		t.Errorf("expected empty list, got %v err=%v", tasks, err)
	}
}

func TestTaskStore_CreateAndGet(t *testing.T) {
	s := newTaskStore(t)
	tk := &Task{ID: "reply", Label: "Reply to customers", Description: "Answer inbound messages."}
	if err := s.Create(tk); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := s.Get("reply")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Label != "Reply to customers" {
		t.Errorf("unexpected label: %q", got.Label)
	}
}

func TestTaskStore_Create_InvalidID(t *testing.T) {
	s := newTaskStore(t)
	if err := s.Create(&Task{ID: "bad id!", Label: "x"}); err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestTaskStore_Create_DuplicateID(t *testing.T) {
	s := newTaskStore(t)
	s.Create(&Task{ID: "t1", Label: "A"})
	if err := s.Create(&Task{ID: "t1", Label: "B"}); err == nil {
		t.Error("expected error on duplicate ID")
	}
}

func TestTaskStore_Get_NotFound(t *testing.T) {
	s := newTaskStore(t)
	if _, err := s.Get("nope"); err == nil {
		t.Error("expected error for missing task")
	}
}

func TestTaskStore_Update(t *testing.T) {
	s := newTaskStore(t)
	s.Create(&Task{ID: "t1", Label: "Old"})
	if err := s.Update(&Task{ID: "t1", Label: "New"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := s.Get("t1")
	if got.Label != "New" {
		t.Errorf("update not persisted: %q", got.Label)
	}
}

func TestTaskStore_Update_NotFound(t *testing.T) {
	s := newTaskStore(t)
	if err := s.Update(&Task{ID: "nope"}); err == nil {
		t.Error("expected error updating non-existent task")
	}
}

func TestTaskStore_Delete(t *testing.T) {
	s := newTaskStore(t)
	s.Create(&Task{ID: "t1", Label: "A"})
	if err := s.Delete("t1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get("t1"); err == nil {
		t.Error("expected task to be gone")
	}
}

func TestTaskStore_Delete_NotFound(t *testing.T) {
	s := newTaskStore(t)
	if err := s.Delete("nope"); err == nil {
		t.Error("expected error deleting non-existent task")
	}
}

func TestTaskStore_List_Multiple(t *testing.T) {
	s := newTaskStore(t)
	s.Create(&Task{ID: "t1", Label: "A"})
	s.Create(&Task{ID: "t2", Label: "B"})
	tasks, err := s.List()
	if err != nil || len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d err=%v", len(tasks), err)
	}
}
