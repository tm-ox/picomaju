package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func newSettingsStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(filepath.Join(t.TempDir(), "settings.json"))
}

func TestSettingsStore_LoadMissingFile(t *testing.T) {
	s := newSettingsStore(t)
	cfg, err := s.Load()
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected empty settings, got nil")
	}
}

func TestSettingsStore_RoundTrip(t *testing.T) {
	s := newSettingsStore(t)
	orig := &Settings{
		BusinessName:    "Bali Surf Co",
		BusinessDetails: "Surf lessons and rentals.",
		Timezone:        "Asia/Makassar",
		Hours:           "8am–6pm",
		Languages:       []string{"en", "id"},
	}
	if err := s.Save(orig); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.BusinessName != orig.BusinessName {
		t.Errorf("BusinessName: got %q, want %q", got.BusinessName, orig.BusinessName)
	}
	if got.Timezone != orig.Timezone {
		t.Errorf("Timezone: got %q, want %q", got.Timezone, orig.Timezone)
	}
	if len(got.Languages) != 2 || got.Languages[0] != "en" {
		t.Errorf("Languages: %v", got.Languages)
	}
}

func TestSettingsStore_Load_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	os.WriteFile(path, []byte("not json {{{"), 0644)
	s := NewStore(path)
	if _, err := s.Load(); err == nil {
		t.Error("expected error loading corrupt JSON")
	}
}

func TestSettingsStore_Save_CreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "config")
	s := NewStore(filepath.Join(dir, "settings.json"))
	if err := s.Save(&Settings{BusinessName: "Test"}); err != nil {
		t.Fatalf("Save should create parent dirs: %v", err)
	}
}
