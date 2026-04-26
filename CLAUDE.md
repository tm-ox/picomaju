# Picomaju

Mobile-first agent orchestrator for small business owners. Runs via **picoclaw** (Go, single binary, <10MB RAM, native APK) on Android. UI served at `:18800`.

Four pillars: Control Plane, Directive Compiler, Sidecar Execution, Managed Lifecycle. **Implemented: Directive Compiler + UI.** Others have planning docs, no code.

## Dev workflow

```bash
task dev                     # hot reload — access at localhost:7331
templ generate && go build   # after manual .templ edits
tailwindcss -i ui/assets/css/input.css -o ui/assets/css/output.css
```

`datastar.js` must be placed in `web/static/` manually — embedded into binary at build time.

## Env vars

| Var | Default | Description |
|-----|---------|-------------|
| `PICOMAJU_CONFIG` | `~/.config/picomaju/settings.json` | config file path |
| `DATA_DIR` | — | skip onboarding; use this data dir |
| `ADDR` | `:18800` | listen address |
| `DEV` | — | serve `web/static/` from disk |

## Sub-docs

- `internal/CLAUDE.md` — packages, data models, API routes
- `ui/CLAUDE.md` — shell, templates, sidebar, UI conventions

## Deferred

Directive Compiler output (AGENTS.md, SOUL.md, picoclaw config.json injection), hot-reload via `POST /agent/:id/reload`, manifest versioning, Control Plane dashboard, Sidecar Execution, Managed Lifecycle, Overview section analytics.
