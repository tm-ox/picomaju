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
- **templui** — component library (CLI workflow; source in `ui/components/`)
- **go-templ-lucide-icons** — Lucide icon set as typed templ components
- **Tailwind CSS v4** — CSS-first config via `@theme`; standalone CLI binary (no Node.js)
- No Node.js, no npm

---

## First run

```sh
git clone <repo>
cd picomaju

# Download datastar.js from https://data-star.dev (place in web/static/)
# Build Tailwind CSS output
tailwindcss -i ./ui/assets/css/input.css -o ./ui/assets/css/output.css

DEV=1 go run .
```

Open `http://localhost:18800`. First visit runs a welcome screen then three onboarding steps (no sidebar):

1. Welcome — choose your language (English / Bahasa Indonesia)
2. Business name, data directory, timezone, and operating hours
3. First staff profile (optional — skip to do this later)
4. Tool picker — select the platforms your business uses; credentials configured afterwards under Tools

No env vars, no pre-created directories.

---

## Development

Hot reload (recommended):

```sh
task dev
# access via http://localhost:7331
```

Saving any `.templ` or `.go` file triggers auto-rebuild and browser reload. CSS changes rebuild automatically; manual browser refresh applies them.

Manual workflow:

```sh
templ generate
go build ./internal/... ./web/... .
tailwindcss -i ./ui/assets/css/input.css -o ./ui/assets/css/output.css
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

**Current:** Full migration to new frontend (`/ui`, templui + Tailwind CSS v4) complete. All sections live: onboarding, dashboard, values (with SSE validation), tools, tasks, staff, settings. Mobile-first shell with sticky top nav, bottom tab bar, per-section FAB. Component workshop at `/ui/workshop`.

**Deferred:** Compiled output (AGENTS.md, SOUL.md, picoclaw config.json injection), hot-reload into running agents, Control Plane dashboard, Sidecar Execution, Managed Lifecycle.

---

## Repo layout

```
main.go                  entry point; mounts both /web and /ui routes
Taskfile.yml             task dev (hot reload), task build:css
.templui.json            templui CLI config
internal/
  settings/              config file store
  value/ tool/ task/ staff/  domain models + file stores
  api/                   HTTP handlers (HTML UI + SSE)
    router.go            all routes + setup gate middleware
    ui.go                core page handlers
    ui_onboarding.go     onboarding step handlers
web/                     legacy frontend (preserved for reference; not active)
  templates/             templ components (all section pages)
  static/                style.css, app.js, datastar.js (not committed), logo SVGs
ui/                      active frontend (templui + Tailwind CSS v4)
  assets/css/input.css   Tailwind @theme config + brand tokens (oklch)
  assets/css/output.css  generated; gitignored
  assets/js/             templui component JS
  components/            templui components (CLI workflow)
  utils/templui.go       TwMerge + script helpers
  templates/             all page templates + nav shell + shared primitives
design/
  tokens.json            W3C DTCG design tokens (canonical)
```
