package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"picomaju/internal/api"
	"picomaju/internal/category"
	"picomaju/internal/role"
	"picomaju/internal/settings"
	"picomaju/internal/sop"
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
	var sopStore *sop.Store
	var roleStore *role.Store
	var catStore *category.Store
	if dataDir != "" {
		sopStore = sop.NewStore(filepath.Join(dataDir, "sops"))
		roleStore = role.NewStore(filepath.Join(dataDir, "roles.json"))
		catStore = category.NewStore(filepath.Join(dataDir, "categories.json"))
		if err := catStore.Seed(); err != nil {
			log.Printf("warn: could not seed categories: %v", err)
		}
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
	r := api.NewRouter(sopStore, roleStore, catStore, settingsStore, dataDir, static)

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
		// Fallback for platforms where UserConfigDir fails (e.g. no $HOME).
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
