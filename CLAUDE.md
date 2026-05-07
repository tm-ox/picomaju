# Picomaju

Mobile-first agent orchestrator for small business owners. Runs via **picoclaw** (Go, single binary, <10MB RAM, native APK) on Android. UI served at `:18800`.

Four pillars: Control Plane, Directive Compiler, Sidecar Execution, Managed Lifecycle.

## Status

**Implemented:** Full UI (staff / values / tools / tasks / chat shell) · Directive Compiler · License store · Plan & Credits page · Payment infrastructure (Stripe + Xendit) · Chat activation gate · Compile/Activate split · Picoclaw lifecycle (download + subprocess management) · LLM proxy (Anthropic passthrough + credit metering + rate limiting) · Subscription renewal verification (Stripe)

**Security hardened (2026-05-07):** Xendit webhook constant-time compare · credits validation · URL-encoded error redirects · store write error handling · value/task ID path traversal prevention · license mutex (atomic credit deduction) · form field length caps · absolute path enforcement for data dir · compiler warnings for unresolved tool refs · onboarding role field wired

**Next:** Control Plane dashboard

## Product tiers

**Free — Configure & Preview**: full UI, directive compilation, workspace file preview. No LLM, no picoclaw.

**Paid — Execute**: picoclaw downloaded on first activation, agent chat, live LLM calls via proxy. Users never configure LLM keys.

**Pricing**: pay-as-you-go credits (primary); subscription plans (Starter / Pro) for power users.

## Payment stack

- **Stripe** — international (cards, Apple/Google Pay)
- **Xendit** — SE Asia (GoPay, OVO, DANA, QRIS; ID/PH/MY/TH/VN)

Webhooks delivered directly to the app (`/webhooks/stripe`, `/webhooks/xendit`). On payment confirmed, `license.json` is written locally. No separate backend required for v1. LLM proxy (future) will require a backend for metering.

Payment infrastructure is fully implemented and gated behind env vars. No code changes needed when accounts are created — just set the env vars.

## Picoclaw integration

Not bundled, not downloaded during onboarding. Fetched from GitHub releases on first activation (`picoclaw-android-universal.zip` or platform equivalent), extracted to `{dataDir}/bin/picoclaw`, managed as a subprocess thereafter. User has zero visibility. Picomaju owns full `config.json` generation.

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
| `PICOCLAW_VERSION` | `0.1.0` | picoclaw release version to download on first activation |
| `ANTHROPIC_API_KEY` | — | Anthropic API key for LLM proxy (`sk-ant-…`) |

## Sub-docs

- `internal/CLAUDE.md` — packages, data models, API routes
- `ui/CLAUDE.md` — shell, templates, sidebar, UI conventions

## Deferred

- **Subscription renewal** — on expiry, re-verify against payment provider; currently 35-day local expiry safety net only
- **Manifest versioning**, Control Plane dashboard, Sidecar Execution, Managed Lifecycle, Overview section analytics
