package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"picomaju/internal/settings"
	"picomaju/internal/staff"
	"picomaju/internal/task"
	"picomaju/internal/tool"
	"picomaju/internal/value"
	uitemplates "picomaju/ui/templates"
)

type uiHandler struct {
	mu       sync.RWMutex
	values   *value.Store
	tasks    *task.Store
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
	h.tasks = task.NewStore(filepath.Join(dataDir, "tasks.json"))
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
	cfg, _ := h.settings.Load()
	tz := "Asia/Jakarta"
	hours := ""
	if cfg != nil {
		if cfg.Timezone != "" {
			tz = cfg.Timezone
		}
		hours = cfg.Hours
	}
	uitemplates.SetupStep1Page("", suggested, tz, hours, "").Render(r.Context(), w)
}

func (h *uiHandler) completeSetup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		uitemplates.SetupStep1Page("", "", "Asia/Jakarta", "", err.Error()).Render(r.Context(), w)
		return
	}
	businessName := strings.TrimSpace(r.FormValue("business_name"))
	dataDir := strings.TrimSpace(r.FormValue("data_dir"))
	tz := strings.TrimSpace(r.FormValue("timezone"))
	hours := strings.TrimSpace(r.FormValue("hours"))
	if dataDir == "" {
		uitemplates.SetupStep1Page(businessName, dataDir, tz, hours, "Data directory is required.").Render(r.Context(), w)
		return
	}
	// Load existing settings to preserve Languages set on welcome screen.
	cfg, _ := h.settings.Load()
	if cfg == nil {
		cfg = &settings.Settings{}
	}
	cfg.BusinessName = businessName
	cfg.DataDir = dataDir
	cfg.Timezone = tz
	cfg.Hours = hours
	if err := h.settings.Save(cfg); err != nil {
		uitemplates.SetupStep1Page(businessName, dataDir, tz, hours, "Could not save settings: "+err.Error()).Render(r.Context(), w)
		return
	}
	if err := h.initStores(dataDir); err != nil {
		uitemplates.SetupStep1Page(businessName, dataDir, tz, hours, "Could not initialise data directory: "+err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/setup/first-staff", http.StatusSeeOther)
}

func (h *uiHandler) integrationsPage(w http.ResponseWriter, r *http.Request) {
	uitemplates.SetupStep3Page(tool.CatalogByCategory(), "").Render(r.Context(), w)
}

func (h *uiHandler) completeIntegrations(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		uitemplates.SetupStep3Page(tool.CatalogByCategory(), err.Error()).Render(r.Context(), w)
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
	activeCat := r.URL.Query().Get("cat")
	vals, _ := h.values.List()
	if vals == nil {
		vals = []*value.Value{}
	}
	if activeCat != "" {
		var filtered []*value.Value
		for _, v := range vals {
			if v.Category == activeCat {
				filtered = append(filtered, v)
			}
		}
		if filtered == nil {
			filtered = []*value.Value{}
		}
		vals = filtered
	}
	uitemplates.ValueListPage(vals, value.DefaultCategories, h.navData(r, "values"), activeCat).Render(r.Context(), w)
}

func (h *uiHandler) newValueForm(w http.ResponseWriter, r *http.Request) {
	uitemplates.ValueFormPage(&value.Value{Version: 1, Priority: 50}, value.DefaultCategories, h.navData(r, "values"), true, "").Render(r.Context(), w)
}

func (h *uiHandler) editValueForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	v, err := h.values.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	uitemplates.ValueFormPage(v, value.DefaultCategories, h.navData(r, "values"), false, "").Render(r.Context(), w)
}

