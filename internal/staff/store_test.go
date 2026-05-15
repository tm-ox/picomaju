package staff

import (
	"path/filepath"
	"testing"
)

func newStaffStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(filepath.Join(t.TempDir(), "staff.json"))
}

func TestStaffStore_Empty(t *testing.T) {
	s := newStaffStore(t)
	members, err := s.List()
	if err != nil || len(members) != 0 {
		t.Errorf("expected empty list, got %v err=%v", members, err)
	}
}

func TestStaffStore_CreateAndGet(t *testing.T) {
	s := newStaffStore(t)
	m := &Staff{ID: "s1", Label: "Support", Description: "Customer support agent", Active: true}
	if err := s.Create(m); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := s.Get("s1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Label != "Support" || !got.Active {
		t.Errorf("unexpected staff: %+v", got)
	}
}

func TestStaffStore_Create_DuplicateID(t *testing.T) {
	s := newStaffStore(t)
	s.Create(&Staff{ID: "s1", Label: "A"})
	if err := s.Create(&Staff{ID: "s1", Label: "B"}); err == nil {
		t.Error("expected error on duplicate ID")
	}
}

func TestStaffStore_Get_NotFound(t *testing.T) {
	s := newStaffStore(t)
	if _, err := s.Get("nope"); err == nil {
		t.Error("expected error for missing staff")
	}
}

func TestStaffStore_Update(t *testing.T) {
	s := newStaffStore(t)
	s.Create(&Staff{ID: "s1", Label: "Old"})
	if err := s.Update(&Staff{ID: "s1", Label: "New", Active: true}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := s.Get("s1")
	if got.Label != "New" || !got.Active {
		t.Errorf("update not persisted: %+v", got)
	}
}

func TestStaffStore_Update_NotFound(t *testing.T) {
	s := newStaffStore(t)
	if err := s.Update(&Staff{ID: "nope"}); err == nil {
		t.Error("expected error updating non-existent staff")
	}
}

func TestStaffStore_Delete(t *testing.T) {
	s := newStaffStore(t)
	s.Create(&Staff{ID: "s1", Label: "A"})
	if err := s.Delete("s1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get("s1"); err == nil {
		t.Error("expected staff to be gone")
	}
}

func TestStaffStore_Delete_NotFound(t *testing.T) {
	s := newStaffStore(t)
	if err := s.Delete("nope"); err == nil {
		t.Error("expected error deleting non-existent staff")
	}
}

func TestStaffStore_TasksAndValues(t *testing.T) {
	s := newStaffStore(t)
	m := &Staff{
		ID:              "s1",
		Label:           "Agent",
		Tasks:           []string{"t1", "t2"},
		Values:          []string{"v1"},
		ValueCategories: []string{"core_values"},
	}
	s.Create(m)
	got, _ := s.Get("s1")
	if len(got.Tasks) != 2 || len(got.Values) != 1 || len(got.ValueCategories) != 1 {
		t.Errorf("slices not persisted: %+v", got)
	}
}
