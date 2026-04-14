package sop

import (
	"os"
	"path/filepath"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(t.TempDir())
}

func baseSOP() *SOP {
	return &SOP{
		ID: "test", Title: "Test", Version: 1,
		Priority: 50, Trigger: "on request",
		Category: "tasks",
		Body:     "Do the thing.",
	}
}

func TestStore_CreateAndGet(t *testing.T) {
	s := testStore(t)
	sop := baseSOP()
	if err := s.Create(sop); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("test")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != sop.ID {
		t.Errorf("ID = %q, want %q", got.ID, sop.ID)
	}
	if got.Category != sop.Category {
		t.Errorf("Category = %q, want %q", got.Category, sop.Category)
	}
	if got.Body != sop.Body {
		t.Errorf("Body = %q, want %q", got.Body, sop.Body)
	}
}

func TestStore_List(t *testing.T) {
	s := testStore(t)
	for i, id := range []string{"a", "b", "c"} {
		_ = i
		if err := s.Create(&SOP{
			ID: id, Title: id, Version: 1,
			Priority: 50, Trigger: "t", Category: "tasks",
		}); err != nil {
			t.Fatal(err)
		}
	}
	sops, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(sops) != 3 {
		t.Errorf("expected 3 sops, got %d", len(sops))
	}
}

func TestStore_ListEmpty(t *testing.T) {
	s := testStore(t)
	sops, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(sops) != 0 {
		t.Errorf("expected 0 sops, got %d", len(sops))
	}
}

func TestStore_ListMissingDir(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "nonexistent"))
	sops, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if sops != nil {
		t.Errorf("expected nil, got %v", sops)
	}
}

func TestStore_Update(t *testing.T) {
	s := testStore(t)
	sop := baseSOP()
	s.Create(sop)

	sop.Title = "Updated"
	sop.Category = "communication"
	sop.Body = "updated body"
	if err := s.Update(sop); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get("test")
	if got.Title != "Updated" {
		t.Errorf("Title = %q", got.Title)
	}
	if got.Category != "communication" {
		t.Errorf("Category = %q", got.Category)
	}
}

func TestStore_Delete(t *testing.T) {
	s := testStore(t)
	s.Create(baseSOP())
	if err := s.Delete("test"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get("test"); err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestStore_DeleteNotFound(t *testing.T) {
	s := testStore(t)
	if err := s.Delete("nope"); err == nil {
		t.Error("expected error deleting nonexistent sop, got nil")
	}
}

func TestStore_GetNotFound(t *testing.T) {
	s := testStore(t)
	if _, err := s.Get("nope"); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestStore_SkipsNonMdFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not an sop"), 0644)
	s := NewStore(dir)
	sops, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(sops) != 0 {
		t.Errorf("expected 0 sops, got %d", len(sops))
	}
}

func TestParseFrontmatter_RoundTrip(t *testing.T) {
	sop := &SOP{
		ID: "rt", Title: "Round Trip", Version: 2,
		Priority: 75, Trigger: "on event",
		Category: "escalation",
		Body:     "multi\nline\nbody",
	}
	store := NewStore(t.TempDir())
	store.Create(sop)

	got, err := store.Get("rt")
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != 2 {
		t.Errorf("Version = %d", got.Version)
	}
	if got.Category != "escalation" {
		t.Errorf("Category = %q", got.Category)
	}
	if got.Body != sop.Body {
		t.Errorf("Body = %q, want %q", got.Body, sop.Body)
	}
}
