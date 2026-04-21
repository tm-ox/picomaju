# Picomaju UI Review & Design System Extraction Plan

**Date:** 2026-04-21
**Scope:** `web/templates/*.templ` (11 files, ~1,420 lines), `web/static/style.css` (1,365 lines), `web/templates/helpers.go`
**Goal:** Code quality review + design system extraction planning for Penpot/Pencil reuse

---

## Status tracker

- [x] C1 — Fix `--color-accent-subtle` undefined token
- [x] M9 — Add `:focus-visible` styles
- [x] M1-M3 — Add aria-labels to icon controls
- [x] M4 — Move inline scripts to `/static/app.js`
- [x] M5-M7 — Extract shared templ components (actions, form helpers, table)
- [x] M8 — Replace raw rgba values with tokens
- [x] Phase 1 — Normalize CSS spacing/type/weight/z-index scales
- [x] Phase 2 — Author `design/tokens.json` (DTCG format)
- [ ] Phase 3 — Import tokens into Penpot/Pencil
- [ ] Phase 4 — Build top-10 components in design tool

---

## Code review findings

### CRITICAL

#### C1 — Undefined `--color-accent-subtle` breaks pick-chip selected state
**File:** `web/static/style.css:771`
```css
.pick-chip:has(.pick-toggle:checked) {
    border-color: var(--color-accent);
    background: var(--color-accent-subtle);   /* NOT DEFINED — falls back to transparent */
    color: var(--color-text);
}
```
`--color-accent-subtle` is never declared in `:root` or `[data-theme="dark"]`. Only `--color-accent-soft` exists (`style.css:18, 82`). Selected pick-chips appear unselected (transparent bg) on the tasks tool picker and staff task/value pickers.

`.integration-card:has(.integration-toggle:checked)` at `style.css:998` correctly uses `--color-accent-soft` — the two checked-state patterns disagree.

**Fix:** Change `var(--color-accent-subtle)` → `var(--color-accent-soft)` at line 771.

---

### MAJOR

#### M1 — Theme toggle button has no accessible name
**File:** `web/templates/layout.templ:56-67`
Icon-only button. `title="Toggle theme"` is not exposed by some AT on buttons. Inner SVGs use `aria-hidden="true"` (correct), so AT reads an empty name. No `aria-pressed` to reflect state.

**Fix:** Add `aria-label="Toggle dark mode"` and dynamic `aria-pressed` (set/updated by the onclick handler).

#### M2 — Settings gear link has no accessible name
**File:** `web/templates/layout.templ:68`
```html
<a href="/settings" class="topnav-settings" title="Settings">&#9881;</a>
```
Glyph enters the accessibility tree with no semantic context. `title` is not reliable for AT on anchors.

**Fix:** Add `aria-label="Settings"` or swap glyph for an `aria-hidden` SVG + `aria-label`.

#### M3 — Sidebar collapse button label/state never updates
**File:** `web/templates/sidebar.templ:25-33`
- Hard-coded `title="Collapse sidebar"` lies when already collapsed.
- No `aria-expanded`, no `aria-controls`.
- Accessible name is the raw chevron character `‹`.
- Inline `onclick` toggles CSS class but never updates `title` or `aria-expanded`.

**Fix:** Add `aria-controls="sidebar"`, `aria-expanded` from persisted state, update both inside the handler.

#### M4 — Inline scripts block CSP + theme script is duplicated
**Files:** `web/templates/layout.templ:16-22, 83-90`; `web/templates/setup.templ:14-20`
Three inline `<script>` blocks. The pre-paint theme script is duplicated verbatim between `layout.templ` and `setup.templ`. Any strict CSP blocks all three (no nonce, no hash). Sidebar-restore at `layout.templ:83` is racy (runs in `<body>` but could move below `<aside>`).

**Fix:** Extract theme init + sidebar-restore into `/static/app.js`. CSP can then allow by source, not `unsafe-inline`. Deduplicates the theme script across entry points.

#### M5 — Delete form/button pattern copy-pasted 4×
**Files:** `values.templ:55-57`, `tools.templ:34-36`, `tasks.templ:40-42`, `staff.templ:41-43`
```html
<form method="post" action={ templ.SafeURL("/<section>/" + id + "/delete") } style="display:contents">
  <button type="submit" class="btn btn-icon btn-icon-danger" onclick="return confirm('Delete this <x>?')" title="Delete">@iconDelete()</button>
</form>
```
Problems: (1) `onclick="return confirm(...)"` fails with strict CSP; (2) edit link also repeated 4× unchanged.

