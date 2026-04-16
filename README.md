<img src="web/static/logo-stack.svg" alt="PicoMaju" height="120"/>

Mobile-first agent orchestrator for small business owners. Define org-level directives as Values, assemble them into Staff profiles via Roles and Tools, and deploy autonomous agents — without prompt engineering.

Runs locally on Android via [picoclaw](https://github.com/sipeed/picoclaw) (Go static binary, <10MB RAM). Also runs on any desktop OS.

---

## Entity model

**Values** are org-level directives — tone, goals, policies. Authored as Markdown with YAML frontmatter, grouped by category (Core Values, Communication, Skills, Escalation, Custom).

**Tools** are capabilities an agent can use. Two kinds:
- **Integrations** — pre-defined catalog entries (WhatsApp Business, Telegram, Instagram, TikTok Shop, Shopee, Xendit, Midtrans, Google Calendar). Added via an onboarding picker or the Add Integration form; credentials configured per-integration.
- **Skills** — custom agent behaviours authored as SKILL.md documents. Define purpose, activation conditions, step-by-step procedure, and expected output.

**Roles** are task definitions. A role describes what an agent does and which Tools it uses.

**Staff** are agent profiles, composed of Roles + Values. Staff is the compile target — the entity that gets assembled into an agent directive.

Relationship chain: `Staff → Roles → Tools + Values (by category or individual) → Compiled Agent Directive`

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

Open `http://localhost:18800`. First visit runs a two-step onboarding (no sidebar):

1. Business name + data directory (defaults to `~/picomaju`)
2. Integration picker — select the platforms your business uses; credentials can be filled in afterwards under Tools

That's it — no env vars, no pre-created directories.

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

Settings (business name, data directory) are managed in the app at `/settings`.

The config file lives at the platform-appropriate location:

| Platform | Path |
|----------|------|
| Linux | `~/.config/picomaju/settings.json` |
| macOS | `~/Library/Application Support/picomaju/settings.json` |
| Android | per-app config dir via `os.UserConfigDir()` |

**Environment variables** (advanced / managed deployments):

| Var | Description |
|-----|-------------|
| `PICOMAJU_CONFIG` | Override config file path |
| `DATA_DIR` | Skip onboarding; use this data directory directly |
| `ADDR` | Listen address (default `:18800`) |
| `DEV` | Serve static files from disk |

---

## Project status

**Current:** Two-step onboarding with integration catalog picker, Values authoring + validation, Tools management (integrations with per-type credential fields + skills with SKILL.md editor), Role definitions, Staff profiles, top-nav + contextual sidebars (hidden during onboarding), pick-chip selection UI, light/dark theming, settings.

**Deferred:** Compiled output to multiple files (AGENTS.md, SOUL.md, picoclaw config.json tool injection), hot-reload into running agents, Control Plane dashboard, Sidecar Execution, Managed Lifecycle.

---

## Repo layout

```
main.go                  entry point
internal/
  settings/              config file store
  value/                 Value model, file store, validator, category defaults
  tool/                  Tool model + store (tools.json); Integration catalog (catalog.go)
  role/                  Role model + store (roles.json) — task definitions
  staff/                 Staff model + store (staff.json) — agent profiles
  api/                   HTTP handlers (HTML UI + SSE)
web/
  templates/             templ components
  static/                style.css, datastar.js (not committed)
AGENTS.md                instructions for AI agents working on this codebase
```