func (h *uiHandler) createValue(w http.ResponseWriter, r *http.Request) {
	v, err := valueFromForm(r)
	if err != nil {
		uitemplates.ValueFormPage(v, value.DefaultCategories, h.navData(r, "values"), true, err.Error()).Render(r.Context(), w)
		return
	}
	if err := h.values.Create(v); err != nil {
		uitemplates.ValueFormPage(v, value.DefaultCategories, h.navData(r, "values"), true, err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/values", http.StatusSeeOther)
}

func (h *uiHandler) updateValue(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	v, err := valueFromForm(r)
	if err != nil {
		uitemplates.ValueFormPage(v, value.DefaultCategories, h.navData(r, "values"), false, err.Error()).Render(r.Context(), w)
		return
	}
	v.ID = id
	if err := h.values.Update(v); err != nil {
		uitemplates.ValueFormPage(v, value.DefaultCategories, h.navData(r, "values"), false, err.Error()).Render(r.Context(), w)
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
	SSEMergeFragment(r.Context(), w, uitemplates.ValidationFragment(res))
}

// --- Tools UI ---

func (h *uiHandler) toolList(w http.ResponseWriter, r *http.Request) {
	tools, _ := h.tools.List()
	if tools == nil {
		tools = []tool.Tool{}
	}
	uitemplates.ToolListPage(tools, h.navData(r, "tools")).Render(r.Context(), w)
}

func (h *uiHandler) newToolForm(w http.ResponseWriter, r *http.Request) {
	uitemplates.NewToolPage(tool.CatalogByCategory(), h.navData(r, "tools"), "").Render(r.Context(), w)
}

func (h *uiHandler) editToolForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	t, err := h.tools.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	uitemplates.ToolFormPage(t, lookupIntegration(t.Type), h.navData(r, "tools"), "").Render(r.Context(), w)
}

func (h *uiHandler) createTool(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		uitemplates.NewToolPage(tool.CatalogByCategory(), h.navData(r, "tools"), err.Error()).Render(r.Context(), w)
		return
	}
	integID := strings.TrimSpace(r.FormValue("integration_id"))
	catalogIndex := tool.CatalogByID()
	integ, ok := catalogIndex[integID]
	if !ok {
		uitemplates.NewToolPage(tool.CatalogByCategory(), h.navData(r, "tools"), "Please select a tool.").Render(r.Context(), w)
		return
	}
	t := &tool.Tool{
		ID:    integ.ID,
		Label: integ.Label,
		Type:  integ.Type,
	}
	if err := h.tools.Create(t); err != nil {
		uitemplates.NewToolPage(tool.CatalogByCategory(), h.navData(r, "tools"), err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/tools/"+t.ID+"/edit", http.StatusSeeOther)
}

func (h *uiHandler) updateTool(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	existing, err := h.tools.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		uitemplates.ToolFormPage(existing, lookupIntegration(existing.Type), h.navData(r, "tools"), err.Error()).Render(r.Context(), w)
		return
	}

	t := &tool.Tool{
		ID:    id,
		Label: strings.TrimSpace(r.FormValue("label")),
		Type:  existing.Type,
	}

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

	if err := h.tools.Update(t); err != nil {
		uitemplates.ToolFormPage(t, integ, h.navData(r, "tools"), err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/tools", http.StatusSeeOther)
}

func (h *uiHandler) deleteTool(w http.ResponseWriter, r *http.Request) {
	h.tools.Delete(chi.URLParam(r, "id"))
	http.Redirect(w, r, "/tools", http.StatusSeeOther)
}

// --- Tasks UI ---

func (h *uiHandler) taskList(w http.ResponseWriter, r *http.Request) {
	tasks, _ := h.tasks.List()
	if tasks == nil {
		tasks = []task.Task{}
	}
	uitemplates.TaskListPage(tasks, h.navData(r, "tasks")).Render(r.Context(), w)
}

func (h *uiHandler) newTaskForm(w http.ResponseWriter, r *http.Request) {
	tools, _ := h.tools.List()
	uitemplates.TaskFormPage(&task.Task{}, tools, h.navData(r, "tasks"), true, "").Render(r.Context(), w)
}

func (h *uiHandler) editTaskForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tk, err := h.tasks.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	tools, _ := h.tools.List()
	uitemplates.TaskFormPage(tk, tools, h.navData(r, "tasks"), false, "").Render(r.Context(), w)
}

func (h *uiHandler) createTask(w http.ResponseWriter, r *http.Request) {
	tools, _ := h.tools.List()
	tk, err := taskFromForm(r)
	if err != nil {
		uitemplates.TaskFormPage(tk, tools, h.navData(r, "tasks"), true, err.Error()).Render(r.Context(), w)
		return
	}
	if err := h.tasks.Create(tk); err != nil {
		uitemplates.TaskFormPage(tk, tools, h.navData(r, "tasks"), true, err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/tasks", http.StatusSeeOther)
}

func (h *uiHandler) updateTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tools, _ := h.tools.List()
	tk, err := taskFromForm(r)
	if err != nil {
		uitemplates.TaskFormPage(tk, tools, h.navData(r, "tasks"), false, err.Error()).Render(r.Context(), w)
		return
	}
	tk.ID = id
	if err := h.tasks.Update(tk); err != nil {
		uitemplates.TaskFormPage(tk, tools, h.navData(r, "tasks"), false, err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/tasks", http.StatusSeeOther)
}

func (h *uiHandler) deleteTask(w http.ResponseWriter, r *http.Request) {
	h.tasks.Delete(chi.URLParam(r, "id"))
	http.Redirect(w, r, "/tasks", http.StatusSeeOther)
}

// --- Staff UI ---

func (h *uiHandler) staffList(w http.ResponseWriter, r *http.Request) {
	members, _ := h.staff.List()
	if members == nil {
		members = []staff.Staff{}
	}
	uitemplates.StaffListPage(members, h.navData(r, "staff")).Render(r.Context(), w)
}

func (h *uiHandler) staffDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m, err := h.staff.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	section := r.URL.Query().Get("s")
	if section == "" {
		section = "overview"
	}
	formErr := r.URL.Query().Get("err")
	tasks, _ := h.tasks.List()
	vals, _ := h.values.List()
	tools, _ := h.tools.List()
	uitemplates.StaffDetailPage(m, tasks, tools, vals, value.DefaultCategories, h.navData(r, "staff"), section, formErr).Render(r.Context(), w)
}

func (h *uiHandler) newStaffForm(w http.ResponseWriter, r *http.Request) {
	uitemplates.StaffFormPage(&staff.Staff{}, h.navData(r, "staff"), "").Render(r.Context(), w)
}

func (h *uiHandler) createStaff(w http.ResponseWriter, r *http.Request) {
	m, err := staffFromForm(r)
	if err != nil {
		uitemplates.StaffFormPage(m, h.navData(r, "staff"), err.Error()).Render(r.Context(), w)
		return
	}
	if err := h.staff.Create(m); err != nil {
		uitemplates.StaffFormPage(m, h.navData(r, "staff"), err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/staff/"+m.ID, http.StatusSeeOther)
}

func (h *uiHandler) updateStaffProfile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m, err := h.staff.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/staff/"+id+"?s=profile&err="+err.Error(), http.StatusSeeOther)
		return
	}
	label := strings.TrimSpace(r.FormValue("label"))
	if label == "" {
		http.Redirect(w, r, "/staff/"+id+"?s=profile&err=Label+is+required", http.StatusSeeOther)
		return
	}
	m.Label = label
	m.Description = strings.TrimSpace(r.FormValue("description"))
	m.Icon = r.FormValue("icon")
	m.Active = r.FormValue("active") == "on"
	if err := h.staff.Update(m); err != nil {
		http.Redirect(w, r, "/staff/"+id+"?s=profile&err="+err.Error(), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/staff/"+id+"?s=profile", http.StatusSeeOther)
}

func (h *uiHandler) updateStaffTasks(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m, err := h.staff.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/staff/"+id+"?s=tasks", http.StatusSeeOther)
		return
	}
	tasks := r.Form["tasks"]
	if tasks == nil {
		tasks = []string{}
	}
	m.Tasks = tasks
	h.staff.Update(m)
	http.Redirect(w, r, "/staff/"+id+"?s=tasks", http.StatusSeeOther)
}

func (h *uiHandler) updateStaffValues(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m, err := h.staff.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/staff/"+id+"?s=values", http.StatusSeeOther)
		return
	}
	valueCats := r.Form["value_categories"]
	if valueCats == nil {
		valueCats = []string{}
	}
	values := r.Form["values"]
	if values == nil {
		values = []string{}
	}
	m.ValueCategories = valueCats
	m.Values = values
	h.staff.Update(m)
	http.Redirect(w, r, "/staff/"+id+"?s=values", http.StatusSeeOther)
}

func (h *uiHandler) deleteStaff(w http.ResponseWriter, r *http.Request) {
	h.staff.Delete(chi.URLParam(r, "id"))
	http.Redirect(w, r, "/", http.StatusSeeOther)
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
	uitemplates.SettingsPage(cfg, activeDir, h.navData(r, ""), saved, "").Render(r.Context(), w)
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
		uitemplates.SettingsPage(cfg, activeDir, h.navData(r, ""), false, err.Error()).Render(r.Context(), w)
		return
	}
	newDataDir := strings.TrimSpace(r.FormValue("data_dir"))
	cfg := &settings.Settings{
		BusinessName: strings.TrimSpace(r.FormValue("business_name")),
		DataDir:      newDataDir,
	}
	if newDataDir != "" && newDataDir != activeDir {
		if err := h.initStores(newDataDir); err != nil {
			uitemplates.SettingsPage(cfg, activeDir, h.navData(r, ""), false, "Cannot use that directory: "+err.Error()).Render(r.Context(), w)
			return
		}
	}
	if err := h.settings.Save(cfg); err != nil {
		h.mu.RLock()
		activeDir = h.dataDir
		h.mu.RUnlock()
		uitemplates.SettingsPage(cfg, activeDir, h.navData(r, ""), false, err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/settings?saved=1", http.StatusSeeOther)
}

// --- Helpers ---

func (h *uiHandler) navData(r *http.Request, section string) uitemplates.NavData {
	cfg, _ := h.settings.Load()
	name := ""
	if cfg != nil {
		name = cfg.BusinessName
	}
	return uitemplates.NavData{
		BusinessName:  name,
		ActiveSection: section,
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

func taskFromForm(r *http.Request) (*task.Task, error) {
	if err := r.ParseForm(); err != nil {
		return &task.Task{}, err
	}
	tools := r.Form["tools"]
	if tools == nil {
		tools = []string{}
	}
	return &task.Task{
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
	return &staff.Staff{
		ID:          strings.TrimSpace(r.FormValue("id")),
		Label:       strings.TrimSpace(r.FormValue("label")),
		Description: strings.TrimSpace(r.FormValue("description")),
		Active:      r.FormValue("active") == "on",
		Icon:        r.FormValue("icon"),
	}, nil
}
