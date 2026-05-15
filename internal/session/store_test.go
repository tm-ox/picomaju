package session

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSession_CreateAndLookup(t *testing.T) {
	s := NewStore()
	id, err := s.Create("user1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty session ID")
	}
	uid, ok := s.Lookup(id)
	if !ok || uid != "user1" {
		t.Errorf("Lookup: ok=%v uid=%q", ok, uid)
	}
}

func TestSession_Lookup_NotFound(t *testing.T) {
	s := NewStore()
	_, ok := s.Lookup("bogus")
	if ok {
		t.Error("expected Lookup to return false for unknown session")
	}
}

func TestSession_Delete(t *testing.T) {
	s := NewStore()
	id, _ := s.Create("user1")
	s.Delete(id)
	_, ok := s.Lookup(id)
	if ok {
		t.Error("expected session to be gone after Delete")
	}
}

func TestSession_Delete_Idempotent(t *testing.T) {
	s := NewStore()
	s.Delete("nonexistent") // should not panic
}

func TestSession_IDs_AreUnique(t *testing.T) {
	s := NewStore()
	ids := make(map[string]bool)
	for range 10 {
		id, err := s.Create("u")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if ids[id] {
			t.Fatalf("duplicate session ID: %q", id)
		}
		ids[id] = true
	}
}

func TestSession_SetCookie(t *testing.T) {
	s := NewStore()
	w := httptest.NewRecorder()
	s.SetCookie(w, "test-session-id")
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected cookie to be set")
	}
	c := cookies[0]
	if c.Name != CookieName {
		t.Errorf("expected cookie name %q, got %q", CookieName, c.Name)
	}
	if c.Value != "test-session-id" {
		t.Errorf("expected cookie value %q, got %q", "test-session-id", c.Value)
	}
	if !c.HttpOnly {
		t.Error("expected HttpOnly cookie")
	}
}

func TestSession_ClearCookie(t *testing.T) {
	s := NewStore()
	w := httptest.NewRecorder()
	s.ClearCookie(w)
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected cookie to be set")
	}
	c := cookies[0]
	if c.MaxAge != -1 {
		t.Errorf("expected MaxAge=-1 to clear cookie, got %d", c.MaxAge)
	}
}

func TestSession_FromRequest(t *testing.T) {
	s := NewStore()
	id, _ := s.Create("user42")
	r := &http.Request{Header: http.Header{}}
	r.AddCookie(&http.Cookie{Name: CookieName, Value: id})
	uid, ok := s.FromRequest(r)
	if !ok || uid != "user42" {
		t.Errorf("FromRequest: ok=%v uid=%q", ok, uid)
	}
}

func TestSession_FromRequest_NoCookie(t *testing.T) {
	s := NewStore()
	r := &http.Request{Header: http.Header{}}
	_, ok := s.FromRequest(r)
	if ok {
		t.Error("expected false with no cookie")
	}
}

func TestSession_WithUser_AndCurrentUser(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	r2 := WithUser(r, "user99")
	uid, ok := CurrentUser(r2)
	if !ok || uid != "user99" {
		t.Errorf("CurrentUser: ok=%v uid=%q", ok, uid)
	}
}

func TestSession_CurrentUser_NotSet(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	_, ok := CurrentUser(r)
	if ok {
		t.Error("expected false when no user in context")
	}
}
