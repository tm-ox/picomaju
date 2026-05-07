package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"picomaju/internal/chat"
	"picomaju/internal/settings"
	"picomaju/internal/staff"
	"picomaju/internal/task"
	"picomaju/internal/tool"
	"picomaju/internal/value"
)

func NewRouter(valStore *value.Store, taskStore *task.Store, toolStore *tool.Store, staffStore *staff.Store, chatStore *chat.Store, settingsStore *settings.Store, dataDir string, static http.FileSystem) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	ui := &uiHandler{
		values:   valStore,
		tasks:    taskStore,
		tools:    toolStore,
		staff:    staffStore,
		chats:    chatStore,
		settings: settingsStore,
		dataDir:  dataDir,
	}

	// Paths that must work before the data dir is configured.
	setupPaths := map[string]bool{
		"/welcome":               true,
		"/setup":                 true,
		"/setup/first-staff":     true,
		"/setup/integrations":    true,
	}

	// Gate: redirect to /welcome until data dir is configured.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if !ui.configured() &&
				!setupPaths[req.URL.Path] &&
				!strings.HasPrefix(req.URL.Path, "/static/") &&
				!strings.HasPrefix(req.URL.Path, "/ui/") {
				http.Redirect(w, req, "/welcome", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, req)
		})
	})

	// Setup (onboarding) — welcome + 3 steps
	r.Post("/welcome", ui.completeWelcome) //   -> /setup
	r.Get("/setup", ui.setupPage)                                     // step 1 — business + data dir + tz + hours
	r.Post("/setup", ui.completeSetup)                                //   -> /setup/first-staff
	r.Get("/setup/first-staff", ui.firstStaffPage)                    // step 2 — first staff profile
	r.Post("/setup/first-staff", ui.completeFirstStaff)               //   -> /setup/integrations
	r.Get("/setup/integrations", ui.integrationsPage)                 // step 3 — tool picker
	r.Post("/setup/integrations", ui.completeIntegrations)            // -> /values

	// Home — staff list
	r.Get("/", ui.staffList)

	// Values
	r.Get("/values", ui.valueList)
	r.Get("/values/new", ui.newValueForm)
	r.Get("/values/{id}/edit", ui.editValueForm)
	r.Post("/values", ui.createValue)
	r.Post("/values/{id}", ui.updateValue)
	r.Post("/values/{id}/delete", ui.deleteValue)
	r.Post("/values/{id}/validate-stream", ui.validateSSE)

	// Tools
	r.Get("/tools", ui.toolList)
	r.Get("/tools/new", ui.newToolForm)
	r.Get("/tools/{id}/edit", ui.editToolForm)
	r.Post("/tools", ui.createTool)
	r.Post("/tools/{id}", ui.updateTool)
	r.Post("/tools/{id}/delete", ui.deleteTool)

	// Tasks
	r.Get("/tasks", ui.taskList)
	r.Get("/tasks/new", ui.newTaskForm)
	r.Get("/tasks/{id}/edit", ui.editTaskForm)
	r.Post("/tasks", ui.createTask)
	r.Post("/tasks/{id}", ui.updateTask)
	r.Post("/tasks/{id}/delete", ui.deleteTask)

	// Staff
	r.Get("/staff", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	r.Get("/staff/new", ui.newStaffForm)
	r.Get("/staff/{id}", ui.staffDetail)
	r.Get("/staff/{id}/edit", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/staff/"+chi.URLParam(r, "id"), http.StatusSeeOther)
	})
	r.Post("/staff", ui.createStaff)
	r.Post("/staff/{id}/profile", ui.updateStaffProfile)
	r.Post("/staff/{id}/tasks", ui.updateStaffTasks)
	r.Post("/staff/{id}/tasks/{taskId}/remove", ui.removeStaffTask)
	r.Post("/staff/{id}/values", ui.updateStaffValues)
	r.Post("/staff/{id}/value-cats/{catId}/remove", ui.removeStaffValueCat)
	r.Post("/staff/{id}/values/{valueId}/remove", ui.removeStaffValue)
	r.Post("/staff/{id}/delete", ui.deleteStaff)
	r.Post("/staff/{id}/compile", ui.compileStaff)

	// Chats
	r.Post("/staff/{id}/chats", ui.createChat)
	r.Get("/staff/{id}/chats/{chatId}", ui.chatPage)
	r.Post("/staff/{id}/chats/{chatId}/messages", ui.createMessage)
	r.Post("/staff/{id}/chats/{chatId}/rename", ui.renameChat)
	r.Post("/staff/{id}/chats/{chatId}/delete", ui.deleteChat)

	// Settings
	r.Get("/settings", ui.settingsPage)
	r.Post("/settings", ui.saveSettings)

	// Static assets
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(static)))

	return r
}
