# web/static/

## Files

```
style.css       — CSS custom properties; light + dark mode; mobile-first (bottom tabs, FAB, card rows)
app.js          — theme init (pre-paint, no-FOUC) + sidebar collapsed-state restore on DOMContentLoaded
datastar.js     — MUST be downloaded manually from data-star.dev releases; embedded into binary at build time
logo-symbol.svg — crimson PM mark (140×98), hardcoded fill="#bf092f"
logo-type.svg   — horizontal wordmark lockup (used in onboarding header + footer)
logo-stack.svg  — stacked symbol + wordmark lockup
```

## Theming and token system (`style.css`)

CSS custom properties throughout. Token blocks in `:root` (lines 1–111):

- Palette primitives (`--crimson`, `--navy-dark`, `--navy`, `--teal`)
- Light mode semantic colors (`--color-*`, `--shadow-*`)
- Spacing scale (`--space-1` → `--space-13`, 0.125rem–2.5rem)
- Type scale (`--font-size-2xs` → `--font-size-3xl`, 0.6875rem–1.75rem)
- Font weight scale (`--font-weight-regular/medium/semibold/bold`)
- Z-index scale (`--z-sticky/fab/overlay`)
- Duration scale (`--duration-fast/base/slow`)

Dark mode overrides in `[data-theme="dark"]` (colors only — scales are theme-invariant).

Palette: `#bf092f` crimson, `#132440` dark navy, `#16476a` mid navy, `#3b9797` teal.

Theme toggle switches `data-theme` on `<html>` and persists in `localStorage`. `app.js` is loaded synchronously (no defer) in `<head>` to apply the saved theme before first paint — prevents flash.

## Design tokens

`design/tokens.json` (repo root) is the DTCG-format canonical token file. It mirrors `:root` exactly. A Style Dictionary pipeline to auto-generate CSS from it is deferred — edit `style.css` `:root` and `tokens.json` in sync when changing token values.
