package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"picomaju/internal/settings"
	"picomaju/internal/staff"
	"picomaju/internal/task"
	"picomaju/internal/tool"
	"picomaju/internal/value"
)

func NewRouter(valStore *value.Store, taskStore *task.Store, toolStore *tool.Store, staffStore *staff.Store, settingsStore *settings.Store, dataDir string, static http.FileSystem) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	ui := &uiHandler{
		values:   valStore,
		tasks:    taskStore,
		tools:    toolStore,
		staff:    staffStore,
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
				!strings.HasPrefix(req.URL.Path, "/static/") {
				http.Redirect(w, req, "/welcome", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, req)
		})
	})

	// Setup (onboarding) — welcome + 3 steps
	r.Get("/welcome", ui.welcomePage)                   // welcome — language picker
	r.Post("/welcome", ui.completeWelcome)              //   -> /setup
	r.Get("/setup", ui.setupPage)                       // step 1 — business + data dir + tz + hours
	r.Post("/setup", ui.completeSetup)                  //   -> /setup/first-staff
	r.Get("/setup/first-staff", ui.firstStaffPage)      // step 2 — first staff profile
	r.Post("/setup/first-staff", ui.completeFirstStaff) //   -> /setup/integrations
	r.Get("/setup/integrations", ui.integrationsPage)   // step 3 — tool picker
	r.Post("/setup/integrations", ui.completeIntegrations) // -> /values

	// Dashboard
	r.Get("/", ui.dashboardPage)

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
	r.Get("/staff", ui.staffList)
	r.Get("/staff/new", ui.newStaffForm)
	r.Get("/staff/{id}/edit", ui.editStaffForm)
	r.Post("/staff", ui.createStaff)
	r.Post("/staff/{id}", ui.updateStaff)
	r.Post("/staff/{id}/delete", ui.deleteStaff)

	// Settings
	r.Get("/settings", ui.settingsPage)
	r.Post("/settings", ui.saveSettings)

	// Static assets
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(static)))

	return r
}
