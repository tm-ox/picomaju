package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"picomaju/internal/category"
	"picomaju/internal/role"
	"picomaju/internal/settings"
	"picomaju/internal/sop"
	"picomaju/web/templates"
)

type uiHandler struct {
	mu       sync.RWMutex
	sops     *sop.Store
	roles    *role.Store
	cats     *category.Store
	settings *settings.Store
	dataDir  string
}

func (h *uiHandler) configured() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.sops != nil
}

func (h *uiHandler) initStores(dataDir string) error {
	sopDir := filepath.Join(dataDir, "sops")
	if err := os.MkdirAll(sopDir, 0755); err != nil {
		return err
	}
	newSops := sop.NewStore(sopDir)
	newRoles := role.NewStore(filepath.Join(dataDir, "roles.json"))
	newCats := category.NewStore(filepath.Join(dataDir, "categories.json"))
	if err := newCats.Seed(); err != nil {
		return err
	}
	h.mu.Lock()
	h.sops, h.roles, h.cats, h.dataDir = newSops, newRoles, newCats, dataDir
	h.mu.Unlock()
	return nil
}

// --- Setup (onboarding) ---

func (h *uiHandler) setupPage(w http.ResponseWriter, r *http.Request) {
	suggested := ""
	if home, err := os.UserHomeDir(); err == nil {
		suggested = filepath.Join(home, "picomaju")
	}
	templates.SetupPage("", suggested, "").Render(r.Context(), w)
}

