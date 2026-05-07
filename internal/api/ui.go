package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"picomaju/internal/chat"
	"picomaju/internal/compiler"
	"picomaju/internal/license"
	"picomaju/internal/payment"
	"picomaju/internal/settings"
	"picomaju/internal/staff"
	"picomaju/internal/task"
	"picomaju/internal/tool"
	"picomaju/internal/value"
	"picomaju/ui/templates/shell"
	licensetpl "picomaju/ui/templates/license"
	settingstpl "picomaju/ui/templates/settings"
	setuptpl "picomaju/ui/templates/setup"
	stafftpl "picomaju/ui/templates/staff"
	taskstpl "picomaju/ui/templates/tasks"
	toolstpl "picomaju/ui/templates/tools"
	valuestpl "picomaju/ui/templates/values"
)

type uiHandler struct {
	mu       sync.RWMutex
	values   *value.Store
	tasks    *task.Store
	tools    *tool.Store
	staff    *staff.Store
	chats    *chat.Store
	settings *settings.Store
	license  *license.Store
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
	h.chats = chat.NewStore(filepath.Join(dataDir, "chats.json"))
	h.license = license.NewStore(filepath.Join(dataDir, "license.json"))
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
	setuptpl.SetupStep1Page("", suggested, tz, hours, "").Render(r.Context(), w)
}

func (h *uiHandler) completeSetup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		setuptpl.SetupStep1Page("", "", "Asia/Jakarta", "", err.Error()).Render(r.Context(), w)
		return
	}
	businessName := strings.TrimSpace(r.FormValue("business_name"))
	dataDir := strings.TrimSpace(r.FormValue("data_dir"))
	tz := strings.TrimSpace(r.FormValue("timezone"))
	hours := strings.TrimSpace(r.FormValue("hours"))
	if dataDir == "" {
		setuptpl.SetupStep1Page(businessName, dataDir, tz, hours, "Data directory is required.").Render(r.Context(), w)
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
		setuptpl.SetupStep1Page(businessName, dataDir, tz, hours, "Could not save settings: "+err.Error()).Render(r.Context(), w)
		return
	}
	if err := h.initStores(dataDir); err != nil {
		setuptpl.SetupStep1Page(businessName, dataDir, tz, hours, "Could not initialise data directory: "+err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/setup/first-staff", http.StatusSeeOther)
}

func (h *uiHandler) integrationsPage(w http.ResponseWriter, r *http.Request) {
	setuptpl.SetupStep3Page(tool.CatalogByCategory(), "").Render(r.Context(), w)
}

func (h *uiHandler) completeIntegrations(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		setuptpl.SetupStep3Page(tool.CatalogByCategory(), err.Error()).Render(r.Context(), w)
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
	valuestpl.ValueListPage(vals, value.DefaultCategories, h.navData(r, "values"), activeCat).Render(r.Context(), w)
}

func (h *uiHandler) newValueForm(w http.ResponseWriter, r *http.Request) {
	valuestpl.ValueFormPage(&value.Value{Version: 1, Priority: 50}, value.DefaultCategories, h.navData(r, "values"), true, "").Render(r.Context(), w)
}

func (h *uiHandler) editValueForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	v, err := h.values.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	valuestpl.ValueFormPage(v, value.DefaultCategories, h.navData(r, "values"), false, "").Render(r.Context(), w)
}

