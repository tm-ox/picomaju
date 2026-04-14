# Picomaju — Agent Instructions

## What this is

Mobile-first agent orchestrator for small business owners. Runs on a dedicated Android device via **picoclaw** (Go, single static binary, <10MB RAM, native APK). Picomaju wraps picoclaw with a purpose-built web UI served on `:18800`.

Four pillars: Control Plane, SOP Compiler, Sidecar Execution, Managed Lifecycle. **v1 covers SOP Compiler + UI only.** The other pillars have planning docs but no implementation yet.

---

## Repository layout

```
picomaju/
  main.go                        — entry point; loads settings; conditionally inits stores
  go.mod                         — module: picomaju; deps: chi/v5, yaml.v3, templ
  internal/
    settings/
      store.go                   — Settings type (business_name, business_details, data_dir) + file-backed store
    sop/
      model.go                   — SOP, Policy, ValidationResult, CompileResult types
      store.go                   — file-based CRUD (.md + YAML frontmatter)
      validator.go               — required field check, priority clamp [0–100]
      compiler.go                — Compile(roleID, categoryIDs, individualSOPIDs, allSOPs)
    category/
      model.go                   — Category type + Defaults (starter set)
      store.go                   — CRUD on categories.json; seeds defaults if missing
    role/
      store.go                   — Role type + full CRUD on roles.json
    api/
      router.go                  — all routes wired here; setup gate middleware
      sop.go                     — JSON API handlers for SOPs
      role.go                    — JSON API handlers for roles
      category.go                — JSON API handlers for categories
      ui.go                      — HTML + SSE handlers; uiHandler with mutex-guarded store init
      sse.go                     — SSEMergeFragment() for datastar
      helpers.go                 — jsonOK / jsonErr
  web/
    templates/
      layout.templ               — base HTML shell, top nav (brand + theme toggle)
      sidebar.templ              — WithSidebar wrapper + sidebar component (collapsible)
      sops.templ                 — SOP list page, SOP form, ValidationFragment
      roles.templ                — Role list page, hiring form, CompileFragment
      settings.templ             — Settings page (business info + data dir)
      setup.templ                — First-run onboarding page
      helpers.go                 — SidebarData type, compileResultJSON(), includesStr()
    static/
      style.css                  — CSS custom properties for theming; light + dark mode
      datastar.js                — MUST be downloaded manually from data-star.dev releases
```

---

## First run / onboarding

On first launch, if no data directory is configured, **all routes redirect to `/setup`**. The setup page asks for:

- **Business Name** — shown in the sidebar header
- **Data Directory** — where SOPs, roles, and categories are stored; pre-filled with `~/picomaju`

On submit, the directory is created, category defaults are seeded, and the user is dropped into the main app. No restart required.

The settings file location is platform-aware (`os.UserConfigDir()/picomaju/settings.json`) and created automatically. On Linux this is `~/.config/picomaju/settings.json`.

---

## Data model

### SOP (`<data_dir>/sops/<id>.md`)
Markdown file with YAML frontmatter. Each SOP is an atomic unit — it has no knowledge of which roles use it.

```yaml
---
id: handle_refund
title: Handle Refund Request
version: 1
priority: 80
trigger: customer mentions refund
category: tasks
---

When a customer requests a refund...
```

Required fields: `id`, `title`, `version`, `priority` (0–100), `trigger`, `category`.
No `roles` field — role assignment is managed by the Role Definition.

### Category (`<data_dir>/categories.json`)
Named grouping for SOPs. Starter set is seeded on first run. System categories cannot be deleted.

```json
{ "categories": [
  { "id": "business_objectives", "label": "Core Values",    "system": true },
  { "id": "communication",       "label": "Communication",  "system": true },
  { "id": "tasks",               "label": "Skills",         "system": true },
  { "id": "escalation",          "label": "Escalation",     "system": true },
  { "id": "uncategorized",       "label": "Custom",         "system": true }
]}
```

`uncategorized` exists as a general-purpose bucket; new SOPs must have an explicit category.

### Role Definition (`<data_dir>/roles.json`)
The "job description" for an agent. Drives compilation via category bulk-inclusion and individual SOP overrides.

```json
{ "roles": [
  {
    "id": "support_agent",
    "label": "Support Agent",
    "categories": ["communication", "tasks"],
    "sops": ["refund_vip_override"]
  }
]}
```

`categories` → bulk: all SOPs in those categories are included.
`sops` → individual overrides: specific SOP IDs added regardless of their category.
Deduplication: if an individual SOP's category is also selected, it appears once.

### Settings (`os.UserConfigDir()/picomaju/settings.json`)

```json
{
  "business_name": "Acme Co.",
  "business_details": "...",
  "data_dir": "/home/tm/picomaju"
}
```

Loaded at startup to determine the data directory. Editable at `/settings`; data dir changes take effect immediately (stores are reinitialized live, no restart needed).

---

## Compiler pipeline

`sop.Compile(roleID, categoryIDs, individualSOPIDs, allSOPs) → CompileResult`

1. Collect SOPs where `sop.Category` is in `categoryIDs`
2. Append individually selected SOPs, skipping duplicates
3. Validate all collected SOPs (gate — abort on any failure)
4. Sort descending by `priority` (stable sort preserves load order on ties)
5. Assemble `policies[]` array

