# Picomaju

Mobile-first agent orchestrator for small business owners. Runs on a dedicated Android device via **picoclaw** (Go, single static binary, <10MB RAM, native APK). Picomaju wraps picoclaw with a purpose-built web UI served on `:18800`.

Four pillars: Control Plane, Directive Compiler, Sidecar Execution, Managed Lifecycle. **Current implementation covers Directive Compiler + UI.** The other pillars have planning docs but no implementation yet.

## Repository layout

```
picomaju/
  main.go                        — entry point; loads settings; conditionally inits stores
  go.mod                         — module: picomaju; deps: chi/v5, yaml.v3, templ
  internal/                      — see internal/CLAUDE.md
  web/                           — see web/templates/CLAUDE.md
    templates/
    static/
      style.css
      datastar.js                — MUST be downloaded manually from data-star.dev releases
      logo-symbol.svg            — crimson PM mark (140×98), hardcoded fill="#bf092f"
      logo-type.svg              — horizontal wordmark lockup
      logo-stack.svg             — stacked symbol + wordmark lockup
```

## Environment variables

| Var | Default | Description |
|-----|---------|-------------|
| `PICOMAJU_CONFIG` | `os.UserConfigDir()/picomaju/settings.json` | override config file path |
| `DATA_DIR` | value from settings, else empty | skip onboarding; use this data dir directly |
| `ADDR` | `:18800` | listen address |
| `DEV` | — | if set, serve `web/static/` from disk |

## Development workflow

```bash
DEV=1 go run .
```

After editing any `.templ` file:

```bash
templ generate
go build ./...
```

`datastar.js` must be placed in `web/static/` manually — it is embedded into the binary at build time.

Build excludes the `patches/` directory (contains design drop-in files, not a Go package):
```bash
go build ./internal/... ./web/... .
```

## Implementation status

**Done:** Welcome screen (language picker) + three-step onboarding (business info/timezone/hours → first staff → tool picker), dashboard home screen, settings, Values authoring + validation, Tools CRUD (catalog integrations with per-type credential fields), integration catalog (8 integrations, Indonesian market focus), Task definitions with tool picker, Staff profiles with task + value picker, mobile-first UI (bottom tab bar, FAB, compact card rows, illustrated empty states, icon-only edit/delete buttons), icon-strip collapsible sidebar, light/dark theming with sun/moon icon toggle, logo SVGs.

**Deferred:** Compiler output (AGENTS.md, SOUL.md, picoclaw config.json injection), hot-reload via `POST /agent/:id/reload`, manifest versioning, Control Plane dashboard, Sidecar Execution, Managed Lifecycle.

## Planning docs

Full architecture and design decisions live in the Obsidian vault at:
`40_projects/43_picomaju/43.02_planning/`
