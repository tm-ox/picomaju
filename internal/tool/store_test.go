package tool

import (
	"path/filepath"
	"testing"
)

func newToolStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(filepath.Join(t.TempDir(), "tools.json"))
}

func TestToolStore_Empty(t *testing.T) {
	s := newToolStore(t)
	tools, err := s.List()
	if err != nil || len(tools) != 0 {
		t.Errorf("expected empty list, got %v err=%v", tools, err)
	}
}

func TestToolStore_CreateAndGet(t *testing.T) {
	s := newToolStore(t)
	tl := &Tool{ID: "wa1", Label: "WhatsApp", Type: "whatsapp"}
	if err := s.Create(tl); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := s.Get("wa1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Label != "WhatsApp" || got.Type != "whatsapp" {
		t.Errorf("unexpected tool: %+v", got)
	}
}

func TestToolStore_Create_DuplicateID(t *testing.T) {
	s := newToolStore(t)
	s.Create(&Tool{ID: "t1", Label: "A"})
	if err := s.Create(&Tool{ID: "t1", Label: "B"}); err == nil {
		t.Error("expected error on duplicate ID")
	}
}

func TestToolStore_Get_NotFound(t *testing.T) {
	s := newToolStore(t)
	if _, err := s.Get("nope"); err == nil {
		t.Error("expected error for missing tool")
	}
}

func TestToolStore_Update(t *testing.T) {
	s := newToolStore(t)
	s.Create(&Tool{ID: "t1", Label: "Old", Type: "email"})
	if err := s.Update(&Tool{ID: "t1", Label: "New", Type: "email"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := s.Get("t1")
	if got.Label != "New" {
		t.Errorf("update not persisted: %q", got.Label)
	}
}

func TestToolStore_Update_NotFound(t *testing.T) {
	s := newToolStore(t)
	if err := s.Update(&Tool{ID: "nope"}); err == nil {
		t.Error("expected error updating non-existent tool")
	}
}

func TestToolStore_Delete(t *testing.T) {
	s := newToolStore(t)
	s.Create(&Tool{ID: "t1", Label: "A"})
	if err := s.Delete("t1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get("t1"); err == nil {
		t.Error("expected tool to be gone")
	}
}

func TestToolStore_Delete_NotFound(t *testing.T) {
	s := newToolStore(t)
	if err := s.Delete("nope"); err == nil {
		t.Error("expected error deleting non-existent tool")
	}
}

func TestToolStore_WithConfig(t *testing.T) {
	s := newToolStore(t)
	tl := &Tool{ID: "t1", Label: "WA", Type: "whatsapp", Config: map[string]any{"token": "abc123"}}
	s.Create(tl)
	got, _ := s.Get("t1")
	if got.Config["token"] != "abc123" {
		t.Errorf("config not persisted: %v", got.Config)
	}
}

func TestCatalogByType(t *testing.T) {
	index := CatalogByType()
	if len(index) == 0 {
		t.Fatal("CatalogByType returned empty map")
	}
	// whatsapp is a known catalog entry
	if _, ok := index["whatsapp"]; !ok {
		t.Error("expected whatsapp in catalog index")
	}
	for typ, entry := range index {
		if entry.Type != typ {
			t.Errorf("key %q maps to entry with type %q", typ, entry.Type)
		}
	}
}
