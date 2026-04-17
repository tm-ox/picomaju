# Picomaju — Agent Instructions

## What this is

Mobile-first agent orchestrator for small business owners. Runs on a dedicated Android device via **picoclaw** (Go, single static binary, <10MB RAM, native APK). Picomaju wraps picoclaw with a purpose-built web UI served on `:18800`.

Four pillars: Control Plane, Directive Compiler, Sidecar Execution, Managed Lifecycle. **Current implementation covers Directive Compiler + UI.** The other pillars have planning docs but no implementation yet.

---

## Repository layout

```
picomaju/
  main.go                        — entry point; loads settings; conditionally inits stores
  go.mod                         — module: picomaju; deps: chi/v5, yaml.v3, templ
  internal/
    settings/
      store.go                   — Settings type (business_name, business_details, data_dir) + file-backed store
    value/
      model.go                   — Value, DirectiveEntry, ValidationResult, ValidationError types
      store.go                   — file-based CRUD (.md + YAML frontmatter)
      validator.go               — required field check, priority clamp [0–100]
      category.go                — Category type + DefaultCategories (built-in, no separate store)
    tool/
      store.go                   — Tool type (id, label, type, config) + CRUD on tools.json
      catalog.go                 — Integration catalog: 8 entries, CatalogByCategory/ID/Type
    task/
      store.go                   — Task type (id, label, description, tools[]) + CRUD on tasks.json
    staff/
      store.go                   — Staff type (id, label, tasks[], value_categories[], values[]) + CRUD on staff.json
    api/
      router.go                  — all routes wired here; setup gate middleware
      ui.go                      — HTML + SSE handlers; uiHandler with mutex-guarded store init
      sse.go                     — SSEMergeFragment() for datastar
      helpers.go                 — jsonOK / jsonErr
  web/
    templates/
      layout.templ               — base HTML shell; top nav (business + avatar | section links | theme + settings); logo-type.svg during onboarding; inline SVG wordmark in footer
      sidebar.templ              — contextual sidebar (sidebarHeader component with toggle + heading inline; switches per active section)
      icons.templ                — toolIcon(type) switch with brand SVGs; categoryIcon, taskItemIcon, staffItemIcon placeholders
      values.templ               — Value list page, Value form, ValidationFragment
      tools.templ                — Tool list, NewToolPage (catalog radio picker), ToolFormPage (edit integration credentials)
      tasks.templ                — Task list page, Task form (with tool picker)
      staff.templ                — Staff list page, Staff form (task + value picker)
      settings.templ             — Settings page (business info + data dir)
      setup.templ                — Two-step onboarding: SetupPage + IntegrationsPage (catalog picker)
      helpers.go                 — SidebarData type, includesStr()
    static/
      style.css                  — CSS custom properties; light + dark mode; 16px REM base
      datastar.js                — MUST be downloaded manually from data-star.dev releases
      logo-symbol.svg            — crimson PM mark (140×98)
      logo-type.svg              — horizontal symbol + wordmark lockup (used in onboarding header)
      logo-stack.svg             — stacked symbol + wordmark lockup
```

---

## First run / onboarding

On first launch, if no data directory is configured, **all routes redirect to `/setup`**. Onboarding is two steps and renders without a sidebar (minimal header with logo-type.svg, full-width layout):

1. **`/setup`** — Business Name + Data Directory (pre-filled `~/picomaju`)
2. **`/setup/integrations`** — Tool picker: select from the catalog to auto-create Tool entries; credentials configured later in Tools. Both steps exempt from the setup gate middleware.

On completion the user lands at `/values`. No restart required.

The settings file location is platform-aware (`os.UserConfigDir()/picomaju/settings.json`). On Linux: `~/.config/picomaju/settings.json`.

---

## Data model

### Value (`<data_dir>/values/<id>.md`)
Markdown file with YAML frontmatter. Org-level directives — tone, goals, policies.

```yaml
---
id: tone_of_voice
title: Tone of Voice
version: 1
priority: 80
category: core_values
---

Always respond in a warm, professional tone...
```

Required fields: `id`, `title`, `version`, `priority` (0–100), `category`.

### Tool (`<data_dir>/tools.json`)
Catalog-based integrations. `type` matches a catalog entry (e.g. `whatsapp`, `telegram`, `shopee`). Config holds credentials keyed by `ConfigField.Key`.

