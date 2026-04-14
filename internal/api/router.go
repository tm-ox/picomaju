package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"picomaju/internal/category"
	"picomaju/internal/role"
	"picomaju/internal/settings"
	"picomaju/internal/sop"
)

func NewRouter(sopStore *sop.Store, roleStore *role.Store, catStore *category.Store, settingsStore *settings.Store, dataDir string, static http.FileSystem) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	ui := &uiHandler{
		sops:     sopStore,
		roles:    roleStore,
		cats:     catStore,
		settings: settingsStore,
		dataDir:  dataDir,
	}

	// Gate: redirect to /setup until data dir is configured.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if !ui.configured() &&
				req.URL.Path != "/setup" &&
				!strings.HasPrefix(req.URL.Path, "/static/") {
				http.Redirect(w, req, "/setup", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, req)
		})
	})

	// JSON API
	r.Route("/api", func(r chi.Router) {
		r.Route("/sops", func(r chi.Router) {
			h := &sopHandler{store: sopStore}
			r.Get("/", h.list)
			r.Post("/", h.create)
			r.Get("/{id}", h.get)
			r.Put("/{id}", h.update)
			r.Delete("/{id}", h.delete)
			r.Post("/{id}/validate", h.validate)
		})
		r.Route("/roles", func(r chi.Router) {
			h := &roleHandler{roles: roleStore, sops: sopStore}
			r.Get("/", h.list)
			r.Post("/", h.create)
			r.Get("/{id}", h.get)
			r.Put("/{id}", h.update)
			r.Delete("/{id}", h.delete)
			r.Post("/{id}/compile", h.compile)
		})
		r.Route("/categories", func(r chi.Router) {
			h := &categoryHandler{store: catStore}
			r.Get("/", h.list)
			r.Post("/", h.create)
			r.Delete("/{id}", h.delete)
		})
	})

	// Setup (onboarding)
	r.Get("/setup", ui.setupPage)
	r.Post("/setup", ui.completeSetup)

	// HTML UI
	r.Get("/", ui.sopList)
	r.Get("/sops/new", ui.newSOPForm)
	r.Get("/sops/{id}/edit", ui.editSOPForm)
	r.Post("/sops", ui.createSOP)
	r.Post("/sops/{id}", ui.updateSOP)
	r.Post("/sops/{id}/delete", ui.deleteSOP)
	r.Post("/sops/{id}/validate-stream", ui.validateSSE)

	r.Get("/roles", ui.roleList)
	r.Get("/roles/new", ui.newRoleForm)
	r.Get("/roles/{id}/edit", ui.editRoleForm)
	r.Post("/roles", ui.createRole)
	r.Post("/roles/{id}", ui.updateRole)
	r.Post("/roles/{id}/delete", ui.deleteRole)
	r.Post("/roles/{id}/compile-stream", ui.compileSSE)

	r.Get("/settings", ui.settingsPage)
	r.Post("/settings", ui.saveSettings)

	// Static assets
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(static)))

	return r
}
