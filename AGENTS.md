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
      store.go                   — Settings type (business_name, business_details, data_dir,
                                   languages[], timezone, hours) + file-backed store
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
      router.go                  — all routes wired here; setup gate middleware (allows setup paths + /static/*)
      ui.go                      — HTML + SSE handlers; uiHandler with mutex-guarded store init; dashboardPage
      ui_onboarding.go           — onboarding step 2 (languagesPage/completeLanguages) + step 3 (firstStaffPage/completeFirstStaff); slugify helper
      sse.go                     — SSEMergeFragment() for datastar
      helpers.go                 — jsonOK / jsonErr
  web/
    templates/
      layout.templ               — base HTML shell; topnav (avatar | Home+Values+Tools+Tasks+Staff | theme icon + settings);
                                   bottomTabs (5-tab mobile bar); fab (section floating action button);
                                   tab/fab icon glyphs; initials() helper; logo-type.svg during onboarding
      dashboard.templ            — DashboardPage: centered logo-symbol.svg home screen
      sidebar.templ              — contextual sidebar (sidebarHeader with toggle; switches per active section)
      empty_state.templ          — EmptyState component (illustrated card + CTA); EmptyIconValues/Tools/Tasks/Staff glyphs
      icons.templ                — toolIcon(type) brand SVGs; categoryIcon/taskItemIcon/staffItemIcon sidebar glyphs;
                                   tabIconHome/Values/Tools/Tasks/Staff tab bar glyphs;
                                   iconSun/iconMoon theme toggle; iconEdit/iconDelete action buttons
      values.templ               — Value list page (EmptyState when empty), Value form, ValidationFragment
      tools.templ                — Tool list (EmptyState), NewToolPage (catalog radio picker), ToolFormPage
      tasks.templ                — Task list (EmptyState), Task form (with tool picker)
      staff.templ                — Staff list (EmptyState), Staff form (task + value picker)
      settings.templ             — Settings page (business info + data dir)
      setup.templ                — Four-step onboarding: SetupPage, LanguagesPage, FirstStaffPage, IntegrationsPage;
                                   setupProgress component; itoa() helper
      helpers.go                 — SidebarData type, includesStr()
    static/
      style.css                  — CSS custom properties; light + dark mode; mobile-first (bottom tabs, FAB, card rows)
      datastar.js                — MUST be downloaded manually from data-star.dev releases
      logo-symbol.svg            — crimson PM mark (140×98), hardcoded fill="#bf092f"
      logo-type.svg              — horizontal wordmark lockup (used in onboarding header + footer)
      logo-stack.svg             — stacked symbol + wordmark lockup
```

---

## First run / onboarding

On first launch, if no data directory is configured, **all routes redirect to `/setup`**. Onboarding is four steps and renders without a sidebar (`HideSidebar: true`, minimal header with logo-type.svg, full-width layout):

1. **`/setup`** — Business Name + Data Directory (pre-filled `~/picomaju`)
2. **`/setup/languages`** — Languages (Bahasa Indonesia / English), timezone, operating hours. Defaults: `["id","en"]`, `Asia/Jakarta`.
3. **`/setup/first-staff`** — First staff profile (name + optional description). Skip button goes directly to step 4.
4. **`/setup/integrations`** — Tool picker: select from the catalog to auto-create Tool entries; credentials configured later.

All four setup paths plus `/static/*` are exempt from the setup gate middleware.

After step 1 (`POST /setup`), user is redirected to `/setup/languages`. After step 4, redirected to `/values`.

On completion the user lands at `/values`. No restart required.

The settings file location is platform-aware (`os.UserConfigDir()/picomaju/settings.json`). On Linux: `~/.config/picomaju/settings.json`.

---

## Data model

### Settings (`os.UserConfigDir()/picomaju/settings.json`)

```json
{
  "business_name": "Acme Co.",
  "business_details": "...",
  "data_dir": "/home/tm/picomaju",
  "languages": ["id", "en"],
  "timezone": "Asia/Jakarta",
  "hours": "Mon–Fri, 09:00–18:00"
}
```

`languages`, `timezone`, `hours` are `omitempty` — existing settings files without them load cleanly.

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
| GET | `/` | dashboard (home screen) |
| GET | `/setup` | onboarding step 1 — business name + data dir |
| POST | `/setup` | save step 1 → redirect `/setup/languages` |
| GET | `/setup/languages` | onboarding step 2 — languages / timezone / hours |
| POST | `/setup/languages` | save step 2 → redirect `/setup/first-staff` |
| GET | `/setup/first-staff` | onboarding step 3 — first staff profile |
| POST | `/setup/first-staff` | create staff → redirect `/setup/integrations` |
| GET | `/setup/integrations` | onboarding step 4 — tool picker |
| POST | `/setup/integrations` | create selected tools → redirect `/values` |
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

Mutations use standard HTML form POST + redirect (no JS required for CRUD). Validate uses datastar SSE.

The setup gate middleware allows `/setup`, `/setup/languages`, `/setup/first-staff`, `/setup/integrations`, and `/static/*` without redirect. All other routes redirect to `/setup` when no data directory is configured.

---

## Navigation

**Top nav** (sticky, `var(--topnav-h): 3.25rem`):
- Left: crimson avatar (business initials via `initials()`) + business name
- Center: Home | Values | Tools | Tasks | Staff (section links, `.active` on current section)
- Right: theme toggle icon button (moon/sun, `2rem` square) + settings gear
- During onboarding: minimal header with logo-type.svg only

**Sidebar** (contextual, collapsible — hidden during onboarding):
- Header row: collapse/expand chevron + section heading
- Collapsed state: `3rem` wide icon strip (both `.sidebar` and `.sidebar-inner` collapse to `3rem` so flex centering keeps icons visible)
- Expanded state: `13.75rem`
- Content switches per `ActiveSection`: `values` → category filter + New Value; `tools` → tool list + Add Tool; `tasks` → task list + New Task; `staff` → staff list + New Staff; `home`/empty → empty nav
- Collapsed state persists in `localStorage` (`sidebar-collapsed`)

**Mobile (≤640px):**
- Sidebar hidden; top nav center links hidden
- Fixed bottom tab bar: 5 tabs — Home | Values | Tools | Tasks | Staff (`grid-template-columns: repeat(5, 1fr)`)
- Floating action button (crimson circle, `3.25rem`) above tab bar for the active section's primary create action
- `viewport-fit=cover` + `env(safe-area-inset-bottom)` for iOS home bar
- Tables render as compact flex-row cards: label left (`flex: 1`), secondary columns hidden, action icon buttons right

**Footer**: logo-type.svg (`color: var(--color-text-muted)`)

`SidebarData` (in `web/templates/helpers.go`) is built by `uiHandler.sidebarData(r, section)` and passed to every page render.

---

## Icons

`web/templates/icons.templ` contains:

**Sidebar / section:**
- `toolIcon(toolType string)` — switch over all 8 catalog types; brand SVG paths from Simple Icons; Midtrans → generic credit card; unknown → stacked-layers
- `categoryIcon()` — tag glyph for value category items
- `taskItemIcon()` — document glyph for task items
- `staffItemIcon()` — person glyph for staff items

**Tab bar (stroke, 1.8px, `viewBox="0 0 24 24"`):**
- `tabIconHome()` — house glyph
- `tabIconValues()` — star glyph
- `tabIconTools()` — wrench glyph
- `tabIconTasks()` — rounded rectangle + checkmark glyph
- `tabIconStaff()` — person circle glyph

**Theme toggle (stroke, `viewBox="0 0 24 24"`):**
- `iconSun()` — `class="icon-sun"` — shown in dark mode (click → light); hidden in light mode via CSS
- `iconMoon()` — `class="icon-moon"` — shown in light mode (click → dark); hidden in dark mode via CSS

**Action buttons (stroke, `viewBox="0 0 16 16"`):**
- `iconEdit()` — pencil glyph; used in `.btn-icon` edit links
- `iconDelete()` — trash glyph; used in `.btn-icon.btn-icon-danger` delete buttons

**FAB:**
- `fabPlus()` — plus glyph

All sidebar/tab icons use `stroke="currentColor"` or `fill="currentColor"` and inherit color from the parent element's CSS.

---

## Theming

`style.css` uses CSS custom properties throughout. Two token blocks:

- `:root` — light mode (default)
- `[data-theme="dark"]` — dark mode

Palette: `#bf092f` crimson, `#132440` dark navy, `#16476a` mid navy, `#3b9797` teal.

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

Build excludes the `patches/` directory (contains design drop-in files, not a Go package):
```bash
go build ./internal/... ./web/... .
```

---

## Implementation status

**Done:** Four-step onboarding (business info → languages/timezone/hours → first staff → tool picker), dashboard home screen, settings, Values authoring + validation, Tools CRUD (catalog integrations with per-type credential fields), integration catalog (8 integrations, Indonesian market focus), Task definitions with tool picker, Staff profiles with task + value picker, mobile-first UI (bottom tab bar, FAB, compact card rows, illustrated empty states, icon-only edit/delete buttons), icon-strip collapsible sidebar, light/dark theming with sun/moon icon toggle, logo SVGs.

**Deferred:** Compiler output (AGENTS.md, SOUL.md, picoclaw config.json injection), hot-reload via `POST /agent/:id/reload`, manifest versioning, Control Plane dashboard, Sidecar Execution, Managed Lifecycle.

---

## Planning docs

Full architecture and design decisions live in the Obsidian vault at:
`40_projects/43_picomaju/43.02_planning/`
