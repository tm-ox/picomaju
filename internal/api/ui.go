package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"picomaju/internal/role"
	"picomaju/internal/settings"
	"picomaju/internal/staff"
	"picomaju/internal/tool"
	"picomaju/internal/value"
	"picomaju/web/templates"
)

type uiHandler struct {
	mu       sync.RWMutex
	values   *value.Store
	roles    *role.Store
	tools    *tool.Store
	staff    *staff.Store
	settings *settings.Store
	dataDir  string
}

func (h *uiHandler) configured() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.values != nil
}

func (h *uiHandler) initStores(dataDir string) error {
	valueDir := filepath.Join(dataDir, "values")
	if err := os.MkdirAll(valueDir, 0755); err != nil {
		return err
	}
	h.mu.Lock()
	h.values = value.NewStore(valueDir)
	h.roles = role.NewStore(filepath.Join(dataDir, "roles.json"))
	h.tools = tool.NewStore(filepath.Join(dataDir, "tools.json"))
	h.staff = staff.NewStore(filepath.Join(dataDir, "staff.json"))
	h.dataDir = dataDir
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
	http.Redirect(w, r, "/setup/integrations", http.StatusSeeOther)
}

func (h *uiHandler) integrationsPage(w http.ResponseWriter, r *http.Request) {
	templates.IntegrationsPage(tool.CatalogByCategory(), "").Render(r.Context(), w)
}

func (h *uiHandler) completeIntegrations(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		templates.IntegrationsPage(tool.CatalogByCategory(), err.Error()).Render(r.Context(), w)
		return
	}

	selected := r.Form["integrations"]
	catalogIndex := tool.CatalogByID()

	for _, id := range selected {
		integ, ok := catalogIndex[id]
		if !ok {
			continue
		}
		t := &tool.Tool{
			ID:    integ.ID,
			Label: integ.Label,
			Type:  integ.Type,
		}
		// Ignore "already exists" — onboarding may be revisited.
		h.tools.Create(t) //nolint:errcheck
	}

	http.Redirect(w, r, "/values", http.StatusSeeOther)
}

// --- Values UI ---

func (h *uiHandler) valueList(w http.ResponseWriter, r *http.Request) {
	sb := h.sidebarData(r, "values")
	vals, _ := h.values.List()
	if vals == nil {
		vals = []*value.Value{}
	}
	if sb.ActiveCat != "" {
		var filtered []*value.Value
		for _, v := range vals {
			if v.Category == sb.ActiveCat {
				filtered = append(filtered, v)
			}
		}
		if filtered == nil {
			filtered = []*value.Value{}
		}
		vals = filtered
	}
	templates.ValueListPage(vals, sb).Render(r.Context(), w)
}

func (h *uiHandler) newValueForm(w http.ResponseWriter, r *http.Request) {
	sb := h.sidebarData(r, "values")
	templates.ValueFormPage(&value.Value{Version: 1, Priority: 50}, sb, true, "").Render(r.Context(), w)
}

func (h *uiHandler) editValueForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	v, err := h.values.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	sb := h.sidebarData(r, "values")
	templates.ValueFormPage(v, sb, false, "").Render(r.Context(), w)
}