func (h *uiHandler) completeSetup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		templates.SetupPage("", "", err.Error()).Render(r.Context(), w)
		return
	}
	businessName := strings.TrimSpace(r.FormValue("business_name"))
	dataDir := strings.TrimSpace(r.FormValue("data_dir"))
	if dataDir == "" {
		templates.SetupPage(businessName, dataDir, "Data directory is required.").Render(r.Context(), w)
		return
	}
	cfg := &settings.Settings{BusinessName: businessName, DataDir: dataDir}
	if err := h.settings.Save(cfg); err != nil {
		templates.SetupPage(businessName, dataDir, "Could not save settings: "+err.Error()).Render(r.Context(), w)
		return
	}
	if err := h.initStores(dataDir); err != nil {
		templates.SetupPage(businessName, dataDir, "Could not initialise data directory: "+err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// --- SOP UI ---

func (h *uiHandler) sopList(w http.ResponseWriter, r *http.Request) {
	sb := h.sidebarData(r)
	sops, _ := h.sops.List()
	if sops == nil {
		sops = []*sop.SOP{}
	}
	if sb.ActiveCat != "" {
		var filtered []*sop.SOP
		for _, s := range sops {
			if s.Category == sb.ActiveCat {
				filtered = append(filtered, s)
			}
		}
		if filtered == nil {
			filtered = []*sop.SOP{}
		}
		sops = filtered
	}
	templates.SOPListPage(sops, sb).Render(r.Context(), w)
}

func (h *uiHandler) newSOPForm(w http.ResponseWriter, r *http.Request) {
	sb := h.sidebarData(r)
	templates.SOPFormPage(&sop.SOP{Version: 1, Priority: 50}, sb, true, "").Render(r.Context(), w)
}

func (h *uiHandler) editSOPForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	s, err := h.sops.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	sb := h.sidebarData(r)
	templates.SOPFormPage(s, sb, false, "").Render(r.Context(), w)
}

func (h *uiHandler) createSOP(w http.ResponseWriter, r *http.Request) {
	sb := h.sidebarData(r)
	s, err := sopFromForm(r)
	if err != nil {
		templates.SOPFormPage(s, sb, true, err.Error()).Render(r.Context(), w)
		return
	}
	if err := h.sops.Create(s); err != nil {
		templates.SOPFormPage(s, sb, true, err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *uiHandler) updateSOP(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sb := h.sidebarData(r)
	s, err := sopFromForm(r)
	if err != nil {
		templates.SOPFormPage(s, sb, false, err.Error()).Render(r.Context(), w)
		return
	}
	s.ID = id
	if err := h.sops.Update(s); err != nil {
		templates.SOPFormPage(s, sb, false, err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *uiHandler) deleteSOP(w http.ResponseWriter, r *http.Request) {
	h.sops.Delete(chi.URLParam(r, "id"))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *uiHandler) validateSSE(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	s, err := h.sops.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	res := sop.Validate(s)
	SSEMergeFragment(r.Context(), w, templates.ValidationFragment(res))
}

// --- Role UI ---

func (h *uiHandler) roleList(w http.ResponseWriter, r *http.Request) {
	sb := h.sidebarData(r)
	roles, _ := h.roles.List()
	if roles == nil {
		roles = []role.Role{}
	}
	templates.RoleListPage(roles, sb).Render(r.Context(), w)
}

func (h *uiHandler) newRoleForm(w http.ResponseWriter, r *http.Request) {
	sb := h.sidebarData(r)
	sopsByCat, _ := h.sopsByCat()
	templates.RoleFormPage(&role.Role{}, sb, sopsByCat, true, "").Render(r.Context(), w)
}

func (h *uiHandler) editRoleForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ro, err := h.roles.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	sb := h.sidebarData(r)
	sopsByCat, _ := h.sopsByCat()
	templates.RoleFormPage(ro, sb, sopsByCat, false, "").Render(r.Context(), w)
}

func (h *uiHandler) createRole(w http.ResponseWriter, r *http.Request) {
	sb := h.sidebarData(r)
	sopsByCat, _ := h.sopsByCat()
	ro, err := roleFromForm(r)
	if err != nil {
		templates.RoleFormPage(ro, sb, sopsByCat, true, err.Error()).Render(r.Context(), w)
		return
	}
	if err := h.roles.Create(ro); err != nil {
		templates.RoleFormPage(ro, sb, sopsByCat, true, err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/roles", http.StatusSeeOther)
}

func (h *uiHandler) updateRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sb := h.sidebarData(r)
	sopsByCat, _ := h.sopsByCat()
	ro, err := roleFromForm(r)
	if err != nil {
		templates.RoleFormPage(ro, sb, sopsByCat, false, err.Error()).Render(r.Context(), w)
		return
	}
	ro.ID = id
	if err := h.roles.Update(ro); err != nil {
		templates.RoleFormPage(ro, sb, sopsByCat, false, err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/roles", http.StatusSeeOther)
}

func (h *uiHandler) deleteRole(w http.ResponseWriter, r *http.Request) {
	h.roles.Delete(chi.URLParam(r, "id"))
	http.Redirect(w, r, "/roles", http.StatusSeeOther)
}

func (h *uiHandler) compileSSE(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ro, err := h.roles.Get(id)
	if err != nil {
		SSEMergeFragment(r.Context(), w, templates.CompileErrorFragment(id, err.Error()))
		return
	}
	allSOPs, _ := h.sops.List()
	result, err := sop.Compile(ro.ID, ro.Categories, ro.SOPs, allSOPs)
	if err != nil {
		SSEMergeFragment(r.Context(), w, templates.CompileErrorFragment(id, err.Error()))
		return
	}
	SSEMergeFragment(r.Context(), w, templates.CompileFragment(id, result))
}

// --- Settings UI ---

func (h *uiHandler) settingsPage(w http.ResponseWriter, r *http.Request) {
	cfg, _ := h.settings.Load()
	if cfg == nil {
		cfg = &settings.Settings{}
	}
	saved := r.URL.Query().Get("saved") == "1"
	h.mu.RLock()
	activeDir := h.dataDir
	h.mu.RUnlock()
	templates.SettingsPage(cfg, activeDir, h.sidebarData(r), saved, "").Render(r.Context(), w)
}

func (h *uiHandler) saveSettings(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	activeDir := h.dataDir
	h.mu.RUnlock()

	if err := r.ParseForm(); err != nil {
		cfg, _ := h.settings.Load()
		if cfg == nil {
			cfg = &settings.Settings{}
		}
		templates.SettingsPage(cfg, activeDir, h.sidebarData(r), false, err.Error()).Render(r.Context(), w)
		return
	}
	newDataDir := strings.TrimSpace(r.FormValue("data_dir"))
	cfg := &settings.Settings{
		BusinessName:    strings.TrimSpace(r.FormValue("business_name")),
		BusinessDetails: strings.TrimSpace(r.FormValue("business_details")),
		DataDir:         newDataDir,
	}
	// Switch data dir live if it changed.
	if newDataDir != "" && newDataDir != activeDir {
		if err := h.initStores(newDataDir); err != nil {
			templates.SettingsPage(cfg, activeDir, h.sidebarData(r), false, "Cannot use that directory: "+err.Error()).Render(r.Context(), w)
			return
		}
	}
	if err := h.settings.Save(cfg); err != nil {
		h.mu.RLock()
		activeDir = h.dataDir
		h.mu.RUnlock()
		templates.SettingsPage(cfg, activeDir, h.sidebarData(r), false, err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/settings?saved=1", http.StatusSeeOther)
}

// --- Helpers ---

func (h *uiHandler) sidebarData(r *http.Request) templates.SidebarData {
	h.mu.RLock()
	sopsStore, rolesStore, catsStore := h.sops, h.roles, h.cats
	h.mu.RUnlock()

	cats, _ := catsStore.List()
	roles, _ := rolesStore.List()
	sops, _ := sopsStore.List()
	counts := make(map[string]int)
	for _, s := range sops {
		counts[s.Category]++
	}
	cfg, _ := h.settings.Load()
	name := ""
	if cfg != nil {
		name = cfg.BusinessName
	}
	return templates.SidebarData{
		Categories:   cats,
		Roles:        roles,
		SOPCounts:    counts,
		ActiveCat:    r.URL.Query().Get("cat"),
		BusinessName: name,
	}
}

func (h *uiHandler) sopsByCat() (map[string][]*sop.SOP, error) {
	all, err := h.sops.List()
	if err != nil {
		return nil, err
	}
	m := make(map[string][]*sop.SOP)
	for _, s := range all {
		m[s.Category] = append(m[s.Category], s)
	}
	return m, nil
}

func sopFromForm(r *http.Request) (*sop.SOP, error) {
	if err := r.ParseForm(); err != nil {
		return &sop.SOP{}, err
	}
	version, _ := strconv.Atoi(r.FormValue("version"))
	priority, _ := strconv.Atoi(r.FormValue("priority"))
	return &sop.SOP{
		ID:       strings.TrimSpace(r.FormValue("id")),
		Title:    strings.TrimSpace(r.FormValue("title")),
		Category: r.FormValue("category"),
		Version:  version,
		Priority: priority,
		Trigger:  strings.TrimSpace(r.FormValue("trigger")),
		Body:     strings.TrimSpace(r.FormValue("body")),
	}, nil
}

func roleFromForm(r *http.Request) (*role.Role, error) {
	if err := r.ParseForm(); err != nil {
		return &role.Role{}, err
	}
	cats := r.Form["categories"]
	if cats == nil {
		cats = []string{}
	}
	individualSOPs := r.Form["sops"]
	if individualSOPs == nil {
		individualSOPs = []string{}
	}
	return &role.Role{
		ID:         strings.TrimSpace(r.FormValue("id")),
		Label:      strings.TrimSpace(r.FormValue("label")),
		Categories: cats,
		SOPs:       individualSOPs,
	}, nil
}
