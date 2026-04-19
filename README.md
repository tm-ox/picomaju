<div align="center">
    <img src="web/static/logo-symbol.svg" alt="PicoMaju" width="170"/>
</div>

<br>

<div align="center">
    <img src="web/static/type.svg" alt="PicoMaju" width="170"/>
</div>

---

**Give your small business its own AI staff — without writing a single prompt.**

Picomaju lets you describe how your business operates — your tone, the tools you use, the roles that keep things running — and compiles that into autonomous AI agents ready to deploy. Set up your business profile through a simple guided flow. No IT department, no prompt engineering.

Built for small businesses in markets like Indonesia, where operations routinely span WhatsApp, Instagram, TikTok Shop, and Shopee at the same time, often with a lean team.

_Runs locally on Android via [picoclaw](https://github.com/sipeed/picoclaw) — a Go static binary under 10MB RAM. Also runs on any desktop OS._

---

## How it works

Everything in Picomaju maps to four building blocks:

**Values** are the rules your business runs by — tone of voice, escalation policies, what your staff should and shouldn't do. Written in plain text, organised by category.

**Tools** are the platforms you use — WhatsApp, Instagram, TikTok Shop, Shopee, payment processors, calendars. Connect them once; your staff uses them automatically.

**Tasks** are what your staff do — each task describes a job and the tools it needs. "Handle customer enquiries on WhatsApp." "Post daily updates to TikTok Shop."

**Staff** are your AI agents. Assign them tasks and values, and Picomaju compiles everything into a complete agent directive ready to run.

`Staff → Tasks → Tools + Values → Compiled Agent Directive`

---

## Stack

- **Go** — `chi/v5` router, `yaml.v3`, `a-h/templ`
- **templ** — server-side HTML templates compiled to Go
- **datastar** — SSE-based reactivity (validate previews stream into the page without a full reload)
- No Node.js, no build pipeline beyond `templ generate`

---

## First run

```sh
git clone <repo>
cd picomaju

# Download datastar.js from https://data-star.dev (place in web/static/)

DEV=1 go run .
```

Open `http://localhost:18800`. First visit runs a four-step onboarding (no sidebar):

1. Business name + data directory (defaults to `~/picomaju`)
2. Languages, timezone, and operating hours
3. First staff profile (optional — skip to do this later)
4. Tool picker — select the platforms your business uses; credentials configured afterwards under Tools

No env vars, no pre-created directories.

`DEV=1` serves static files from disk so CSS changes apply on browser refresh without rebuilding.

---

## Development

After editing any `.templ` file:

```sh
templ generate
go build ./...
```

---

## Configuration

Settings are managed in the app at `/settings`. The config file lives at the platform-appropriate location:

| Platform | Path                                                   |
| -------- | ------------------------------------------------------ |
| Linux    | `~/.config/picomaju/settings.json`                     |
| macOS    | `~/Library/Application Support/picomaju/settings.json` |
| Android  | per-app config dir via `os.UserConfigDir()`            |

**Environment variables** (advanced / managed deployments):

| Var               | Description                                       |
| ----------------- | ------------------------------------------------- |
| `PICOMAJU_CONFIG` | Override config file path                         |
| `DATA_DIR`        | Skip onboarding; use this data directory directly |
| `ADDR`            | Listen address (default `:18800`)                 |
| `DEV`             | Serve static files from disk                      |

---

## Project status

**Current:** Four-step onboarding (business info → languages/timezone/hours → first staff profile → tool picker), dashboard home screen, Values authoring + validation, Tools management (catalog integrations with per-type credential fields), Task definitions, Staff profiles, mobile-first UI (fixed bottom tab bar, floating action button, compact card rows, illustrated empty states), icon-strip collapsible sidebar, light/dark theming with icon toggle.

**Deferred:** Compiled output to multiple files (AGENTS.md, SOUL.md, picoclaw config.json tool injection), hot-reload into running agents, Control Plane dashboard, Sidecar Execution, Managed Lifecycle.

---

## Repo layout

```
main.go                  entry point
internal/
  settings/              config file store (business_name, data_dir, languages, timezone, hours)
  value/                 Value model, file store, validator, category defaults
  tool/                  Tool model + store (tools.json); Integration catalog (catalog.go)
  task/                  Task model + store (tasks.json)
  staff/                 Staff model + store (staff.json) — agent profiles
  api/                   HTTP handlers (HTML UI + SSE)
    ui.go                core page handlers + sidebarData helper
    ui_onboarding.go     onboarding step 2 (languages) + step 3 (first staff)
    router.go            all routes + setup gate middleware
    sse.go               SSEMergeFragment for datastar
web/
  templates/             templ components
    layout.templ         base shell (topnav, sidebar, bottom tabs, FAB, footer)
    dashboard.templ      home screen (centered logo symbol)
    sidebar.templ        contextual collapsible sidebar
    empty_state.templ    illustrated empty state component + section icons
    icons.templ          tool brand SVGs, nav/tab icons, edit/delete/theme icons
    setup.templ          four-step onboarding pages
    values/tools/tasks/staff.templ  section pages + forms
  static/                style.css, datastar.js (not committed), logo SVGs
AGENTS.md                instructions for AI agents working on this codebase
```
