package user

import (
	"path/filepath"
	"testing"
)

// ── Role ─────────────────────────────────────────────────────────────────────

func TestRoleLabel(t *testing.T) {
	cases := []struct{ role Role; want string }{
		{RoleOwner, "Owner"},
		{RoleManager, "Manager"},
		{RoleStaff, "Staff"},
		{"unknown", "unknown"},
	}
	for _, c := range cases {
		if got := c.role.Label(); got != c.want {
			t.Errorf("role %q: got %q, want %q", c.role, got, c.want)
		}
	}
}

func TestPermissions(t *testing.T) {
	owner := &User{Role: RoleOwner}
	manager := &User{Role: RoleManager}
	staff := &User{Role: RoleStaff}

	if !owner.CanManageUsers() || !owner.CanManageSettings() || !owner.CanManageBilling() {
		t.Error("owner should have all permissions")
	}
	if manager.CanManageUsers() || manager.CanManageSettings() || manager.CanManageBilling() {
		t.Error("manager should have no owner permissions")
	}
	if staff.CanManageUsers() || staff.CanManageSettings() || staff.CanManageBilling() {
		t.Error("staff should have no owner permissions")
	}
}

// ── PIN ───────────────────────────────────────────────────────────────────────

func TestSetAndCheckPIN(t *testing.T) {
	u := &User{}
	if err := u.SetPIN("1234"); err != nil {
		t.Fatalf("SetPIN: %v", err)
	}
	if u.PINHash == "" {
		t.Error("PINHash should be set")
	}
	if !u.CheckPIN("1234") {
		t.Error("correct PIN should pass")
	}
	if u.CheckPIN("0000") {
		t.Error("wrong PIN should fail")
	}
}

func TestCheckPIN_EmptyHash(t *testing.T) {
	u := &User{}
	if u.CheckPIN("1234") {
		t.Error("empty hash should always fail")
	}
}

// ── Store ─────────────────────────────────────────────────────────────────────

func newUserStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(filepath.Join(t.TempDir(), "users.json"))
}

func TestUserStore_Empty(t *testing.T) {
	s := newUserStore(t)
	users, err := s.List()
	if err != nil || len(users) != 0 {
		t.Errorf("expected empty list, got %v err=%v", users, err)
	}
	n, err := s.Count()
	if err != nil || n != 0 {
		t.Errorf("expected count 0, got %d err=%v", n, err)
	}
}

func TestUserStore_CreateAndGet(t *testing.T) {
	s := newUserStore(t)
	u := &User{ID: "u1", Name: "Alice", Role: RoleOwner}
	if err := s.Create(u); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := s.Get("u1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "Alice" || got.Role != RoleOwner {
		t.Errorf("unexpected user: %+v", got)
	}
}

func TestUserStore_Create_DuplicateID(t *testing.T) {
	s := newUserStore(t)
	u := &User{ID: "u1", Name: "Alice"}
	s.Create(u)
	if err := s.Create(u); err == nil {
		t.Error("expected error on duplicate ID")
	}
}

func TestUserStore_Get_NotFound(t *testing.T) {
	s := newUserStore(t)
	if _, err := s.Get("nope"); err == nil {
		t.Error("expected error for missing user")
	}
}

func TestUserStore_Update(t *testing.T) {
	s := newUserStore(t)
	s.Create(&User{ID: "u1", Name: "Alice", Role: RoleStaff})
	if err := s.Update(&User{ID: "u1", Name: "Alice B", Role: RoleManager}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := s.Get("u1")
	if got.Name != "Alice B" || got.Role != RoleManager {
		t.Errorf("update not persisted: %+v", got)
	}
}

func TestUserStore_Update_NotFound(t *testing.T) {
	s := newUserStore(t)
	if err := s.Update(&User{ID: "nope"}); err == nil {
		t.Error("expected error updating non-existent user")
	}
}

func TestUserStore_Delete(t *testing.T) {
	s := newUserStore(t)
	s.Create(&User{ID: "u1", Name: "Alice"})
	if err := s.Delete("u1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get("u1"); err == nil {
		t.Error("expected user to be gone after delete")
	}
}

func TestUserStore_Delete_NotFound(t *testing.T) {
	s := newUserStore(t)
	if err := s.Delete("nope"); err == nil {
		t.Error("expected error deleting non-existent user")
	}
}

func TestUserStore_Count(t *testing.T) {
	s := newUserStore(t)
	for i, name := range []string{"Alice", "Bob", "Carol"} {
		s.Create(&User{ID: string(rune('a' + i)), Name: name})
	}
	n, err := s.Count()
	if err != nil || n != 3 {
		t.Errorf("expected count 3, got %d err=%v", n, err)
	}
}