```json
{ "tools": [
  { "id": "whatsapp", "label": "WhatsApp Business", "type": "whatsapp",
    "config": { "phone_number_id": "...", "access_token": "..." } }
]}
```

### Task (`<data_dir>/tasks.json`)
Task definition. Describes what the agent does and which Tools it uses.

```json
{ "tasks": [
  {
    "id": "manage_social_media",
    "label": "Manage Social Media",
    "description": "Post updates and respond to comments",
    "tools": ["tiktok_shop"]
  }
]}
```

### Staff (`<data_dir>/staff.json`)
Agent profile. Composed of Tasks + Values. The compile target.

```json
{ "staff": [
  {
    "id": "support_agent",
    "label": "Support Agent",
    "tasks": ["manage_social_media"],
    "value_categories": ["core_values", "communication"],
    "values": ["escalation_override"]
  }
]}
```

`value_categories` → bulk inclusion of all Values in those categories.
`values` → individual Value IDs added on top.

### Settings (`os.UserConfigDir()/picomaju/settings.json`)

```json
{
  "business_name": "Acme Co.",
  "business_details": "...",
  "data_dir": "/home/tm/picomaju"
}
```

---

## Value categories

Built-in constants in `internal/value/category.go` — no separate file on disk.

| ID | Label |
|----|-------|
| `core_values` | Core Values |
| `communication` | Communication |
| `skills` | Skills |
| `escalation` | Escalation |
| `custom` | Custom |

---

## Integration catalog

8 entries in `internal/tool/catalog.go`:

| ID | Label | Category |
|----|-------|----------|
| `whatsapp` | WhatsApp Business | messaging |
| `telegram` | Telegram Bot | messaging |
| `instagram` | Instagram | messaging |
| `tiktok_shop` | TikTok Shop | commerce |
| `shopee` | Shopee | commerce |
| `xendit` | Xendit | payments |
| `midtrans` | Midtrans | payments |
| `google_calendar` | Google Calendar | utilities |

---

## Compiler

**Deferred.** Output will be multiple files per Staff member: `AGENTS.md`, `SOUL.md`, tool injection into picoclaw `config.json`. Pipeline logic exists as a stub (`value.DirectiveEntry`) but serialization is not implemented.

---

## UI routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/setup` | onboarding step 1 — business name + data dir |
| POST | `/setup` | save step 1 → redirect `/setup/integrations` |
| GET | `/setup/integrations` | onboarding step 2 — tool picker |
| POST | `/setup/integrations` | create selected tools → redirect `/values` |
| GET | `/` | redirect → `/values` |
| GET | `/values` | Value list (filterable by `?cat=<id>`) |
| GET | `/values/new` | Value form (new) |
| POST | `/values` | create Value → redirect `/values` |
| GET | `/values/:id/edit` | Value form (edit) |
| POST | `/values/:id` | update Value → redirect `/values` |
| POST | `/values/:id/delete` | delete Value → redirect `/values` |
| POST | `/values/:id/validate-stream` | SSE: `ValidationFragment` |
| GET | `/tools` | Tool list |
| GET | `/tools/new` | Add Tool — catalog radio card picker |
| POST | `/tools` | create Tool from `integration_id` → redirect to edit |
| GET | `/tools/:id/edit` | Edit Tool (credential fields) |
| POST | `/tools/:id` | update Tool → redirect `/tools` |
| POST | `/tools/:id/delete` | delete Tool → redirect `/tools` |
| GET | `/tasks` | Task list |
| GET | `/tasks/new` | Task form (new) |
| POST | `/tasks` | create Task → redirect `/tasks` |
| GET | `/tasks/:id/edit` | Task form (edit) |
| POST | `/tasks/:id` | update Task → redirect `/tasks` |
| POST | `/tasks/:id/delete` | delete Task → redirect `/tasks` |
| GET | `/staff` | Staff list |
| GET | `/staff/new` | Staff form (new) |
| POST | `/staff` | create Staff → redirect `/staff` |
| GET | `/staff/:id/edit` | Staff form (edit) |
| POST | `/staff/:id` | update Staff → redirect `/staff` |
| POST | `/staff/:id/delete` | delete Staff → redirect `/staff` |
| GET | `/settings` | settings page |
| POST | `/settings` | save settings → redirect `/settings?saved=1` |
| GET | `/static/*` | embedded static assets |

