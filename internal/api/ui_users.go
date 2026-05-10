package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"picomaju/internal/session"
	"picomaju/internal/user"
	logintpl "picomaju/ui/templates/login"
	userstpl "picomaju/ui/templates/users"
)

// ── Login / logout ────────────────────────────────────────────────────────────

func (h *uiHandler) loginPage(w http.ResponseWriter, r *http.Request) {
	users, _ := h.users.List()
	entries := make([]logintpl.UserEntry, len(users))
	for i, u := range users {
		entries[i] = logintpl.UserEntry{ID: u.ID, Name: u.Name, Role: string(u.Role)}
	}
	selectedID := r.URL.Query().Get("u")
	formErr := r.URL.Query().Get("err")
	logintpl.LoginPage(entries, selectedID, formErr).Render(r.Context(), w)
}

func (h *uiHandler) loginSubmit(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.FormValue("user_id"))
	pin := r.FormValue("pin")

	u, err := h.users.Get(userID)
	if err != nil || !u.CheckPIN(pin) {
		http.Redirect(w, r, "/login?u="+userID+"&err=Incorrect+PIN", http.StatusSeeOther)
		return
	}

	sid, err := h.sessions.Create(u.ID)
	if err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}
	h.sessions.SetCookie(w, sid)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *uiHandler) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(session.CookieName); err == nil {
		h.sessions.Delete(c.Value)
	}
	h.sessions.ClearCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// ── User management ───────────────────────────────────────────────────────────

func (h *uiHandler) userList(w http.ResponseWriter, r *http.Request) {
	if !h.isOwner(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	users, _ := h.users.List()
	rows := make([]userstpl.UserRow, len(users))
	for i, u := range users {
		rows[i] = userstpl.UserRow{
			ID:          u.ID,
			Name:        u.Name,
			Role:        string(u.Role),
			StaffID:     u.StaffID,
			Description: u.Description,
			HasPIN:      u.PINHash != "",
		}
	}
	userstpl.UserListPage(rows, h.navData(r, "users")).Render(r.Context(), w)
}

func (h *uiHandler) newUserForm(w http.ResponseWriter, r *http.Request) {
	if !h.isOwner(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	staffOpts, _ := h.staffOptions()
	userstpl.UserFormPage(userstpl.UserRow{Role: "manager"}, staffOpts, h.navData(r, "users"), true, "").Render(r.Context(), w)
}

func (h *uiHandler) createUser(w http.ResponseWriter, r *http.Request) {
	if !h.isOwner(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	role := user.Role(r.FormValue("role"))
	pin := r.FormValue("pin")
	description := strings.TrimSpace(r.FormValue("description"))
	staffID := r.FormValue("staff_id")

	if name == "" || pin == "" {
		h.userFormErr(w, r, userstpl.UserRow{Name: name, Role: string(role), Description: description, StaffID: staffID}, true, "Name and PIN are required")
		return
	}
	if len(pin) < 4 {
		h.userFormErr(w, r, userstpl.UserRow{Name: name, Role: string(role), Description: description, StaffID: staffID}, true, "PIN must be at least 4 digits")
		return
	}

	u := &user.User{
		ID:          fmt.Sprintf("%x", time.Now().UnixNano()),
		Name:        name,
		Role:        role,
		Description: description,
		StaffID:     staffID,
	}
	if err := u.SetPIN(pin); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.users.Create(u); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func (h *uiHandler) editUserForm(w http.ResponseWriter, r *http.Request) {
	if !h.isOwner(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	id := chi.URLParam(r, "id")
	u, err := h.users.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	staffOpts, _ := h.staffOptions()
	row := userstpl.UserRow{
		ID:          u.ID,
		Name:        u.Name,
		Role:        string(u.Role),
		StaffID:     u.StaffID,
		Description: u.Description,
		HasPIN:      u.PINHash != "",
	}
	userstpl.UserFormPage(row, staffOpts, h.navData(r, "users"), false, r.URL.Query().Get("err")).Render(r.Context(), w)
}

func (h *uiHandler) updateUser(w http.ResponseWriter, r *http.Request) {
	if !h.isOwner(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	id := chi.URLParam(r, "id")
	u, err := h.users.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Redirect(w, r, "/users/"+id+"/edit?err=Name+is+required", http.StatusSeeOther)
		return
	}
	pin := r.FormValue("pin")
	if pin != "" && len(pin) < 4 {
		http.Redirect(w, r, "/users/"+id+"/edit?err=PIN+must+be+at+least+4+digits", http.StatusSeeOther)
		return
	}

	u.Name = name
	u.Role = user.Role(r.FormValue("role"))
	u.Description = strings.TrimSpace(r.FormValue("description"))
	u.StaffID = r.FormValue("staff_id")
	if pin != "" {
		if err := u.SetPIN(pin); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err := h.users.Update(u); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func (h *uiHandler) deleteUser(w http.ResponseWriter, r *http.Request) {
	if !h.isOwner(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	id := chi.URLParam(r, "id")
	// Prevent deleting yourself.
	if uid, ok := session.CurrentUser(r); ok && uid == id {
		http.Redirect(w, r, "/users?err=Cannot+delete+your+own+account", http.StatusSeeOther)
		return
	}
	h.users.Delete(id)
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

// ── Profile (self-edit) ───────────────────────────────────────────────────────

func (h *uiHandler) profilePage(w http.ResponseWriter, r *http.Request) {
	u := h.currentUser(r)
	if u == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	row := userstpl.UserRow{
		ID:          u.ID,
		Name:        u.Name,
		Role:        string(u.Role),
		Description: u.Description,
		HasPIN:      u.PINHash != "",
	}
	userstpl.ProfilePage(row, h.navData(r, ""), r.URL.Query().Get("err")).Render(r.Context(), w)
}

func (h *uiHandler) updateProfile(w http.ResponseWriter, r *http.Request) {
	u := h.currentUser(r)
	if u == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Redirect(w, r, "/profile?err=Name+is+required", http.StatusSeeOther)
		return
	}
	pin := r.FormValue("pin")
	if pin != "" && len(pin) < 4 {
		http.Redirect(w, r, "/profile?err=PIN+must+be+at+least+4+digits", http.StatusSeeOther)
		return
	}
	u.Name = name
	u.Description = strings.TrimSpace(r.FormValue("description"))
	if pin != "" {
		if err := u.SetPIN(pin); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err := h.users.Update(u); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (h *uiHandler) isOwner(r *http.Request) bool {
	u := h.currentUser(r)
	return u != nil && u.Role == user.RoleOwner
}

func (h *uiHandler) staffOptions() ([]userstpl.StaffOption, error) {
	members, err := h.staff.List()
	if err != nil {
		return nil, err
	}
	opts := make([]userstpl.StaffOption, len(members))
	for i, m := range members {
		opts[i] = userstpl.StaffOption{ID: m.ID, Label: m.Label}
	}
	return opts, nil
}

func (h *uiHandler) userFormErr(w http.ResponseWriter, r *http.Request, row userstpl.UserRow, isNew bool, msg string) {
	staffOpts, _ := h.staffOptions()
	userstpl.UserFormPage(row, staffOpts, h.navData(r, "users"), isNew, msg).Render(r.Context(), w)
}
