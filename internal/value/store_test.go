package value

import (
	"path/filepath"
	"testing"
)

func newValueStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(filepath.Join(t.TempDir(), "values"))
}

func sampleValue() *Value {
	return &Value{
		ID:       "be-honest",
		Title:    "Be Honest",
		Version:  1,
		Priority: 80,
		Category: "core_values",
		Body:     "Always tell the truth, even when it's difficult.",
	}
}

// ── validID ───────────────────────────────────────────────────────────────────

func TestValueValidID(t *testing.T) {
	valid := []string{"abc", "be-honest", "v_1", "ABC123"}
	for _, id := range valid {
		if !validID(id) {
			t.Errorf("expected %q to be valid", id)
		}
	}
	invalid := []string{"", "a b", "a.b", "a/b", "../evil"}
	for _, id := range invalid {
		if validID(id) {
			t.Errorf("expected %q to be invalid", id)
		}
	}
}

// ── parseFrontmatter ─────────────────────────────────────────────────────────

func TestParseFrontmatter_Valid(t *testing.T) {
	raw := []byte("---\nid: test\ntitle: Test\nversion: 1\npriority: 50\ncategory: custom\n---\n\nBody text here.")
	v, err := parseFrontmatter(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ID != "test" || v.Title != "Test" || v.Version != 1 || v.Priority != 50 {
		t.Errorf("unexpected value: %+v", v)
	}
	if v.Body != "Body text here." {
		t.Errorf("unexpected body: %q", v.Body)
	}
}

func TestParseFrontmatter_EmptyBody(t *testing.T) {
	raw := []byte("---\nid: x\ntitle: X\nversion: 1\ncategory: custom\n---\n")
	v, err := parseFrontmatter(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Body != "" {
		t.Errorf("expected empty body, got %q", v.Body)
	}
}

func TestParseFrontmatter_MissingOpen(t *testing.T) {
	raw := []byte("no frontmatter here")
	if _, err := parseFrontmatter(raw); err == nil {
		t.Error("expected error for missing opening delimiter")
	}
}

func TestParseFrontmatter_MissingClose(t *testing.T) {
	raw := []byte("---\nid: x\n")
	if _, err := parseFrontmatter(raw); err == nil {
		t.Error("expected error for missing closing delimiter")
	}
}

// ── Store CRUD ────────────────────────────────────────────────────────────────

func TestValueStore_Empty(t *testing.T) {
	s := newValueStore(t)
	vals, err := s.List()
	if err != nil || len(vals) != 0 {
		t.Errorf("expected empty list, got %v err=%v", vals, err)
	}
}

func TestValueStore_CreateAndGet(t *testing.T) {
	s := newValueStore(t)
	v := sampleValue()
	if err := s.Create(v); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := s.Get("be-honest")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Be Honest" || got.Body != "Always tell the truth, even when it's difficult." {
		t.Errorf("unexpected value: %+v", got)
	}
}

func TestValueStore_Create_InvalidID(t *testing.T) {
	s := newValueStore(t)
	v := sampleValue()
	v.ID = "bad id!"
	if err := s.Create(v); err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestValueStore_Get_InvalidID(t *testing.T) {
	s := newValueStore(t)
	if _, err := s.Get("../evil"); err == nil {
		t.Error("expected error for invalid ID (path traversal)")
	}
}

func TestValueStore_Get_NotFound(t *testing.T) {
	s := newValueStore(t)
	if _, err := s.Get("nope"); err == nil {
		t.Error("expected error for missing value")
	}
}

func TestValueStore_Update(t *testing.T) {
	s := newValueStore(t)
	s.Create(sampleValue())
	updated := sampleValue()
	updated.Title = "Be Very Honest"
	updated.Body = "Updated body."
	if err := s.Update(updated); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := s.Get("be-honest")
	if got.Title != "Be Very Honest" || got.Body != "Updated body." {
		t.Errorf("update not persisted: %+v", got)
	}
}

func TestValueStore_Update_InvalidID(t *testing.T) {
	s := newValueStore(t)
	v := sampleValue()
	v.ID = "bad id!"
	if err := s.Update(v); err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestValueStore_Delete(t *testing.T) {
	s := newValueStore(t)
	s.Create(sampleValue())
	if err := s.Delete("be-honest"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get("be-honest"); err == nil {
		t.Error("expected value to be gone")
	}
}

func TestValueStore_Delete_NotFound(t *testing.T) {
	s := newValueStore(t)
	if err := s.Delete("nope"); err == nil {
		t.Error("expected error deleting non-existent value")
	}
}

func TestValueStore_Delete_InvalidID(t *testing.T) {
	s := newValueStore(t)
	if err := s.Delete("../evil"); err == nil {
		t.Error("expected error for invalid ID (path traversal)")
	}
}

func TestValueStore_List_Multiple(t *testing.T) {
	s := newValueStore(t)
	v1 := sampleValue()
	v2 := sampleValue()
	v2.ID = "speak-clearly"
	v2.Title = "Speak Clearly"
	s.Create(v1)
	s.Create(v2)
	vals, err := s.List()
	if err != nil || len(vals) != 2 {
		t.Errorf("expected 2 values, got %d err=%v", len(vals), err)
	}
}