Mutations use standard HTML form POST + redirect (no JS required for CRUD). Validate uses datastar SSE — button triggers a POST, server responds with a merge-fragments event patching `<div id="validation-result">`.

The setup gate middleware redirects all routes except `/setup`, `/setup/integrations`, and `/static/*` to `/setup` when no data directory is configured.

---

## Navigation

**Top nav** (sticky, `var(--topnav-h): 3rem`):
- Left: avatar placeholder + business name (or logo-type.svg during onboarding)
- Center: Values | Tools | Tasks | Staff (section links, `.active` on current section)
- Right: theme toggle + settings gear

**Sidebar** (contextual, collapsible — hidden entirely during onboarding):
- Header row: collapse/expand chevron (`‹`) on the left + section heading on the right, inline
- Collapsed state: `3rem` wide icon strip; section items show icons only (labels hidden)
- Expanded state: `13.75rem` (220px)
- All item lists sorted alphabetically by label
- Mobile (`≤640px`): `position: fixed` floating overlay; `.app-body` has `padding-left: 3rem` so content is never hidden behind the collapsed strip; collapse/expand behaviour identical to desktop
- Collapsed state persists in `localStorage` (`sidebar-collapsed`); defaults to collapsed on mobile if no saved preference
- Content switches per `ActiveSection` in `SidebarData`:
  - `values` → category filter links (tag icon) with counts + New Value
  - `tools` → tool list (brand SVG icons via `toolIcon(type)`) + Add Tool
  - `tasks` → task list (document icon placeholder) + New Task
  - `staff` → staff list (person icon placeholder) + New Staff

**Footer**: inline SVG wordmark (`fill="currentColor"` — adapts to light/dark).

`SidebarData` (in `web/templates/helpers.go`) is built by `uiHandler.sidebarData(r, section)` and passed to every page render.

---

## Icons

`web/templates/icons.templ` contains:
- `toolIcon(toolType string)` — switch over all 8 catalog types; brand SVG paths from Simple Icons; Midtrans falls back to a generic credit card; unknown types fall back to a stacked-layers icon
- `categoryIcon()` — tag placeholder for value category items
- `taskItemIcon()` — document placeholder for task items
- `staffItemIcon()` — person placeholder for staff items

All icons use `fill="currentColor"` and inherit color from the sidebar item's CSS.

---

## Theming

`style.css` uses CSS custom properties throughout. Two token blocks:

- `:root` — light mode (default)
- `[data-theme="dark"]` — dark mode

Palette: `#bf092f` crimson, `#132440` dark navy, `#16476a` mid navy, `#3b9797` teal.
Base font-size: `1rem` (16px).

Theme toggle switches `data-theme` on `<html>` and persists in `localStorage`. Inline `<script>` in `<head>` applies the saved theme before first paint to prevent flash.

---

## Environment variables

| Var | Default | Description |
|-----|---------|-------------|
| `PICOMAJU_CONFIG` | `os.UserConfigDir()/picomaju/settings.json` | override config file path |
| `DATA_DIR` | value from settings, else empty | skip onboarding; use this data dir directly |
| `ADDR` | `:18800` | listen address |
| `DEV` | — | if set, serve `web/static/` from disk |

---

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

---

## Implementation status

**Done:** Two-step onboarding with tool catalog picker, settings, Values authoring + validation, Tools CRUD (catalog integrations with per-type credential fields), integration catalog (8 integrations, Indonesian market), Task definitions with tool picker, Staff profiles with task + value picker, top-nav + contextual collapsible sidebar with brand/placeholder icons (hidden during onboarding), mobile floating sidebar, alphabetical sidebar sorting, light/dark theming, pick-chip selection UX (no visible checkboxes), logo SVGs (symbol, type lockup, stack lockup).

**Deferred:** Compiler output (AGENTS.md, SOUL.md, picoclaw config.json injection), hot-reload via `POST /agent/:id/reload`, manifest versioning, Control Plane dashboard, Sidecar Execution, Managed Lifecycle.

---

## Planning docs

Full architecture and design decisions live in the Obsidian vault at:
`40_projects/43_picomaju/43.02_planning/`
