# Picomaju

Mobile-first agent orchestrator for small business owners. Runs via **picoclaw** (Go, single binary, <10MB RAM, native APK) on Android. UI served at `:18800`.

Four pillars: Control Plane, Directive Compiler, Sidecar Execution, Managed Lifecycle.

## Product tiers

**Free — Configure & Preview**: full UI, directive compilation, workspace file preview. No LLM, no picoclaw.

**Paid — Execute**: picoclaw downloaded on first activation, agent chat, live LLM calls via proxy. Users never configure LLM keys.

**Pricing**: pay-as-you-go credits (primary); subscription plans (Starter / Pro) for power users.

## Payment stack

- **Stripe** — international (cards, Apple/Google Pay)
- **Xendit** — SE Asia (GoPay, OVO, DANA, QRIS; ID/PH/MY/TH/VN)

Webhooks delivered directly to the app (`/webhooks/stripe`, `/webhooks/xendit`). License written locally on payment confirmed. Gated behind env vars — no code changes needed when accounts are created.

## Dev workflow

```bash
task dev                     # hot reload — access at localhost:7331
templ generate && go build   # after manual .templ edits
tailwindcss -i ui/assets/css/input.css -o ui/assets/css/output.css
```

`datastar.js` must be placed in `ui/static/` manually — embedded into binary at build time.

## Env vars

| Var | Default | Description |
|-----|---------|-------------|
| `PICOMAJU_CONFIG` | `~/.config/picomaju/settings.json` | config file path |
| `DATA_DIR` | — | skip onboarding; use this data dir |
| `ADDR` | `:18800` | listen address |
| `DEV` | — | serve static from disk; enables dev-activate route |
| `STRIPE_SECRET_KEY` | — | Stripe secret key (`sk_live_…` or `sk_test_…`) |
| `STRIPE_WEBHOOK_SECRET` | — | Stripe webhook signing secret (`whsec_…`) |
| `XENDIT_API_KEY` | — | Xendit API key (`xnd_production_…`) |
| `XENDIT_WEBHOOK_TOKEN` | — | Xendit callback token (from Xendit dashboard) |
| `PICOMAJU_BASE_URL` | — | Public base URL for payment redirect URLs |
| `PICOCLAW_VERSION` | `0.2.8` | picoclaw release version to download on first activation |
| `ANTHROPIC_API_KEY` | — | Anthropic API key for LLM proxy (`sk-ant-…`) |

## Sub-docs

- `internal/CLAUDE.md` — packages, data models, API routes, security notes, test coverage
- `ui/CLAUDE.md` — shell, templates, sidebar, UI conventions

**Sub-doc rule:** All implementation detail — new packages, data model changes, route additions, UI components, test coverage updates, security notes — goes into the relevant sub-doc, not here. This file stays at orientation level only.

## Deferred

- **Manifest versioning**, Sidecar Execution, Managed Lifecycle, Overview section analytics
