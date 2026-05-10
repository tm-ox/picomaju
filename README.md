<div align="center">
    <img src="ui/static/logo-symbol.svg" alt="PicoMaju" width="170"/>
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

# Download datastar.js from https://data-star.dev (place in ui/static/)
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
go build .
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

| Var                     | Description                                                        |
| ----------------------- | ------------------------------------------------------------------ |
| `PICOMAJU_CONFIG`       | Override config file path                                          |
| `DATA_DIR`              | Skip onboarding; use this data directory directly                  |
| `ADDR`                  | Listen address (default `:18800`)                                  |
| `DEV`                   | Serve static files from disk; enables `/license/activate-dev`      |
| `STRIPE_SECRET_KEY`     | Stripe secret key (`sk_live_…` or `sk_test_…`)                     |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook signing secret (`whsec_…`)                          |
| `XENDIT_API_KEY`        | Xendit API key (`xnd_production_…`)                                |
| `XENDIT_WEBHOOK_TOKEN`  | Xendit callback token                                              |
| `PICOMAJU_BASE_URL`     | Public base URL for payment redirect URLs                          |
| `PICOCLAW_VERSION`      | picoclaw release to download on first activation (default `0.1.0`) |
| `ANTHROPIC_API_KEY`     | Anthropic API key for LLM proxy (`sk-ant-…`)                       |

---

## Project status

**Implemented:** Full UI (home dashboard · staff · values · tools · tasks · chat · settings · users · login · profile) · Directive Compiler · License store · Plan & Credits page · Payment infrastructure (Stripe + Xendit) · Chat activation gate · Picoclaw lifecycle (download + subprocess management) · LLM proxy (Anthropic passthrough + credit metering + rate limiting) · User system (PIN auth · roles · session · user management · profile). Component workshop at `/ui/workshop`.

**Next:** Home dashboard — wire real data (agent count, active agents, messages today, credits); Activity tab (chat history); Agents tab (live picoclaw status).

**Deferred:** Sidecar Execution, Managed Lifecycle, manifest versioning.

---

## Repo layout

```
main.go                  entry point; serves /static/* and all app routes
Taskfile.yml             task dev (hot reload), task build:css
.templui.json            templui CLI config
internal/
  settings/              config file store
  value/ tool/ task/ staff/ chat/ user/  domain models + file stores
  license/ payment/      license store + Stripe/Xendit checkout
  picoclaw/              binary lifecycle manager + config writer
  llmproxy/              Anthropic passthrough proxy with credit metering
  api/                   HTTP handlers (HTML UI + SSE + webhooks)
    router.go            all routes + setup gate + auth gate middleware
    ui.go                core page handlers
    ui_onboarding.go     onboarding step handlers
    ui_users.go          login/logout, user CRUD, profile
    webhooks.go          Stripe + Xendit webhook handlers
ui/                      frontend (templui + Tailwind CSS v4)
  static/                logo SVGs, datastar.js (not committed — place manually)
  assets/css/input.css   Tailwind @theme config + brand tokens
  assets/css/output.css  generated; gitignored
  assets/js/             templui component JS
  components/            templui components (CLI workflow)
  utils/templui.go       TwMerge + script helpers
  templates/             all page templates + nav shell + shared primitives
design/
  tokens.json            W3C DTCG design tokens (canonical)
```
