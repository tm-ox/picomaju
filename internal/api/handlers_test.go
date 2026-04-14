package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"picomaju/internal/category"
	"picomaju/internal/role"
	"picomaju/internal/settings"
	"picomaju/internal/sop"
)

func newTestRouter(t *testing.T) (*chiRouter, *sop.Store, *role.Store) {
	t.Helper()
	dir := t.TempDir()
	sopStore := sop.NewStore(dir + "/sops")
	roleStore := role.NewStore(dir + "/roles.json")
	catStore := category.NewStore(dir + "/categories.json")
	settingsStore := settings.NewStore(dir + "/settings.json")
	return &chiRouter{r: NewRouter(sopStore, roleStore, catStore, settingsStore, dir, http.Dir("."))}, sopStore, roleStore
}

type chiRouter struct {
	r http.Handler
}

func (cr *chiRouter) do(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	cr.r.ServeHTTP(w, req)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.NewDecoder(w.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v\nbody: %s", err, w.Body.String())
	}
}

func validSOP(id string) sop.SOP {
	return sop.SOP{
		ID: id, Title: id, Version: 1,
		Priority: 50, Trigger: "t", Category: "tasks",
	}
}

// --- SOP handlers ---

func TestSOPList_Empty(t *testing.T) {
	r, _, _ := newTestRouter(t)
	w := r.do(t, http.MethodGet, "/api/sops", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var sops []*sop.SOP
	decode(t, w, &sops)
	if len(sops) != 0 {
		t.Errorf("expected empty list, got %d", len(sops))
	}
}

func TestSOPCreate(t *testing.T) {
	r, _, _ := newTestRouter(t)
	w := r.do(t, http.MethodPost, "/api/sops", validSOP("test"))
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestSOPGet(t *testing.T) {
	r, store, _ := newTestRouter(t)
	s := validSOP("x")
	store.Create(&s)
	w := r.do(t, http.MethodGet, "/api/sops/x", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var got sop.SOP
	decode(t, w, &got)
	if got.ID != "x" {
		t.Errorf("ID = %q", got.ID)
	}
}

func TestSOPGet_NotFound(t *testing.T) {
	r, _, _ := newTestRouter(t)
	w := r.do(t, http.MethodGet, "/api/sops/nope", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestSOPUpdate(t *testing.T) {
	r, store, _ := newTestRouter(t)
	s := validSOP("upd")
	store.Create(&s)
	s.Title = "New"
	s.Version = 2
	w := r.do(t, http.MethodPut, "/api/sops/upd", s)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestSOPDelete(t *testing.T) {
	r, store, _ := newTestRouter(t)
	s := validSOP("del")
	store.Create(&s)
	w := r.do(t, http.MethodDelete, "/api/sops/del", nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d", w.Code)
	}
	w2 := r.do(t, http.MethodGet, "/api/sops/del", nil)
	if w2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", w2.Code)
	}
}

func TestSOPValidate_Valid(t *testing.T) {
	r, store, _ := newTestRouter(t)
	s := validSOP("v")
	store.Create(&s)
	w := r.do(t, http.MethodPost, "/api/sops/v/validate", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var res sop.ValidationResult
	decode(t, w, &res)
	if !res.Valid {
		t.Errorf("expected valid, got errors: %v", res.Errors)
	}
}

func TestSOPValidate_Invalid(t *testing.T) {
	r, store, _ := newTestRouter(t)
	bad := sop.SOP{ID: "bad", Title: "Bad", Version: 1, Priority: 50, Trigger: "", Category: ""}
	store.Create(&bad)
	w := r.do(t, http.MethodPost, "/api/sops/bad/validate", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var res sop.ValidationResult
	decode(t, w, &res)
	if res.Valid {
		t.Error("expected invalid")
	}
}

// --- Role handlers ---

func TestRoleList_Empty(t *testing.T) {
	r, _, _ := newTestRouter(t)
	w := r.do(t, http.MethodGet, "/api/roles", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var roles []role.Role
	decode(t, w, &roles)
	if len(roles) != 0 {
		t.Errorf("expected empty, got %d", len(roles))
	}
}

func TestRoleCompile_NoSOPs(t *testing.T) {
	r, _, roleStore := newTestRouter(t)
	roleStore.Create(&role.Role{
		ID: "agent", Label: "Agent",
		Categories: []string{"tasks"},
	})
	w := r.do(t, http.MethodPost, "/api/roles/agent/compile", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var res sop.CompileResult
	decode(t, w, &res)
	if res.Role != "agent" {
		t.Errorf("role = %q", res.Role)
	}
	if len(res.Policies) != 0 {
		t.Errorf("expected 0 policies, got %d", len(res.Policies))
	}
}

func TestRoleCompile_WithSOPsFromCategory(t *testing.T) {
	r, sopStore, roleStore := newTestRouter(t)
	s := sop.SOP{ID: "s1", Title: "S1", Version: 1, Priority: 50, Trigger: "t", Category: "tasks", Body: "do it"}
	sopStore.Create(&s)
	roleStore.Create(&role.Role{ID: "agent", Label: "Agent", Categories: []string{"tasks"}})

	w := r.do(t, http.MethodPost, "/api/roles/agent/compile", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var res sop.CompileResult
	decode(t, w, &res)
	if len(res.Policies) != 1 {
		t.Errorf("expected 1 policy, got %d", len(res.Policies))
	}
}

func TestRoleCompile_WithIndividualSOP(t *testing.T) {
	r, sopStore, roleStore := newTestRouter(t)
	s := sop.SOP{ID: "s1", Title: "S1", Version: 1, Priority: 50, Trigger: "t", Category: "communication", Body: "do it"}
	sopStore.Create(&s)
	roleStore.Create(&role.Role{ID: "agent", Label: "Agent", SOPs: []string{"s1"}})

	w := r.do(t, http.MethodPost, "/api/roles/agent/compile", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var res sop.CompileResult
	decode(t, w, &res)
	if len(res.Policies) != 1 {
		t.Errorf("expected 1 policy, got %d", len(res.Policies))
	}
}

func TestRoleCompile_InvalidSOP_Returns422(t *testing.T) {
	r, sopStore, roleStore := newTestRouter(t)
	bad := sop.SOP{ID: "bad", Title: "Bad", Version: 1, Priority: 50, Trigger: "", Category: "tasks"}
	sopStore.Create(&bad)
	roleStore.Create(&role.Role{ID: "agent", Label: "Agent", Categories: []string{"tasks"}})

	w := r.do(t, http.MethodPost, "/api/roles/agent/compile", nil)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Code)
	}
}

func TestRoleCompile_NotFound(t *testing.T) {
	r, _, _ := newTestRouter(t)
	w := r.do(t, http.MethodPost, "/api/roles/nobody/compile", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
