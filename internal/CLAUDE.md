# internal/

## Packages

```
settings/
  store.go        — Settings type (business_name, business_details, data_dir,
                    languages[], timezone, hours) + file-backed store
value/
  model.go        — Value, DirectiveEntry, ValidationResult, ValidationError types
  store.go        — file-based CRUD (.md + YAML frontmatter)
  validator.go    — required field check, priority clamp [0–100]
  category.go     — Category type + DefaultCategories (built-in, no separate store)
tool/
  store.go        — Tool type (id, label, type, config) + CRUD on tools.json
  catalog.go      — Integration catalog: 8 entries, CatalogByCategory/ID/Type
task/
  store.go        — Task type (id, label, description, tools[]) + CRUD on tasks.json
staff/
  store.go        — Staff type (id, label, tasks[], value_categories[], values[]) + CRUD on staff.json
api/
  router.go       — all routes wired here; setup gate middleware (allows /welcome + setup paths + /static/*)
  ui.go           — HTML + SSE handlers; uiHandler with mutex-guarded store init; navData() helper
  ui_onboarding.go — completeWelcome; firstStaffPage/completeFirstStaff; slugify helper
  sse.go          — SSEMergeFragment() for datastar
  helpers.go      — jsonOK / jsonErr
```

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
Settings file location is platform-aware (`os.UserConfigDir()/picomaju/settings.json`). On Linux: `~/.config/picomaju/settings.json`.

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

## Value categories

Built-in constants in `value/category.go` — no separate file on disk.

| ID | Label |
|----|-------|
| `core_values` | Core Values |
| `communication` | Communication |
| `skills` | Skills |
| `escalation` | Escalation |
| `custom` | Custom |

## Integration catalog

8 entries in `tool/catalog.go`:

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

## Compiler

**Deferred.** Output will be multiple files per Staff member: `AGENTS.md`, `SOUL.md`, tool injection into picoclaw `config.json`. Pipeline logic exists as a stub (`value.DirectiveEntry`) but serialization is not implemented.

## UI routes

Mutations use standard HTML form POST + redirect (no JS required for CRUD). Validate uses datastar SSE.

The setup gate middleware allows `/welcome`, `/setup`, `/setup/first-staff`, `/setup/integrations`, and `/static/*` without redirect. All other routes redirect to `/welcome` when no data directory is configured.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | dashboard (home screen) |
| GET | `/welcome` | welcome screen — language picker |
| POST | `/welcome` | save language → redirect `/setup` |
| GET | `/setup` | onboarding step 1 — business name + data dir + timezone + hours |
| POST | `/setup` | save step 1 → redirect `/setup/first-staff` |
| GET | `/setup/first-staff` | onboarding step 2 — first staff profile |
| POST | `/setup/first-staff` | create staff → redirect `/setup/integrations` |
| GET | `/setup/integrations` | onboarding step 3 — tool picker |
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
