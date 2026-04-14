# Picomaju

Mobile-first agent orchestrator for small business owners. Define business rules as SOPs, assemble them into role-based policy sets, and deploy them to autonomous agents — without prompt engineering.

Runs locally on Android via [picoclaw](https://github.com/sipeed/picoclaw) (Go static binary, <10MB RAM). Also runs on any desktop OS.

---

## What it does

**SOPs (Standard Operating Procedures)** are atomic business rules — a trigger, a priority, and a natural-language instruction. You author them through a structured form; they're stored as Markdown with YAML frontmatter.

**Roles** are job descriptions for agents. You hire a role by selecting which SOP categories it covers (bulk) and any individual SOPs on top. The compiler assembles the selected SOPs into a policy set, validates them, and sorts by priority.

**Categories** group SOPs by domain. The starter set is: Core Values, Communication, Skills, Escalation, Custom.

---

## Stack

- **Go** — `chi/v5` router, `yaml.v3`, `a-h/templ`
- **templ** — server-side HTML templates compiled to Go
- **datastar** — SSE-based reactivity (validate/compile previews stream into the page without a full reload)
- No Node.js, no build pipeline beyond `templ generate`

---

## First run

```sh
git clone <repo>
cd picomaju

# Download datastar.js from https://data-star.dev (place in web/static/)

DEV=1 go run .
```

Open `http://localhost:18800`. On first visit you'll be asked for a business name and a data directory (defaults to `~/picomaju`). That's it — no env vars, no pre-created directories.

`DEV=1` serves static files from disk so CSS changes apply on browser refresh without rebuilding.

---

## Development

After editing any `.templ` file:

```sh
templ generate
go build ./...
go test ./...
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

**v1 — built:** SOP authoring, category management, role definitions, in-memory compilation with preview, sidebar navigation with category filtering, onboarding, settings.

**v2 — not started:** JSON manifest output, hot-reload into running agents, Control Plane dashboard, Sidecar Execution, Managed Lifecycle.

---

## Repo layout

```
main.go                  entry point
internal/
  settings/              config file store
  sop/                   SOP model, file store, validator, compiler
  category/              category model + store
  role/                  role model + store
  api/                   HTTP handlers (JSON API + HTML UI + SSE)
web/
  templates/             templ components
  static/                style.css, datastar.js (not committed)
AGENTS.md                instructions for AI agents working on this codebase
```
