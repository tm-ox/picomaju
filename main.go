package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"picomaju/internal/api"
	"picomaju/internal/settings"
	"picomaju/internal/staff"
	"picomaju/internal/task"
	"picomaju/internal/tool"
	"picomaju/internal/value"
	"picomaju/ui/templates"
)

//go:embed web/static
var staticFiles embed.FS

func main() {
	configFile := configPath()
	settingsStore := settings.NewStore(configFile)
	cfg, err := settingsStore.Load()
	if err != nil {
		log.Printf("warn: could not load settings: %v", err)
		cfg = &settings.Settings{}
	}

	// DATA_DIR env overrides saved setting (useful for managed deployments).
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = cfg.DataDir
	}

	// Init stores only when we have a data dir; otherwise the setup flow handles it.
	var valStore *value.Store
	var taskStore *task.Store
	var toolStore *tool.Store
	var staffStore *staff.Store
	if dataDir != "" {
		valStore = value.NewStore(filepath.Join(dataDir, "values"))
		taskStore = task.NewStore(filepath.Join(dataDir, "tasks.json"))
		toolStore = tool.NewStore(filepath.Join(dataDir, "tools.json"))
		staffStore = staff.NewStore(filepath.Join(dataDir, "staff.json"))
	}

	var static http.FileSystem
	if os.Getenv("DEV") != "" {
		static = http.Dir("web/static")
		log.Println("dev mode: serving static files from disk")
	} else {
		sub, err := fs.Sub(staticFiles, "web/static")
		if err != nil {
			log.Fatal(err)
		}
		static = http.FS(sub)
	}

	addr := env("ADDR", ":18800")
	r := api.NewRouter(valStore, taskStore, toolStore, staffStore, settingsStore, dataDir, static)

	// ui/assets served from disk during active development
	r.Handle("GET /ui/assets/*", http.StripPrefix("/ui/assets/", http.FileServer(http.Dir("ui/assets"))))

	// new frontend routes
	r.Get("/welcome", func(w http.ResponseWriter, req *http.Request) {
		templates.WelcomePage("", "").Render(req.Context(), w)
	})
	r.Get("/ui/workshop", func(w http.ResponseWriter, req *http.Request) {
		templates.WorkshopPage().Render(req.Context(), w)
	})
	r.Get("/ui/home", func(w http.ResponseWriter, req *http.Request) {
		nd := templates.NavData{ActiveSection: "home"}
		if cfg != nil {
			nd.BusinessName = cfg.BusinessName
		}
		templates.DashboardPage(nd).Render(req.Context(), w)
	})

	log.Printf("picomaju listening on %s (config: %s)", addr, configFile)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}

// configPath returns the platform-appropriate settings file location.
// Can be overridden with PICOMAJU_CONFIG env.
func configPath() string {
	if v := os.Getenv("PICOMAJU_CONFIG"); v != "" {
		return v
	}
	base, err := os.UserConfigDir()
	if err != nil {
		base = "."
	}
	return filepath.Join(base, "picomaju", "settings.json")
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