v1 output: in-memory `CompileResult` returned to the caller (displayed in UI).
v2: JSON manifest written to `<data_dir>/manifests/<role>.json` + hot-reload via picoclaw.

---

## HTTP API

All JSON API routes are under `/api/`. UI routes are at root.

### SOPs
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/sops` | list all |
| POST | `/api/sops` | create |
| GET | `/api/sops/:id` | get one |
| PUT | `/api/sops/:id` | replace |
| DELETE | `/api/sops/:id` | delete |
| POST | `/api/sops/:id/validate` | validate → `ValidationResult` JSON |

### Roles
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/roles` | list all |
| POST | `/api/roles` | create |
| GET | `/api/roles/:id` | get one |
| PUT | `/api/roles/:id` | replace |
| DELETE | `/api/roles/:id` | delete |
| POST | `/api/roles/:id/compile` | compile → `CompileResult` JSON |

### Categories
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/categories` | list all |
| POST | `/api/categories` | create custom |
| DELETE | `/api/categories/:id` | delete (system categories rejected) |

---

## UI routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/setup` | onboarding (shown when no data dir configured) |
| POST | `/setup` | complete setup → redirect `/` |
| GET | `/` | SOP list (filterable by `?cat=<id>`) |
| GET | `/sops/new` | SOP form (new) |
| POST | `/sops` | create SOP → redirect `/` |
| GET | `/sops/:id/edit` | SOP form (edit) |
| POST | `/sops/:id` | update SOP → redirect `/` |
| POST | `/sops/:id/delete` | delete SOP → redirect `/` |
| POST | `/sops/:id/validate-stream` | SSE: `ValidationFragment` |
| GET | `/roles` | role list |
| GET | `/roles/new` | hiring form (new) |
| POST | `/roles` | create role → redirect `/roles` |
| GET | `/roles/:id/edit` | hiring form (edit) |
| POST | `/roles/:id` | update role → redirect `/roles` |
| POST | `/roles/:id/delete` | delete role → redirect `/roles` |
| POST | `/roles/:id/compile-stream` | SSE: `CompileFragment` |
| GET | `/settings` | settings page |
| POST | `/settings` | save settings → redirect `/settings?saved=1` |
| GET | `/static/*` | embedded static assets |

Mutations from the UI use standard HTML form POST + redirect (no JS required for CRUD).
Validate and compile use datastar SSE — the button triggers a POST, server responds with a merge-fragments event that patches the relevant `<div id="...">` in place.

The setup gate middleware redirects all routes except `/setup` and `/static/*` to `/setup` when no data directory has been configured.

---

## Sidebar

Every page uses the `WithSidebar` component. The sidebar is collapsible (toggle button, state persisted in `localStorage`). It contains:

- **SOPs section** — "All" link + one link per category with SOP count badges; active category highlighted
- **Roles section** — link per role (edit page); shown only when roles exist
- **Manage Roles / + Hire Agent** links
- **Settings** link (bottom section)
- **Business name** shown at the top of the nav when set

The `SidebarData` struct (in `web/templates/helpers.go`) is built by `uiHandler.sidebarData()` and passed to every page render.

---

## Theming

`style.css` uses CSS custom properties throughout. Two blocks define tokens:

- `:root` — light mode (default)
- `[data-theme="dark"]` — dark mode

Palette: `#bf092f` crimson, `#132440` dark navy, `#16476a` mid navy, `#3b9797` teal.

The theme toggle in the top nav switches `data-theme` on `<html>` and persists the choice in `localStorage`. An inline `<script>` in `<head>` applies the saved theme before first paint to prevent flash.

---

## Environment variables

| Var | Default | Description |
|-----|---------|-------------|
| `PICOMAJU_CONFIG` | `os.UserConfigDir()/picomaju/settings.json` | override config file path |
| `DATA_DIR` | value from settings, else empty | skip onboarding; use this data dir directly |
| `ADDR` | `:18800` | listen address |
| `DEV` | — | if set, serve `web/static/` from disk (CSS/JS changes apply on browser refresh) |

`SOP_DIR`, `ROLES_FILE`, `CATEGORIES_FILE` no longer exist — paths are derived from `data_dir`.

---

## Development workflow

```bash
# First run — opens onboarding in browser
DEV=1 go run .

# Subsequent runs — data dir already saved in settings
DEV=1 go run .
```

`DEV=1` serves static files from disk so CSS/JS edits apply on browser refresh without rebuilding.

After editing any `.templ` file, regenerate before building:

```bash
templ generate
go build ./...
go test ./...
```

`datastar.js` must be placed in `web/static/` manually — it is embedded into the binary at build time. Download from the data-star.dev releases page.

---

## v1 / v2 boundary

**v1 (built):** First-run onboarding, settings (business info + data dir), SOP authoring, category management, role definitions (job descriptions), in-memory compilation, compile result preview in UI, collapsible sidebar with category filtering.

**v2 (not started):** JSON manifest serialization to disk, hot-reload via `POST /agent/:role/reload`, manifest versioning/hashing, compile-on-save, Control Plane dashboard, Sidecar Execution, Managed Lifecycle.

---

## Planning docs

Full architecture and design decisions live in the Obsidian vault at:
`40_projects/43_picomaju/43.02_planning/`

Key files: `plan.md`, `02_sop-compiler/spec.md`, `02_sop-compiler/notes.md`.
