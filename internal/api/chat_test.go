package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"picomaju/internal/chat"
	"picomaju/internal/license"
	"picomaju/internal/picoclaw"
	"picomaju/internal/session"
	"picomaju/internal/settings"
	"picomaju/internal/staff"
	"picomaju/internal/user"
)

func newChatHandler(t *testing.T) (*uiHandler, string) {
	t.Helper()
	dir := t.TempDir()
	h := &uiHandler{
		staff:    staff.NewStore(filepath.Join(dir, "staff.json")),
		chats:    chat.NewStore(filepath.Join(dir, "chats.json")),
		license:  license.NewStore(filepath.Join(dir, "license.json")),
		settings: settings.NewStore(filepath.Join(dir, "settings.json")),
		users:    user.NewStore(filepath.Join(dir, "users.json")),
		sessions: session.NewStore(),
		picoclaw: picoclaw.NewManager(),
		dataDir:  dir,
	}
	return h, dir
}

func chiRequest(method, path string, routeParams map[string]string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	rctx := chi.NewRouteContext()
	for k, v := range routeParams {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestCreateChat_Redirect(t *testing.T) {
	h, _ := newChatHandler(t)
	h.staff.Create(&staff.Staff{ID: "s1", Label: "Agent"})

	r := chiRequest(http.MethodPost, "/staff/s1/chats", map[string]string{"id": "s1"})
	w := httptest.NewRecorder()
	h.createChat(w, r)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if !strings.HasPrefix(loc, "/staff/s1/chats/") {
		t.Errorf("unexpected redirect location: %q", loc)
	}
}

func TestCreateChat_UnknownStaff(t *testing.T) {
	h, _ := newChatHandler(t)

	r := chiRequest(http.MethodPost, "/staff/nope/chats", map[string]string{"id": "nope"})
	w := httptest.NewRecorder()
	h.createChat(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestCreateChat_NoWorkspace_NoUserMD(t *testing.T) {
	h, dir := newChatHandler(t)
	h.staff.Create(&staff.Staff{ID: "s1", Label: "Agent"})

	r := chiRequest(http.MethodPost, "/staff/s1/chats", map[string]string{"id": "s1"})
	w := httptest.NewRecorder()
	h.createChat(w, r)

	userMD := filepath.Join(dir, "agents", "workspace-s1", "USER.md")
	if _, err := os.Stat(userMD); !os.IsNotExist(err) {
		t.Error("USER.md should not be written when workspace does not exist")
	}
}

func TestCreateChat_WorkspaceExists_WritesUserMD(t *testing.T) {
	h, dir := newChatHandler(t)
	h.staff.Create(&staff.Staff{ID: "s1", Label: "Agent"})

	wsDir := filepath.Join(dir, "agents", "workspace-s1")
	os.MkdirAll(wsDir, 0755)

	u := &user.User{ID: "u1", Name: "Alice", Role: user.RoleOwner, Description: "Handles support"}
	h.users.Create(u)

	r := chiRequest(http.MethodPost, "/staff/s1/chats", map[string]string{"id": "s1"})
	r = session.WithUser(r, "u1")
	w := httptest.NewRecorder()
	h.createChat(w, r)

	userMD := filepath.Join(wsDir, "USER.md")
	b, err := os.ReadFile(userMD)
	if err != nil {
		t.Fatalf("USER.md not written: %v", err)
	}
	content := string(b)
	if !strings.Contains(content, "Alice") {
		t.Errorf("USER.md missing user name, got:\n%s", content)
	}
	if !strings.Contains(content, "owner") {
		t.Errorf("USER.md missing role, got:\n%s", content)
	}
	if !strings.Contains(content, "Handles support") {
		t.Errorf("USER.md missing description, got:\n%s", content)
	}
}

func TestCreateChat_WorkspaceExists_NoSession_WritesBusinessOnly(t *testing.T) {
	h, dir := newChatHandler(t)
	h.staff.Create(&staff.Staff{ID: "s1", Label: "Agent"})

	wsDir := filepath.Join(dir, "agents", "workspace-s1")
	os.MkdirAll(wsDir, 0755)

	cfg := &settings.Settings{BusinessName: "Bali Surf Co"}
	h.settings.Save(cfg)

	r := chiRequest(http.MethodPost, "/staff/s1/chats", map[string]string{"id": "s1"})
	w := httptest.NewRecorder()
	h.createChat(w, r)

	b, err := os.ReadFile(filepath.Join(wsDir, "USER.md"))
	if err != nil {
		t.Fatalf("USER.md not written: %v", err)
	}
	content := string(b)
	if strings.Contains(content, "Current User") {
		t.Errorf("USER.md should not contain user section without session, got:\n%s", content)
	}
	if !strings.Contains(content, "Bali Surf Co") {
		t.Errorf("USER.md missing business name, got:\n%s", content)
	}
}