**Fix:** Extract `@rowActions(basePath, id, label string)` templ component. Replace `confirm()` with a `<dialog>`-based confirm helper.

#### M6 — `*FormTitle`/`*FormAction`/`*Count` helpers duplicated 3×
**Files:** `values.templ:150-171`, `tasks.templ:104-127`, `staff.templ:146-168`
Three copies of the same "new/edit" + "post to /x or /x/:id" + pluralizing count shape.

**Fix:** Move to `helpers.go`: `formTitle(noun string, isNew bool)`, `formAction(base, id string, isNew bool)`, `countWord(n int, singular, plural string)`.

#### M7 — Table row markup duplicated 4×
**Files:** `values.templ:33-63`, `tools.templ:19-41`, `tasks.templ:22-48`, `staff.templ:23-48`
Four copies of the same `<table><thead>/<tbody>` shell with the same actions column. Staff has already drifted from the others.

**Fix:** Extract `@tableRowActions(editHref, deleteHref, label string)` templ component at minimum; consider a generic table shell.

#### M8 — Raw hex/rgba values escape the token system
**File:** `web/static/style.css`
- FAB shadow `style.css:548`: `rgba(191, 9, 47, 0.5)` — hardcoded crimson. Should use `--shadow-accent` or `color-mix(in srgb, var(--color-accent) 50%, transparent)`.
- Theme toggle border `style.css:222-223, 237`: `rgba(255,255,255,0.15)` / `0.3` — light-on-dark assumption, will break if nav ever goes light. Define `--color-nav-overlay`.
- Commented-out ghost rule at `style.css:206-211` (dead code).

#### M9 — No `:focus-visible` styles on any interactive element
**File:** `web/static/style.css:716`
```css
input[type="text"]:focus, … { outline: none; border-color: var(--color-accent); }
```
`outline: none` on inputs without a replacement ring. No focus styling on: `.btn` (all variants), `.topnav-link`, `.topnav-settings`, `.theme-toggle`, `.sidebar-item`, `.sidebar-toggle`, `.tab`, `.fab`, `.pick-chip`, `.integration-card`.

**Fix:**
```css
:is(.btn, .btn-icon, .topnav-link, .theme-toggle, .sidebar-item,
    .sidebar-toggle, .tab, .fab, .pick-chip, .integration-card):focus-visible,
input:focus-visible, select:focus-visible, textarea:focus-visible {
    outline: 2px solid var(--color-accent);
    outline-offset: 2px;
}
```

---

### MINOR

| ID | File | Issue |
|----|------|-------|
| m1 | `layout.templ:128-142` | FAB uses same `fabPlus()` icon for all sections; consider `fabIcon(section)` for future divergence |
| m2 | `staff.templ:160-168`, `tasks.templ:118-127` | Pluralization helpers are anglocentric; Bahasa Indonesia doesn't pluralize — will break under i18n |
| m3 | `setup.templ:245-266` | `itoa()` reinvents `strconv.Itoa` in 20 lines; delete it |
| m4 | `tools.templ:142-152` | `configValue()` silently drops non-string types with `s, _ := v.(string)` |
| m5 | `layout.templ:182-204` | `initials()` doc says "uppercase letters" but logic allows digits |
| m6 | All 4 list pages | `confirm()` dialogs are not translatable; use `<dialog>` for Indonesian SMB audience |
| m7 | Multiple templates | 8 `style="display:contents"` inline styles; move to `.inline-form` utility |
| m8 | `style.css:958-966, 1035-1043, 423-433` | 3 identical eyebrow-heading definitions under different class names; consolidate to `.eyebrow-heading` |
| m9 | `layout.templ:128-142` | `.fab` has no focus ring or deliberate `tabindex` — keyboard order is unexpected |
| m10 | `settings.templ:19,27`, `setup.templ:127,131,152` | Missing `autocomplete` attributes on business info fields |
| m11 | `style.css:697-718` | `<select>` is half-customized: `appearance:none` applied but no chevron added back |
| m12 | `style.css:29`, `style.css:846` | `--color-text-faint` consumed once; `--color-interactive` consumed once — consider dropping the tokens or expanding usage |

---

### NOTES