func (h *uiHandler) createValue(w http.ResponseWriter, r *http.Request) {
	v, err := valueFromForm(r)
	if err != nil {
		valuestpl.ValueFormPage(v, value.DefaultCategories, h.navData(r, "values"), true, err.Error()).Render(r.Context(), w)
		return
	}
	if err := h.values.Create(v); err != nil {
		valuestpl.ValueFormPage(v, value.DefaultCategories, h.navData(r, "values"), true, err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/values", http.StatusSeeOther)
}

func (h *uiHandler) updateValue(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	v, err := valueFromForm(r)
	if err != nil {
		valuestpl.ValueFormPage(v, value.DefaultCategories, h.navData(r, "values"), false, err.Error()).Render(r.Context(), w)
		return
	}
	v.ID = id
	if err := h.values.Update(v); err != nil {
		valuestpl.ValueFormPage(v, value.DefaultCategories, h.navData(r, "values"), false, err.Error()).Render(r.Context(), w)
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
	SSEMergeFragment(r.Context(), w, valuestpl.ValidationFragment(res))
}

// --- Tools UI ---

func (h *uiHandler) toolList(w http.ResponseWriter, r *http.Request) {
	activeCat := r.URL.Query().Get("cat")
	tools, _ := h.tools.List()
	if tools == nil {
		tools = []tool.Tool{}
	}
	if activeCat != "" {
		catIndex := tool.CatalogByType()
		var filtered []tool.Tool
		for _, t := range tools {
			if integ, ok := catIndex[t.Type]; ok && integ.Category == activeCat {
				filtered = append(filtered, t)
			}
		}
		if filtered == nil {
			filtered = []tool.Tool{}
		}
		tools = filtered
	}
	toolstpl.ToolListPage(tools, h.navData(r, "tools"), activeCat).Render(r.Context(), w)
}

func (h *uiHandler) newToolForm(w http.ResponseWriter, r *http.Request) {
	toolstpl.NewToolPage(tool.CatalogByCategory(), h.navData(r, "tools"), "").Render(r.Context(), w)
}

func (h *uiHandler) editToolForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	t, err := h.tools.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	toolstpl.ToolFormPage(t, lookupIntegration(t.Type), h.navData(r, "tools"), "").Render(r.Context(), w)
}

func (h *uiHandler) createTool(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		toolstpl.NewToolPage(tool.CatalogByCategory(), h.navData(r, "tools"), err.Error()).Render(r.Context(), w)
		return
	}
	integID := strings.TrimSpace(r.FormValue("integration_id"))
	catalogIndex := tool.CatalogByID()
	integ, ok := catalogIndex[integID]
	if !ok {
		toolstpl.NewToolPage(tool.CatalogByCategory(), h.navData(r, "tools"), "Please select a tool.").Render(r.Context(), w)
		return
	}
	t := &tool.Tool{
		ID:    integ.ID,
		Label: integ.Label,
		Type:  integ.Type,
	}
	if err := h.tools.Create(t); err != nil {
		toolstpl.NewToolPage(tool.CatalogByCategory(), h.navData(r, "tools"), err.Error()).Render(r.Context(), w)
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
		toolstpl.ToolFormPage(existing, lookupIntegration(existing.Type), h.navData(r, "tools"), err.Error()).Render(r.Context(), w)
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
		toolstpl.ToolFormPage(t, integ, h.navData(r, "tools"), err.Error()).Render(r.Context(), w)
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
	toolCat := r.URL.Query().Get("tool_cat")
	tasks, _ := h.tasks.List()
	tools, _ := h.tools.List()
	if tasks == nil {
		tasks = []task.Task{}
	}
	if tools == nil {
		tools = []tool.Tool{}
	}
	if toolCat != "" {
		catIndex := tool.CatalogByType()
		catToolIDs := map[string]bool{}
		for _, t := range tools {
			if integ, ok := catIndex[t.Type]; ok && integ.Category == toolCat {
				catToolIDs[t.ID] = true
			}
		}
		var filtered []task.Task
		for _, t := range tasks {
			for _, id := range t.Tools {
				if catToolIDs[id] {
					filtered = append(filtered, t)
					break
				}
			}
		}
		if filtered == nil {
			filtered = []task.Task{}
		}
		tasks = filtered
	}
	taskstpl.TaskListPage(tasks, h.navData(r, "tasks"), toolCat).Render(r.Context(), w)
}

func (h *uiHandler) newTaskForm(w http.ResponseWriter, r *http.Request) {
	tools, _ := h.tools.List()
	taskstpl.TaskFormPage(&task.Task{}, tools, h.navData(r, "tasks"), true, "").Render(r.Context(), w)
}

func (h *uiHandler) editTaskForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tk, err := h.tasks.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	tools, _ := h.tools.List()
	taskstpl.TaskFormPage(tk, tools, h.navData(r, "tasks"), false, "").Render(r.Context(), w)
}

func (h *uiHandler) createTask(w http.ResponseWriter, r *http.Request) {
	tools, _ := h.tools.List()
	tk, err := taskFromForm(r)
	if err != nil {
		taskstpl.TaskFormPage(tk, tools, h.navData(r, "tasks"), true, err.Error()).Render(r.Context(), w)
		return
	}
	if err := h.tasks.Create(tk); err != nil {
		taskstpl.TaskFormPage(tk, tools, h.navData(r, "tasks"), true, err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/tasks", http.StatusSeeOther)
}

func (h *uiHandler) updateTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tools, _ := h.tools.List()
	tk, err := taskFromForm(r)
	if err != nil {
		taskstpl.TaskFormPage(tk, tools, h.navData(r, "tasks"), false, err.Error()).Render(r.Context(), w)
		return
	}
	tk.ID = id
	if err := h.tasks.Update(tk); err != nil {
		taskstpl.TaskFormPage(tk, tools, h.navData(r, "tasks"), false, err.Error()).Render(r.Context(), w)
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
	stafftpl.StaffListPage(members, h.navData(r, "staff")).Render(r.Context(), w)
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
	compiled := r.URL.Query().Get("compiled") == "1"
	tasks, _ := h.tasks.List()
	vals, _ := h.values.List()
	tools, _ := h.tools.List()
	chats, _ := h.chats.ListByStaff(id)
	if chats == nil {
		chats = []chat.Chat{}
	}
	stafftpl.StaffDetailPage(m, tasks, tools, vals, value.DefaultCategories, h.navData(r, "staff"), section, formErr, chats, compiled).Render(r.Context(), w)
}

func (h *uiHandler) newStaffForm(w http.ResponseWriter, r *http.Request) {
	stafftpl.StaffFormPage(&staff.Staff{}, h.navData(r, "staff"), "").Render(r.Context(), w)
}

func (h *uiHandler) createStaff(w http.ResponseWriter, r *http.Request) {
	m, err := staffFromForm(r)
	if err != nil {
		stafftpl.StaffFormPage(m, h.navData(r, "staff"), err.Error()).Render(r.Context(), w)
		return
	}
	if err := h.staff.Create(m); err != nil {
		stafftpl.StaffFormPage(m, h.navData(r, "staff"), err.Error()).Render(r.Context(), w)
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

func (h *uiHandler) removeStaffTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	taskID := chi.URLParam(r, "taskId")
	m, err := h.staff.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var kept []string
	for _, t := range m.Tasks {
		if t != taskID {
			kept = append(kept, t)
		}
	}
	m.Tasks = kept
	h.staff.Update(m)
	http.Redirect(w, r, "/staff/"+id+"?s=tasks", http.StatusSeeOther)
}

func (h *uiHandler) removeStaffValueCat(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	catID := chi.URLParam(r, "catId")
	m, err := h.staff.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var kept []string
	for _, c := range m.ValueCategories {
		if c != catID {
			kept = append(kept, c)
		}
	}
	m.ValueCategories = kept
	h.staff.Update(m)
	http.Redirect(w, r, "/staff/"+id+"?s=values", http.StatusSeeOther)
}

func (h *uiHandler) removeStaffValue(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	valueID := chi.URLParam(r, "valueId")
	m, err := h.staff.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var kept []string
	for _, v := range m.Values {
		if v != valueID {
			kept = append(kept, v)
		}
	}
	m.Values = kept
	h.staff.Update(m)
	http.Redirect(w, r, "/staff/"+id+"?s=values", http.StatusSeeOther)
}

func (h *uiHandler) deleteStaff(w http.ResponseWriter, r *http.Request) {
	h.staff.Delete(chi.URLParam(r, "id"))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// --- Compiler ---

func (h *uiHandler) compileStaff(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m, err := h.staff.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	in, err := h.resolveCompilerInput(m)
	if err != nil {
		http.Redirect(w, r, "/staff/"+id+"?err="+err.Error(), http.StatusSeeOther)
		return
	}
	out := compiler.Compile(in)
	workspaceDir := h.workspaceDir(id)
	if err := compiler.Write(out, workspaceDir); err != nil {
		http.Redirect(w, r, "/staff/"+id+"?err="+err.Error(), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/staff/"+id+"?compiled=1", http.StatusSeeOther)
}

func (h *uiHandler) workspaceDir(staffID string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return filepath.Join(h.dataDir, "agents", "workspace-"+staffID)
}

func (h *uiHandler) resolveCompilerInput(m *staff.Staff) (compiler.Input, error) {
	allTasks, err := h.tasks.List()
	if err != nil {
		return compiler.Input{}, err
	}
	allTools, err := h.tools.List()
	if err != nil {
		return compiler.Input{}, err
	}
	allValues, err := h.values.List()
	if err != nil {
		return compiler.Input{}, err
	}
	cfg, _ := h.settings.Load()

	// Resolve assigned tasks
	taskSet := make(map[string]bool, len(m.Tasks))
	for _, tid := range m.Tasks {
		taskSet[tid] = true
	}
	var tasks []task.Task
	for _, t := range allTasks {
		if taskSet[t.ID] {
			tasks = append(tasks, t)
		}
	}

	// Collect tool IDs referenced by assigned tasks
	toolSet := make(map[string]bool)
	for _, t := range tasks {
		for _, tid := range t.Tools {
			toolSet[tid] = true
		}
	}
	var tools []tool.Tool
	for _, tl := range allTools {
		if toolSet[tl.ID] {
			tools = append(tools, tl)
		}
	}

	// Resolve values: categories + individual IDs
	catSet := make(map[string]bool, len(m.ValueCategories))
	for _, c := range m.ValueCategories {
		catSet[c] = true
	}
	indivSet := make(map[string]bool, len(m.Values))
	for _, vid := range m.Values {
		indivSet[vid] = true
	}
	var values []*value.Value
	for _, v := range allValues {
		if catSet[v.Category] || indivSet[v.ID] {
			values = append(values, v)
		}
	}

	return compiler.Input{
		Staff:    m,
		Tasks:    tasks,
		Tools:    tools,
		Values:   values,
		Settings: cfg,
	}, nil
}

// --- Chat UI ---

func (h *uiHandler) createChat(w http.ResponseWriter, r *http.Request) {
	staffID := chi.URLParam(r, "id")
	if _, err := h.staff.Get(staffID); err != nil {
		http.NotFound(w, r)
		return
	}
	c := &chat.Chat{
		ID:      fmt.Sprintf("%x", time.Now().UnixNano()),
		StaffID: staffID,
		Title:   "New Chat",
	}
	if err := h.chats.Create(c); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/staff/"+staffID+"/chats/"+c.ID, http.StatusSeeOther)
}

func (h *uiHandler) chatPage(w http.ResponseWriter, r *http.Request) {
	staffID := chi.URLParam(r, "id")
	chatID := chi.URLParam(r, "chatId")
	m, err := h.staff.Get(staffID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	c, err := h.chats.Get(chatID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	chats, _ := h.chats.ListByStaff(staffID)
	if chats == nil {
		chats = []chat.Chat{}
	}
	l, _ := h.license.Load()
	stafftpl.StaffChatPage(m, c, chats, h.navData(r, "staff"), l.IsActive()).Render(r.Context(), w)
}

func (h *uiHandler) createMessage(w http.ResponseWriter, r *http.Request) {
	staffID := chi.URLParam(r, "id")
	chatID := chi.URLParam(r, "chatId")
	l, _ := h.license.Load()
	if !l.IsActive() {
		http.Redirect(w, r, "/license", http.StatusSeeOther)
		return
	}
	c, err := h.chats.Get(chatID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/staff/"+staffID+"/chats/"+chatID, http.StatusSeeOther)
		return
	}
	content := strings.TrimSpace(r.FormValue("content"))
	if content == "" {
		http.Redirect(w, r, "/staff/"+staffID+"/chats/"+chatID, http.StatusSeeOther)
		return
	}
	c.Messages = append(c.Messages, chat.Message{
		Role:      "user",
		Content:   content,
		Timestamp: time.Now().Unix(),
	})
	if c.Title == "New Chat" && len(c.Messages) == 1 {
		if len(content) > 40 {
			c.Title = content[:40] + "…"
		} else {
			c.Title = content
		}
	}
	h.chats.Update(c) //nolint:errcheck
	http.Redirect(w, r, "/staff/"+staffID+"/chats/"+chatID, http.StatusSeeOther)
}

func (h *uiHandler) renameChat(w http.ResponseWriter, r *http.Request) {
	staffID := chi.URLParam(r, "id")
	chatID := chi.URLParam(r, "chatId")
	c, err := h.chats.Get(chatID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/staff/"+staffID+"/chats/"+chatID, http.StatusSeeOther)
		return
	}
	if title := strings.TrimSpace(r.FormValue("title")); title != "" {
		c.Title = title
		h.chats.Update(c) //nolint:errcheck
	}
	http.Redirect(w, r, "/staff/"+staffID+"/chats/"+chatID, http.StatusSeeOther)
}

func (h *uiHandler) deleteChat(w http.ResponseWriter, r *http.Request) {
	staffID := chi.URLParam(r, "id")
	chatID := chi.URLParam(r, "chatId")
	h.chats.Delete(chatID) //nolint:errcheck
	http.Redirect(w, r, "/staff/"+staffID, http.StatusSeeOther)
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
	settingstpl.SettingsPage(cfg, activeDir, h.navData(r, ""), saved, "").Render(r.Context(), w)
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
		settingstpl.SettingsPage(cfg, activeDir, h.navData(r, ""), false, err.Error()).Render(r.Context(), w)
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
			settingstpl.SettingsPage(cfg, activeDir, h.navData(r, ""), false, "Cannot use that directory: "+err.Error()).Render(r.Context(), w)
			return
		}
	}
	if err := h.settings.Save(cfg); err != nil {
		h.mu.RLock()
		activeDir = h.dataDir
		h.mu.RUnlock()
		settingstpl.SettingsPage(cfg, activeDir, h.navData(r, ""), false, err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/settings?saved=1", http.StatusSeeOther)
}

// --- License UI ---

func (h *uiHandler) licensePage(w http.ResponseWriter, r *http.Request) {
	l, _ := h.license.Load()
	if l == nil {
		l = &license.License{}
	}
	activated := r.URL.Query().Get("activated") == "1"
	formErr := r.URL.Query().Get("err")
	dev := os.Getenv("DEV") != ""
	licensetpl.LicensePage(l, h.navData(r, "license"), activated, formErr, dev).Render(r.Context(), w)
}

func (h *uiHandler) licenseCheckout(w http.ResponseWriter, r *http.Request) {
	packID := r.URL.Query().Get("pkg")
	planID := r.URL.Query().Get("plan")
	provider := payment.Provider(r.URL.Query().Get("provider"))

	cfg := payment.LoadConfig()

	var (
		redirectURL string
		err         error
	)
	switch provider {
	case payment.ProviderXendit:
		if !cfg.XenditConfigured() {
			http.Redirect(w, r, "/license?err=xendit_not_configured", http.StatusSeeOther)
			return
		}
		redirectURL, err = payment.XenditCheckoutURL(cfg, packID, planID)
	default:
		if !cfg.StripeConfigured() {
			http.Redirect(w, r, "/license?err=stripe_not_configured", http.StatusSeeOther)
			return
		}
		redirectURL, err = payment.StripeCheckoutURL(cfg, packID, planID)
	}
	if err != nil {
		http.Redirect(w, r, "/license?err="+err.Error(), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (h *uiHandler) licenseCheckoutSuccess(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/license?activated=1", http.StatusSeeOther)
}

// licenseActivateDev writes a test license directly — only available when DEV env is set.
func (h *uiHandler) licenseActivateDev(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("DEV") == "" {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plan := r.FormValue("plan") // "credits", "starter", "pro"
	if plan == "" {
		plan = license.PlanCredits
	}
	l := &license.License{
		Active: true,
		Plan:   plan,
		Token:  "dev",
	}
	switch plan {
	case license.PlanCredits:
		l.CreditsRemaining = 999
	case license.PlanStarter, license.PlanPro:
		l.ExpiresAt = time.Now().AddDate(0, 1, 0).Unix()
	}
	if err := h.license.Save(l); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/license?activated=1", http.StatusSeeOther)
}

// --- Helpers ---

func (h *uiHandler) navData(r *http.Request, section string) shell.NavData {
	cfg, _ := h.settings.Load()
	name := ""
	if cfg != nil {
		name = cfg.BusinessName
	}
	return shell.NavData{
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
