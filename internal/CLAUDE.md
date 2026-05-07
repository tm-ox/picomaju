# internal/

## Packages

```
settings/store.go    — Settings{business_name, business_details, data_dir, languages[], timezone, hours}; file-backed JSON at PICOMAJU_CONFIG
license/store.go     — License{active, plan, credits_remaining, token, expires_at}; IsActive(); DeductCredit(); file-backed JSON at {dataDir}/license.json
payment/config.go    — Config from env vars; StripeConfigured()/XenditConfigured()
payment/pricing.go   — CreditPacks (100/500/1500); Plans (starter/pro) with USD+IDR amounts; CreditPackByID/PlanByID
payment/stripe.go    — StripeCheckoutURL(cfg, packID, planID) → Stripe Checkout session URL
payment/xendit.go    — XenditCheckoutURL(cfg, packID, planID) → Xendit invoice URL (IDR)
compiler/compiler.go — Compile(Input) Output; builds AGENT.md, SOUL.md, USER.md
compiler/write.go    — Write(Output, workspaceDir); InjectConfig(configPath, AgentEntry)
value/model.go       — Value, DirectiveEntry, ValidationResult, ValidationError
value/store.go       — CRUD on <data_dir>/values/<id>.md (YAML frontmatter + body)
value/validator.go   — required field check; priority clamp [0–100]
value/category.go    — Category + DefaultCategories (built-in, no disk store)
tool/store.go        — Tool{id, label, type, config map[string]any}; CRUD on tools.json
tool/catalog.go      — Integration catalog: 8 entries; CatalogByCategory/ID/Type helpers
task/store.go        — Task{id, label, description, tools[]}; CRUD on tasks.json
staff/store.go       — Staff{id, label, description, active, icon, tasks[], value_categories[], values[]}; CRUD on staff.json
chat/store.go        — Chat{id, staff_id, title, created_at, messages[]}, Message{role, content, ts}; CRUD on chats.json; ListByStaff(staffID)
api/router.go        — all routes; setup gate middleware (exempts /welcome, /setup*, /static/*, /webhooks/*)
api/ui.go            — HTML + SSE handlers; uiHandler{stores + license + settings}; navData() helper
api/ui_onboarding.go — completeWelcome; firstStaffPage/completeFirstStaff; slugify
api/webhooks.go      — stripeWebhook; xenditWebhook; activateFromStripe/Xendit → license.json
api/sse.go           — SSEMergeFragment() for datastar
api/helpers.go       — jsonOK / jsonErr
```

## Data models

**Settings** — `~/.config/picomaju/settings.json` (overridable via `PICOMAJU_CONFIG`).

**License** — `{dataDir}/license.json`.
- `plan`: `""` (free) | `"credits"` | `"starter"` | `"pro"`
- `expires_at`: unix timestamp; `0` = no expiry (credits plan)
- `IsActive()`: active=true AND (credits>0 if credits plan) AND not expired
- Credits are additive — top-ups increment `credits_remaining`
- Subscription plans get 35-day local expiry as safety net

**Value** — `<data_dir>/values/<id>.md`. YAML frontmatter + markdown body. Required: `id`, `title`, `version`, `priority` (0–100), `category`.

**Tool** — `<data_dir>/tools.json`. `type` matches catalog entry. `config` holds credentials keyed by `ConfigField.Key`.

**Task** — `<data_dir>/tasks.json`. `tools` is a list of tool IDs.

**Staff** — `<data_dir>/staff.json`. `value_categories` → bulk inclusion; `values` → individual IDs. `icon` is a Lucide icon name (28 options) or `""` for initials fallback.

**Chat** — `<data_dir>/chats.json`. `title` auto-set from first message (truncated 40 chars). `messages[]{role, content, ts}` — role is `"user"` or `"assistant"`. ID is hex-encoded `time.Now().UnixNano()`.

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

## Payment pricing

| ID | Credits | USD | IDR |
|----|---------|-----|-----|
| `credits_100` | 100 | $5 | 80k |
| `credits_500` | 500 | $20 | 320k |
| `credits_1500` | 1,500 | $50 | 790k |
| `starter` (plan) | 300/mo | $12/mo | 190k/mo |
| `pro` (plan) | unlimited | $29/mo | 460k/mo |

Stripe plan activation requires `Plan.StripePriceID` to be set (populated when products are created in Stripe dashboard).

## Routes

Setup gate: redirect to `/welcome` until data dir configured. Exempt: `/welcome`, `/setup*`, `/static/*`, `/ui/*`, `/webhooks/*`.

| Method | Path | Handler |
|--------|------|---------|
| GET | `/` | staff dashboard |
| GET/POST | `/welcome` | language picker → `/setup` |
| GET/POST | `/setup` | step 1: business name, data dir, tz, hours |
| GET/POST | `/setup/first-staff` | step 2: first staff profile |
| GET/POST | `/setup/integrations` | step 3: tool picker → `/values` |
| GET | `/values[?cat=]` | value list |
| GET/POST | `/values/new` `/values` | create value |
| GET/POST | `/values/:id/edit` `/values/:id` | edit value |
| POST | `/values/:id/delete` | delete |
| POST | `/values/:id/validate-stream` | SSE ValidationFragment |
| GET | `/tools[?cat=]` | tool list |
| GET/POST | `/tools/new` `/tools` | catalog picker → create |
| GET/POST | `/tools/:id/edit` `/tools/:id` | edit credentials |
| POST | `/tools/:id/delete` | delete |
| GET | `/tasks[?tool_cat=]` | task list |
| GET/POST | `/tasks/new` `/tasks` | create task |
| GET/POST | `/tasks/:id/edit` `/tasks/:id` | edit task |
| POST | `/tasks/:id/delete` | delete |
| GET | `/staff/new` | new staff form |
| POST | `/staff` | create → `/staff/:id` |
| GET | `/staff/:id[?s=overview\|profile\|values\|tools\|tasks]` | detail page (`?compiled=1`, `?err=`) |
| POST | `/staff/:id/profile` | update profile |
| POST | `/staff/:id/tasks` | update task assignments |
| POST | `/staff/:id/values` | update value/category assignments |
| POST | `/staff/:id/delete` | delete → `/` |
| POST | `/staff/:id/compile` | compile workspace files → `?compiled=1` |
| POST | `/staff/:id/chats` | create chat → chat page |
| GET | `/staff/:id/chats/:chatId` | chat page |
| POST | `/staff/:id/chats/:chatId/messages` | append message (redirects `/license` if not active) |
| POST | `/staff/:id/chats/:chatId/rename` | rename |
| POST | `/staff/:id/chats/:chatId/delete` | delete → `/staff/:id` |
| GET | `/license` | plan & credits (`?activated=1`, `?err=`) |
| GET | `/license/checkout` | → Stripe/Xendit checkout (`?pkg=`, `?plan=`, `?provider=stripe\|xendit`) |
| GET | `/license/checkout/success` | post-payment landing → `/license?activated=1` |
| POST | `/license/activate-dev` | DEV env only — instant test license |
| POST | `/webhooks/stripe` | Stripe webhook (sig verify → license update) |
| POST | `/webhooks/xendit` | Xendit webhook (token verify → license update) |
| GET/POST | `/settings` | settings page |
| GET | `/static/*` | embedded static assets |