| ID | Issue |
|----|-------|
| n1 | `style.css:833-848`: `.manifest`, `.skill-editor`, `details summary` styles are dead — no `<details>` or `.manifest` in any templ file. Deferred features per CLAUDE.md; add a comment or prune. |
| n2 | `.empty-state-ill` uses `132px`/`92px` — only pixel-unit holdouts in an otherwise rem-only file |
| n3 | 11 unique font-size values hardcoded per component — no type scale variable (see design system section) |
| n4 | `.skill-editor` at `style.css:727` references a feature not yet in any template |
| n5 | `.welcome-screen` and `.setup-card` both use `max-width: 480px` — worth a `--container-sm` token |
| n6 | `icons.templ` SVGs declare `xmlns=...` (redundant in HTML5); `layout.templ` tab icons don't — inconsistent |

---

### Security

- **XSS:** None found. Templ auto-escaping applied consistently on all user-controlled strings.
- **URL injection:** `templ.SafeURL(...)` used throughout. Confirm server enforces `^[a-z0-9_-]+$` on ID input — IDs embed in path segments without explicit percent-encoding.
- **CSRF:** No token on any POST form. Acceptable for local single-user device at `:18800`; flag before any network-exposed mode.
- **CSP:** Inline scripts (see M4) make `unsafe-inline` required today. Blocked by M4 fix.

---

## Design system extraction

### Current token status

| Token family | Status | Notes |
|-------------|--------|-------|
| Semantic color tokens (~30 vars) | **Done** | Dual light/dark, DTCG-friendly naming, solid foundation |
| Radius scale (5 steps) | **Done** | Clean, keep as-is |
| Shadow tokens (2 elevations) | **Done** | Dual theme |
| Font families (3) | **Done** | Good stacks |
| Layout constants (`--topnav-h`, `--tabbar-h`) | **Done** | Useful for layout frames |
| `--color-accent-subtle` | **Bug** | Defined nowhere, blocks pick-chip extraction |
| Spacing scale | **Missing** | 80 raw values, 15+ unique sizes — extraction blocker |
| Type scale | **Missing** | 48 raw font-size declarations, 8 unique values — extraction blocker |
| Weight scale | **Missing** | 4 values, never tokenized |
| Z-index scale | **Missing** | 3 raw values (100, 199, 200) |
| Duration scale | **Missing** | `0.1s`, `0.15s`, `0.2s` across 18+ rules |
| Letter-spacing scale | **Missing** | 4 unique values |

### Proposed token taxonomy

**Tier 1 — Primitives**
```
color.brand.crimson    #bf092f
color.brand.navy.dark  #132440
color.brand.navy       #16476a  (unused in product — keep as brand reserve)
color.brand.teal       #3b9797
font.family.brand      "URW Gothic", "Urbanist", system-ui, sans-serif
font.family.ui         "Inter", system-ui, -apple-system, sans-serif
font.family.mono       "JetBrains Mono", ui-monospace, monospace
```

**Tier 2 — Scales**

Spacing (4px base):
```
space.1   0.125rem   space.5   0.625rem   space.9   1.5rem
space.2   0.25rem    space.6   0.75rem    space.10  1.75rem
space.3   0.375rem   space.7   1rem       space.11  2rem
space.4   0.5rem     space.8   1.25rem    space.12  2.5rem
```
Normalize one-offs (`0.05rem`, `0.15rem`, `0.2rem`, `0.3rem`, `0.4rem`, `0.55rem`, `0.875rem`) to nearest stop.

Type scale:
```
font.size.xs   0.6875rem   (th, cat-tag, sidebar-heading, tab label)
font.size.sm   0.8125rem   (hint, btn-sm, manifest, label)
font.size.base 0.875rem    (body, sidebar-item, btn, td)
font.size.md   0.9375rem   (input, integration-label, welcome-desc)
font.size.lg   1.125rem    (h2)
font.size.xl   1.25rem     (empty-state-title)
font.size.2xl  1.5rem      (h1)
font.size.3xl  1.75rem     (setup-title, welcome-tagline)
```

Weight scale:
```
font.weight.regular  400
font.weight.medium   500
font.weight.semibold 600
font.weight.bold     700
```

Z-index:
```
z.sticky   100    (topnav)
z.fab      199
z.overlay  200    (tabbar)
z.modal    1000   (reserved)
```

Duration:
```
duration.fast   0.1s
duration.base   0.15s
duration.slow   0.2s
```

**Tier 3 — Semantic** (already good — keep existing `--color-*` vars, add `--color-accent-subtle` + `--color-nav-overlay` + `--shadow-accent`)

**Tier 4 — Component tokens** — skip at this scale, overkill.

