package api

import (
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"picomaju/internal/chat"
	"picomaju/internal/license"
	"picomaju/internal/llmproxy"
	"picomaju/internal/picoclaw"
	"picomaju/internal/session"
	"picomaju/internal/settings"
	"picomaju/internal/staff"
	"picomaju/internal/task"
	"picomaju/internal/tool"
	"picomaju/internal/user"
	"picomaju/internal/value"
)

func NewRouter(valStore *value.Store, taskStore *task.Store, toolStore *tool.Store, staffStore *staff.Store, chatStore *chat.Store, licenseStore *license.Store, settingsStore *settings.Store, userStore *user.Store, sessions *session.Store, dataDir string, static http.FileSystem, pm *picoclaw.Manager) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	ui := &uiHandler{
		values:   valStore,
		tasks:    taskStore,
		tools:    toolStore,
		staff:    staffStore,
		chats:    chatStore,
		license:  licenseStore,
		settings: settingsStore,
		users:    userStore,
		sessions: sessions,
		dataDir:  dataDir,
		picoclaw: pm,
	}

	// Paths exempt from the setup gate.
	setupPaths := map[string]bool{
		"/welcome":            true,
		"/setup":              true,
		"/setup/owner":        true,
		"/setup/first-staff":  true,
		"/setup/integrations": true,
	}

	// exempt returns true for paths that bypass both the setup gate and the auth gate.
	exempt := func(path string) bool {
		return setupPaths[path] ||
			strings.HasPrefix(path, "/static/") ||
			strings.HasPrefix(path, "/ui/") ||
			strings.HasPrefix(path, "/webhooks/") ||
			strings.HasPrefix(path, "/proxy/") ||
			path == "/login"
	}

	// Gate 1: redirect to /welcome until data dir is configured.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if !ui.configured() && !exempt(req.URL.Path) {
				http.Redirect(w, req, "/welcome", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, req)
		})
	})

	// Gate 2: auth — redirect to /login when users exist and no valid session.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if exempt(req.URL.Path) || !ui.configured() {
				next.ServeHTTP(w, req)
				return
			}
			// If no users are configured yet, bypass auth entirely.
			n, _ := userStore.Count()
			if n == 0 {
				next.ServeHTTP(w, req)
				return
			}
			uid, ok := sessions.FromRequest(req)
			if !ok {
				http.Redirect(w, req, "/login", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, session.WithUser(req, uid))
		})
	})

	// LLM proxy — picoclaw routes LLM calls here; proxy adds Anthropic auth + metering.
	proxy := llmproxy.NewHandler(licenseStore, os.Getenv("ANTHROPIC_API_KEY"))
	r.Handle("/proxy/v1/*", proxy)

	// Auth
	r.Get("/login", ui.loginPage)
	r.Post("/login", ui.loginSubmit)
	r.Post("/logout", ui.logout)

	// Profile (any logged-in user)
	r.Get("/profile", ui.profilePage)
	r.Post("/profile", ui.updateProfile)

	// Users (owner only)
	r.Get("/users", ui.userList)
	r.Get("/users/new", ui.newUserForm)
	r.Post("/users", ui.createUser)
	r.Get("/users/{id}/edit", ui.editUserForm)
	r.Post("/users/{id}", ui.updateUser)
	r.Post("/users/{id}/delete", ui.deleteUser)

	// Setup (onboarding) — welcome + 4 steps
	r.Post("/welcome", ui.completeWelcome)          // -> /setup
	r.Get("/setup", ui.setupPage)                   // step 1 — business info
	r.Post("/setup", ui.completeSetup)              //   -> /setup/owner
	r.Get("/setup/owner", ui.ownerPage)             // step 2 — owner account
	r.Post("/setup/owner", ui.completeOwner)        //   -> /setup/first-staff
	r.Get("/setup/first-staff", ui.firstStaffPage)  // step 3 — first staff profile
	r.Post("/setup/first-staff", ui.completeFirstStaff) //   -> /setup/integrations
	r.Get("/setup/integrations", ui.integrationsPage)   // step 4 — tool picker
	r.Post("/setup/integrations", ui.completeIntegrations) // -> /values

	// Home
	r.Get("/", ui.homePage)

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
	r.Post("/staff/{id}/activate", ui.activateStaff)
	r.Post("/staff/{id}/deactivate", ui.deactivateStaff)

	// Chats
	r.Post("/staff/{id}/chats", ui.createChat)
	r.Get("/staff/{id}/chats/{chatId}", ui.chatPage)
	r.Post("/staff/{id}/chats/{chatId}/messages", ui.createMessage)
	r.Post("/staff/{id}/chats/{chatId}/rename", ui.renameChat)
	r.Post("/staff/{id}/chats/{chatId}/delete", ui.deleteChat)

	// License
	r.Get("/license", ui.licensePage)
	r.Get("/license/checkout", ui.licenseCheckout)
	r.Get("/license/checkout/success", ui.licenseCheckoutSuccess)
	r.Post("/license/activate-dev", ui.licenseActivateDev)

	// Payment webhooks — exempt from setup gate via path prefix
	r.Post("/webhooks/stripe", ui.stripeWebhook)
	r.Post("/webhooks/xendit", ui.xenditWebhook)

	// Settings
	r.Get("/settings", ui.settingsPage)
	r.Post("/settings", ui.saveSettings)

	// Static assets
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(static)))

	return r
}