func (h *uiHandler) createValue(w http.ResponseWriter, r *http.Request) {
	sb := h.sidebarData(r, "values")
	v, err := valueFromForm(r)
	if err != nil {
		templates.ValueFormPage(v, sb, true, err.Error()).Render(r.Context(), w)
		return
	}
	if err := h.values.Create(v); err != nil {
		templates.ValueFormPage(v, sb, true, err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/values", http.StatusSeeOther)
}

func (h *uiHandler) updateValue(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sb := h.sidebarData(r, "values")
	v, err := valueFromForm(r)
	if err != nil {
		templates.ValueFormPage(v, sb, false, err.Error()).Render(r.Context(), w)
		return
	}
	v.ID = id
	if err := h.values.Update(v); err != nil {
		templates.ValueFormPage(v, sb, false, err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/values", http.StatusSeeOther)
}

func (h *uiHandler) deleteValue(w http.ResponseWriter, r *http.Request) {
	h.values.Delete(chi.URLParam(r, "id"))
	http.Redirect(w, r, "/values", http.StatusSeeOther)
}

func (h *uiHandler) validateSSE(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	v, err := h.values.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	res := value.Validate(v)
	SSEMergeFragment(r.Context(), w, templates.ValidationFragment(res))
}

// --- Tools UI ---

func (h *uiHandler) toolList(w http.ResponseWriter, r *http.Request) {
	sb := h.sidebarData(r, "tools")
	tools, _ := h.tools.List()
	if tools == nil {
		tools = []tool.Tool{}
	}
	templates.ToolListPage(tools, sb).Render(r.Context(), w)
}

func (h *uiHandler) newToolForm(w http.ResponseWriter, r *http.Request) {
	sb := h.sidebarData(r, "tools")
	templates.NewIntegrationPage(tool.CatalogByCategory(), sb, "").Render(r.Context(), w)
}

func (h *uiHandler) newSkillForm(w http.ResponseWriter, r *http.Request) {
	sb := h.sidebarData(r, "tools")
	templates.SkillFormPage(&tool.Tool{}, sb, true, "").Render(r.Context(), w)
}

func (h *uiHandler) editToolForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	t, err := h.tools.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	sb := h.sidebarData(r, "tools")
	if t.Type == "skill" {
		templates.SkillFormPage(t, sb, false, "").Render(r.Context(), w)
		return
	}
	templates.ToolFormPage(t, lookupIntegration(t.Type), sb, "").Render(r.Context(), w)
}

func (h *uiHandler) createTool(w http.ResponseWriter, r *http.Request) {
	sb := h.sidebarData(r, "tools")
	if err := r.ParseForm(); err != nil {
		templates.NewIntegrationPage(tool.CatalogByCategory(), sb, err.Error()).Render(r.Context(), w)
		return
	}
	integID := strings.TrimSpace(r.FormValue("integration_id"))
	catalogIndex := tool.CatalogByID()
	integ, ok := catalogIndex[integID]
	if !ok {
		templates.NewIntegrationPage(tool.CatalogByCategory(), sb, "Please select an integration.").Render(r.Context(), w)
		return
	}
	t := &tool.Tool{
		ID:    integ.ID,
		Label: integ.Label,
		Type:  integ.Type,
	}
	if err := h.tools.Create(t); err != nil {
		templates.NewIntegrationPage(tool.CatalogByCategory(), sb, err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/tools/"+t.ID+"/edit", http.StatusSeeOther)
}

func (h *uiHandler) createSkill(w http.ResponseWriter, r *http.Request) {
	sb := h.sidebarData(r, "tools")
	if err := r.ParseForm(); err != nil {
		templates.SkillFormPage(&tool.Tool{}, sb, true, err.Error()).Render(r.Context(), w)
		return
	}
	t := &tool.Tool{
		ID:    strings.TrimSpace(r.FormValue("id")),
		Label: strings.TrimSpace(r.FormValue("label")),
		Type:  "skill",
		Config: map[string]any{
			"content": strings.TrimSpace(r.FormValue("content")),
		},
	}
	if err := h.tools.Create(t); err != nil {
		templates.SkillFormPage(t, sb, true, err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/tools/"+t.ID+"/edit", http.StatusSeeOther)
}

func (h *uiHandler) updateTool(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sb := h.sidebarData(r, "tools")

	existing, err := h.tools.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		if existing.Type == "skill" {
			templates.SkillFormPage(existing, sb, false, err.Error()).Render(r.Context(), w)
		} else {
			templates.ToolFormPage(existing, lookupIntegration(existing.Type), sb, err.Error()).Render(r.Context(), w)
		}
		return
	}

	t := &tool.Tool{
		ID:    id,
		Label: strings.TrimSpace(r.FormValue("label")),
		Type:  existing.Type,
	}

	if existing.Type == "skill" {
		t.Config = map[string]any{
			"content": strings.TrimSpace(r.FormValue("content")),
		}
	} else {
		integ := lookupIntegration(existing.Type)
		if integ != nil && len(integ.Fields) > 0 {
			cfg := make(map[string]any)
			for _, f := range integ.Fields {
				val := strings.TrimSpace(r.FormValue("cfg_" + f.Key))
				if val != "" {
					cfg[f.Key] = val
				}
			}
			if len(cfg) > 0 {
				t.Config = cfg
			}
		}
	}

	if err := h.tools.Update(t); err != nil {
		if existing.Type == "skill" {
			templates.SkillFormPage(t, sb, false, err.Error()).Render(r.Context(), w)
		} else {
			templates.ToolFormPage(t, lookupIntegration(existing.Type), sb, err.Error()).Render(r.Context(), w)
		}
		return
	}
	http.Redirect(w, r, "/tools", http.StatusSeeOther)
}

func (h *uiHandler) deleteTool(w http.ResponseWriter, r *http.Request) {
	h.tools.Delete(chi.URLParam(r, "id"))
	http.Redirect(w, r, "/tools", http.StatusSeeOther)
}

// --- Roles UI ---

func (h *uiHandler) roleList(w http.ResponseWriter, r *http.Request) {
	sb := h.sidebarData(r, "roles")
	roles, _ := h.roles.List()
	if roles == nil {
		roles = []role.Role{}
	}
	templates.RoleListPage(roles, sb).Render(r.Context(), w)
}

func (h *uiHandler) newRoleForm(w http.ResponseWriter, r *http.Request) {
	sb := h.sidebarData(r, "roles")
	tools, _ := h.tools.List()
	templates.RoleFormPage(&role.Role{}, tools, sb, true, "").Render(r.Context(), w)
}

func (h *uiHandler) editRoleForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ro, err := h.roles.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	sb := h.sidebarData(r, "roles")
	tools, _ := h.tools.List()
	templates.RoleFormPage(ro, tools, sb, false, "").Render(r.Context(), w)
}

func (h *uiHandler) createRole(w http.ResponseWriter, r *http.Request) {
	sb := h.sidebarData(r, "roles")
	tools, _ := h.tools.List()
	ro, err := roleFromForm(r)
	if err != nil {
		templates.RoleFormPage(ro, tools, sb, true, err.Error()).Render(r.Context(), w)
		return
	}
	if err := h.roles.Create(ro); err != nil {
		templates.RoleFormPage(ro, tools, sb, true, err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/roles", http.StatusSeeOther)
}

func (h *uiHandler) updateRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sb := h.sidebarData(r, "roles")
	tools, _ := h.tools.List()
	ro, err := roleFromForm(r)
	if err != nil {
		templates.RoleFormPage(ro, tools, sb, false, err.Error()).Render(r.Context(), w)
		return
	}
	ro.ID = id
	if err := h.roles.Update(ro); err != nil {
		templates.RoleFormPage(ro, tools, sb, false, err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/roles", http.StatusSeeOther)
}

func (h *uiHandler) deleteRole(w http.ResponseWriter, r *http.Request) {
	h.roles.Delete(chi.URLParam(r, "id"))
	http.Redirect(w, r, "/roles", http.StatusSeeOther)
}

// --- Staff UI ---

func (h *uiHandler) staffList(w http.ResponseWriter, r *http.Request) {
	sb := h.sidebarData(r, "staff")
	members, _ := h.staff.List()
	if members == nil {
		members = []staff.Staff{}
	}
	templates.StaffListPage(members, sb).Render(r.Context(), w)
}

func (h *uiHandler) newStaffForm(w http.ResponseWriter, r *http.Request) {
	sb := h.sidebarData(r, "staff")
	roles, _ := h.roles.List()
	vals, _ := h.values.List()
	templates.StaffFormPage(&staff.Staff{}, roles, vals, sb, true, "").Render(r.Context(), w)
}

func (h *uiHandler) editStaffForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m, err := h.staff.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	sb := h.sidebarData(r, "staff")
	roles, _ := h.roles.List()
	vals, _ := h.values.List()
	templates.StaffFormPage(m, roles, vals, sb, false, "").Render(r.Context(), w)
}

func (h *uiHandler) createStaff(w http.ResponseWriter, r *http.Request) {
	sb := h.sidebarData(r, "staff")
	roles, _ := h.roles.List()
	vals, _ := h.values.List()
	m, err := staffFromForm(r)
	if err != nil {
		templates.StaffFormPage(m, roles, vals, sb, true, err.Error()).Render(r.Context(), w)
		return
	}
	if err := h.staff.Create(m); err != nil {
		templates.StaffFormPage(m, roles, vals, sb, true, err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/staff", http.StatusSeeOther)
}

func (h *uiHandler) updateStaff(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sb := h.sidebarData(r, "staff")
	roles, _ := h.roles.List()
	vals, _ := h.values.List()
	m, err := staffFromForm(r)
	if err != nil {
		templates.StaffFormPage(m, roles, vals, sb, false, err.Error()).Render(r.Context(), w)
		return
	}
	m.ID = id
	if err := h.staff.Update(m); err != nil {
		templates.StaffFormPage(m, roles, vals, sb, false, err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/staff", http.StatusSeeOther)
}

func (h *uiHandler) deleteStaff(w http.ResponseWriter, r *http.Request) {
	h.staff.Delete(chi.URLParam(r, "id"))
	http.Redirect(w, r, "/staff", http.StatusSeeOther)
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
	templates.SettingsPage(cfg, activeDir, h.sidebarData(r, ""), saved, "").Render(r.Context(), w)
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
		templates.SettingsPage(cfg, activeDir, h.sidebarData(r, ""), false, err.Error()).Render(r.Context(), w)
		return
	}
	newDataDir := strings.TrimSpace(r.FormValue("data_dir"))
	cfg := &settings.Settings{
		BusinessName:    strings.TrimSpace(r.FormValue("business_name")),
		BusinessDetails: strings.TrimSpace(r.FormValue("business_details")),
		DataDir:         newDataDir,
	}
	if newDataDir != "" && newDataDir != activeDir {
		if err := h.initStores(newDataDir); err != nil {
			templates.SettingsPage(cfg, activeDir, h.sidebarData(r, ""), false, "Cannot use that directory: "+err.Error()).Render(r.Context(), w)
			return
		}
	}
	if err := h.settings.Save(cfg); err != nil {
		h.mu.RLock()
		activeDir = h.dataDir
		h.mu.RUnlock()
		templates.SettingsPage(cfg, activeDir, h.sidebarData(r, ""), false, err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/settings?saved=1", http.StatusSeeOther)
}

// --- Helpers ---

func (h *uiHandler) sidebarData(r *http.Request, section string) templates.SidebarData {
	h.mu.RLock()
	valStore, roleStore, toolStore, staffStore := h.values, h.roles, h.tools, h.staff
	h.mu.RUnlock()

	var vals []*value.Value
	counts := make(map[string]int)
	if valStore != nil {
		vals, _ = valStore.List()
		for _, v := range vals {
			counts[v.Category]++
		}
	}

	var roles []role.Role
	if roleStore != nil {
		roles, _ = roleStore.List()
	}

	var tools []tool.Tool
	if toolStore != nil {
		tools, _ = toolStore.List()
	}
	catalog := tool.CatalogByType()
	var integrations, skills []tool.Tool
	for _, t := range tools {
		if _, ok := catalog[t.Type]; ok {
			integrations = append(integrations, t)
		} else {
			skills = append(skills, t)
		}
	}

	var members []staff.Staff
	if staffStore != nil {
		members, _ = staffStore.List()
	}

	cfg, _ := h.settings.Load()
	name := ""
	if cfg != nil {
		name = cfg.BusinessName
	}

	return templates.SidebarData{
		Categories:    value.DefaultCategories,
		Staff:         members,
		Roles:         roles,
		Integrations:  integrations,
		Skills:        skills,
		ValueCounts:   counts,
		ActiveCat:     r.URL.Query().Get("cat"),
		ActiveSection: section,
		BusinessName:  name,
	}
}

func lookupIntegration(typeName string) *tool.Integration {
	catalog := tool.CatalogByType()
	if integ, ok := catalog[typeName]; ok {
		return &integ
	}
	return nil
}

func valueFromForm(r *http.Request) (*value.Value, error) {
	if err := r.ParseForm(); err != nil {
		return &value.Value{}, err
	}
	version, _ := strconv.Atoi(r.FormValue("version"))
	priority, _ := strconv.Atoi(r.FormValue("priority"))
	return &value.Value{
		ID:       strings.TrimSpace(r.FormValue("id")),
		Title:    strings.TrimSpace(r.FormValue("title")),
		Category: r.FormValue("category"),
		Version:  version,
		Priority: priority,
		Body:     strings.TrimSpace(r.FormValue("body")),
	}, nil
}

func roleFromForm(r *http.Request) (*role.Role, error) {
	if err := r.ParseForm(); err != nil {
		return &role.Role{}, err
	}
	tools := r.Form["tools"]
	if tools == nil {
		tools = []string{}
	}
	return &role.Role{
		ID:          strings.TrimSpace(r.FormValue("id")),
		Label:       strings.TrimSpace(r.FormValue("label")),
		Description: strings.TrimSpace(r.FormValue("description")),
		Tools:       tools,
	}, nil
}

func staffFromForm(r *http.Request) (*staff.Staff, error) {
	if err := r.ParseForm(); err != nil {
		return &staff.Staff{}, err
	}
	roles := r.Form["roles"]
	if roles == nil {
		roles = []string{}
	}
	valueCats := r.Form["value_categories"]
	if valueCats == nil {
		valueCats = []string{}
	}
	values := r.Form["values"]
	if values == nil {
		values = []string{}
	}
	return &staff.Staff{
		ID:              strings.TrimSpace(r.FormValue("id")),
		Label:           strings.TrimSpace(r.FormValue("label")),
		Roles:           roles,
		ValueCategories: valueCats,
		Values:          values,
	}, nil
}