### Components to formalize (ranked by reuse value)

| Rank | Component | Extract? | Notes |
|------|-----------|----------|-------|
| 1 | Button (primary/secondary/danger + sm) | Yes | `style.css:592-637`. 4 variants × 2 sizes × 2 themes |
| 2 | Button icon (neutral + danger) | Yes | `style.css:638-671`. Every table row |
| 3 | Form field (label + input/textarea/select + hint) | Yes | `style.css:687-726, 910-914` |
| 4 | Card | Yes | `style.css:679-686`. Surface container primitive |
| 5 | Alert (ok/error/warn) | Yes | `style.css:806-830` |
| 6 | Pick chip (inline + block variants) | Yes | `style.css:749-788`. Onboarding hot path |
| 7 | Empty state | Yes | `style.css:857-908`. Every list page |
| 8 | Top nav (full + minimal variants) | Yes | `style.css:137-277` |
| 9 | Sidebar item (active/action/count variants) | Yes | `style.css:435-485` |
| 10 | Tab bar + tab | Yes | `style.css:488-534`. Mobile-critical |
| 11 | FAB | Yes | `style.css:537-558` |
| 12 | Cat tag badge | Yes | `style.css:926-942` |
| 13 | Integration card | Borderline | Consider merging as pick-chip-block variant |
| 14 | Data table row (desktop) | Yes | `style.css:561-589` |
| 15 | List card row (mobile) | Yes | `style.css:1179-1209`. Separate from table row |
| 16 | Setup progress indicator | Borderline | Onboarding-only |
| 17 | Welcome screen layout | No | One-shot marketing composition |
| 18 | Dashboard home | No | One-shot |

**Icon assets to export (18 total):**
8 integration brand icons + 5 tab icons + edit, delete, sun, moon, plus (FAB) = 18 SVG assets → single Penpot library page.

### Structural concerns for extraction

1. **Responsive table→card transform** (`style.css:1166-1209`) cannot be a single component with variants in Penpot/Pencil — model as two separate components with a design note.
2. **Sidebar collapsed state** — icon-rail mode requires building the collapsed variant from scratch in the design tool, not as a CSS modifier.
3. **`[data-theme="dark"]` theming** maps cleanly to Penpot variable modes (light/dark at board level).
4. **`logo-symbol.svg` hardcoded `fill="#bf092f"`** — convert to `fill="currentColor"` before extraction.
5. **Icons are templ functions** — 18 manual SVG exports required, no sprite sheet.
6. **No prior Figma/Penpot source** — clean slate, no legacy reconciliation needed.

### Migration path

**Phase 1 — Normalize CSS (est. ~2h)**
Add `--space-*`, `--font-size-*`, `--font-weight-*`, `--z-*`, `--duration-*` blocks to `:root` after `--radius-sm`. Sweep and replace raw values. Fix the two bugs: define `--color-accent-subtle` for both themes; add `--shadow-accent` and use at `style.css:548`. Visual diff with `DEV=1`.

**Phase 2 — DTCG tokens (est. ~half day)**
Create `design/tokens.json` in W3C DTCG format. Use Style Dictionary to generate `web/static/tokens.css`. Replace hand-written `:root` and `[data-theme="dark"]` blocks in `style.css` with `@import url('./tokens.css')`. `tokens.json` becomes the canonical source.

**Phase 3 — Import to Penpot or Pencil (est. ~half day)**
- **Pencil:** `mcp__pencil__set_variables` is available in this environment — script a one-pass call ingesting primitive + semantic tokens (flatten `color.surface.subtle` → `color-surface-subtle`).
- **Penpot:** JSON variable import via the Penpot UI variable panel (~40 tokens, manual but fast). Or write a Style Dictionary custom transform (~30 lines) for automated sync.

**Phase 4 — Build components (est. ~1 day for top 10)**
Build top-10 components in order. Each consumes only variables, never raw values. Build dark-mode variants for anything using `color.surface`, `color.border`, or `color.accent`.

**Phase 5 — Ongoing sync**
`tokens.json` is the contract. When CSS changes a token, regenerate `tokens.css` and re-import to the design tool. Components will drift — accept that cost; the token layer is what stays synced.

### Recommendation

Do Phase 1 first. Do not export to Penpot/Pencil while 80 unnormalized spacing values exist — you'll reproduce the chaos in the design tool. Phase 1 is the load-bearing step; phases 2–4 are mechanical once it's done.
