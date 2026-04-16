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
    role/
      store.go                   — Role type (id, label, description, tools[]) + CRUD on roles.json
    staff/
      store.go                   — Staff type (id, label, roles[], value_categories[], values[]) + CRUD on staff.json
    api/
      router.go                  — all routes wired here; setup gate middleware
      ui.go                      — HTML + SSE handlers; uiHandler with mutex-guarded store init
      sse.go                     — SSEMergeFragment() for datastar
      helpers.go                 — jsonOK / jsonErr
  web/
    templates/
      layout.templ               — base HTML shell; top nav (business + avatar | section links | theme + settings)
      sidebar.templ              — contextual sidebar component (switches per active section)
      values.templ               — Value list page, Value form, ValidationFragment
      tools.templ                — Tool list, NewIntegrationPage (radio picker), ToolFormPage (edit integration), SkillFormPage (SKILL.md editor)
      roles.templ                — Role list page, Role form (with tool picker)
      staff.templ                — Staff list page, Staff form (role + value picker)
      settings.templ             — Settings page (business info + data dir)
      setup.templ                — Two-step onboarding: SetupPage + IntegrationsPage (catalog picker)
      helpers.go                 — SidebarData type, includesStr()
    static/
      style.css                  — CSS custom properties; light + dark mode; 16px REM base
      datastar.js                — MUST be downloaded manually from data-star.dev releases
```

---

## First run / onboarding

On first launch, if no data directory is configured, **all routes redirect to `/setup`**. Onboarding is two steps and renders without a sidebar (minimal header, full-width layout):

1. **`/setup`** — Business Name + Data Directory (pre-filled `~/picomaju`)
2. **`/setup/integrations`** — Integration picker: select from the catalog to auto-create Tool entries; credentials configured later in Tools. Both steps exempt from the setup gate middleware.

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
No `trigger` field — Values are directives, not event-driven rules.

### Tool (`<data_dir>/tools.json`)
Capabilities available to a Role. Two kinds:

- **Integration** — type matches a catalog entry (e.g. `whatsapp`, `telegram`, `shopee`). Config holds credentials keyed by `ConfigField.Key`.
- **Skill** — type `skill`. Config holds a single `content` key containing a SKILL.md markdown document.

```json
{ "tools": [
  { "id": "whatsapp", "label": "WhatsApp Business", "type": "whatsapp",
    "config": { "phone_number_id": "...", "access_token": "..." } },
  { "id": "handle_escalation", "label": "Handle Escalation", "type": "skill",
    "config": { "content": "# Handle Escalation\n\n## Purpose\n..." } }
]}
```

### Role (`<data_dir>/roles.json`)
Task definition. Describes what the agent does and which Tools it uses.

```json
{ "roles": [
  {
    "id": "manage_social_media",
    "label": "Manage Social Media",
    "description": "Post updates and respond to comments",
    "tools": ["email_sendgrid"]
  }
]}
```

### Staff (`<data_dir>/staff.json`)
Agent profile. Composed of Roles + Values. The compile target.

```json
{ "staff": [
  {
    "id": "support_agent",
    "label": "Support Agent",
    "roles": ["manage_social_media"],
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

## Compiler

**Deferred.** Output will be multiple files per Staff member: `AGENTS.md`, `SOUL.md`, tool injection into picoclaw `config.json`. Pipeline logic exists as a stub (`value.DirectiveEntry`) but serialization is not implemented.

---

## UI routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/setup` | onboarding step 1 — business name + data dir |
| POST | `/setup` | save step 1 → redirect `/setup/integrations` |
| GET | `/setup/integrations` | onboarding step 2 — integration picker |
| POST | `/setup/integrations` | create selected tools → redirect `/values` |
| GET | `/` | redirect → `/values` |
| GET | `/values` | Value list (filterable by `?cat=<id>`) |
| GET | `/values/new` | Value form (new) |
| POST | `/values` | create Value → redirect `/values` |
| GET | `/values/:id/edit` | Value form (edit) |
| POST | `/values/:id` | update Value → redirect `/values` |
| POST | `/values/:id/delete` | delete Value → redirect `/values` |
| POST | `/values/:id/validate-stream` | SSE: `ValidationFragment` |
| GET | `/tools` | Tool list (integrations + skills) |
| GET | `/tools/new` | Add Integration — catalog radio card picker |
| POST | `/tools` | create Integration from `integration_id` → redirect to edit |
| GET | `/tools/new/skill` | New Skill — SKILL.md editor |
| POST | `/tools/skill` | create Skill → redirect to edit |
| GET | `/tools/:id/edit` | Edit Integration (credential fields) or Edit Skill (SKILL.md editor) |
| POST | `/tools/:id` | update Tool/Skill → redirect `/tools` |
| POST | `/tools/:id/delete` | delete Tool → redirect `/tools` |
| GET | `/roles` | Role list |
| GET | `/roles/new` | Role form (new) |
| POST | `/roles` | create Role → redirect `/roles` |
| GET | `/roles/:id/edit` | Role form (edit) |
| POST | `/roles/:id` | update Role → redirect `/roles` |
| POST | `/roles/:id/delete` | delete Role → redirect `/roles` |
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
- Left: avatar placeholder + business name
- Center: Values | Tools | Roles | Staff (section links, `.active` on current section)
- Right: theme toggle + settings gear

**Sidebar** (contextual, collapsible — hidden entirely during onboarding):
- Collapsed state: `2.25rem` wide (no icons yet — content hidden via `opacity: 0`)
- Content switches per `ActiveSection` in `SidebarData`:
  - `values` → category filter links with counts + New Value
  - `tools` → **Integrations** section (catalog-type tools) + **Skills** section (type `skill`) + Add Integration / New Skill links
  - `roles` → role list + New Role
  - `staff` → staff list + New Staff

**Footer**: "Powered by PicoMaju" branding strip.

`SidebarData` (in `web/templates/helpers.go`) is built by `uiHandler.sidebarData(r, section)` and passed to every page render.

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

**Done:** Two-step onboarding with integration picker, settings, Values authoring + validation, Tools CRUD (integrations + skills), Integration catalog (WhatsApp, Telegram, Instagram, TikTok Shop, Shopee, Xendit, Midtrans, Google Calendar), SKILL.md editor, Role definitions with tool picker, Staff profiles with role + value picker, top-nav + contextual sidebar (hidden during onboarding), light/dark theming, pick-chip selection UX (no visible checkboxes).

**Deferred:** Compiler output (AGENTS.md, SOUL.md, picoclaw config.json injection), hot-reload via `POST /agent/:id/reload`, manifest versioning, Control Plane dashboard, Sidecar Execution, Managed Lifecycle.

---

## Planning docs

Full architecture and design decisions live in the Obsidian vault at:
`40_projects/43_picomaju/43.02_planning/`
