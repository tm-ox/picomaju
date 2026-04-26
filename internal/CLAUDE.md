# internal/

## Packages

```
settings/store.go   — Settings{business_name, business_details, data_dir, languages[], timezone, hours}; file-backed JSON
value/model.go      — Value, DirectiveEntry, ValidationResult, ValidationError
value/store.go      — CRUD on <data_dir>/values/<id>.md (YAML frontmatter + body)
value/validator.go  — required field check; priority clamp [0–100]
value/category.go   — Category + DefaultCategories (built-in, no disk store)
tool/store.go       — Tool{id, label, type, config map[string]any}; CRUD on tools.json
tool/catalog.go     — Integration catalog: 8 entries; CatalogByCategory/ID/Type helpers
task/store.go       — Task{id, label, description, tools[]}; CRUD on tasks.json
staff/store.go      — Staff{id, label, description, active, icon, tasks[], value_categories[], values[]}; CRUD on staff.json
api/router.go       — all routes; setup gate middleware
api/ui.go           — HTML + SSE handlers; uiHandler{mutex-guarded stores}; navData() helper
api/ui_onboarding.go — completeWelcome; firstStaffPage/completeFirstStaff; slugify
api/sse.go          — SSEMergeFragment() for datastar
api/helpers.go      — jsonOK / jsonErr
```

## Data model

**Settings** — `~/.config/picomaju/settings.json`. `languages`, `timezone`, `hours` are omitempty.

**Value** — `<data_dir>/values/<id>.md`. Required: `id`, `title`, `version`, `priority` (0–100), `category`. Body is raw markdown after frontmatter.

**Tool** — `<data_dir>/tools.json`. `type` matches a catalog entry. `config` holds credentials keyed by `ConfigField.Key`.

**Task** — `<data_dir>/tasks.json`. `tools` is a list of tool IDs.

**Staff** — `<data_dir>/staff.json`. `value_categories` → all values in those categories; `values` → individual value IDs on top. `icon` is a Lucide icon name (28 options) or `""` for initials fallback.

## Value categories

| ID | Label |
|----|-------|
| `core_values` | Core Values |
| `communication` | Communication |
| `skills` | Skills |
| `escalation` | Escalation |
| `custom` | Custom |

## Integration catalog

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

## Routes

Mutations: form POST + redirect. SSE validate uses datastar. Setup gate exempts `/welcome`, `/setup*`, `/static/*`.

| Method | Path | Handler |
|--------|------|---------|
| GET | `/` | staff dashboard |
| GET/POST | `/welcome` | language picker → `/setup` |
| GET/POST | `/setup` | step 1: business name, data dir, tz, hours |
| GET/POST | `/setup/first-staff` | step 2: first staff profile |
| GET/POST | `/setup/integrations` | step 3: tool picker → `/values` |
| GET | `/values[?cat=]` | value list, filtered by category |
| GET/POST | `/values/new` `/values` | create value |
| GET/POST | `/values/:id/edit` `/values/:id` | edit value |
| POST | `/values/:id/delete` | delete |
| POST | `/values/:id/validate-stream` | SSE ValidationFragment |
| GET | `/tools[?cat=]` | tool list, filtered by catalog category |
| GET/POST | `/tools/new` `/tools` | catalog picker → create → redirect edit |
| GET/POST | `/tools/:id/edit` `/tools/:id` | edit credentials |
| POST | `/tools/:id/delete` | delete |
| GET | `/tasks[?tool_cat=]` | task list, filtered by tool catalog category |
| GET/POST | `/tasks/new` `/tasks` | create task |
| GET/POST | `/tasks/:id/edit` `/tasks/:id` | edit task |
| POST | `/tasks/:id/delete` | delete |
| GET | `/staff/new` | new staff form |
| POST | `/staff` | create → `/staff/:id` |
| GET | `/staff/:id[?s=overview\|profile\|values\|tools\|tasks]` | detail page |
| POST | `/staff/:id/profile` | update label/description/icon/active |
| POST | `/staff/:id/tasks` | update task assignments |
| POST | `/staff/:id/values` | update value/category assignments |
| POST | `/staff/:id/delete` | delete → `/` |
| GET/POST | `/settings` | settings page |
| GET | `/static/*` | embedded static assets |
