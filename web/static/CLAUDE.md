# web/static/

## Files

```
style.css       — CSS custom properties; light + dark mode; mobile-first (bottom tabs, FAB, card rows)
datastar.js     — MUST be downloaded manually from data-star.dev releases; embedded into binary at build time
logo-symbol.svg — crimson PM mark (140×98), hardcoded fill="#bf092f"
logo-type.svg   — horizontal wordmark lockup (used in onboarding header + footer)
logo-stack.svg  — stacked symbol + wordmark lockup
```

## Theming (`style.css`)

CSS custom properties throughout. Two token blocks:

- `:root` — light mode (default)
- `[data-theme="dark"]` — dark mode

Palette: `#bf092f` crimson, `#132440` dark navy, `#16476a` mid navy, `#3b9797` teal.

Theme toggle switches `data-theme` on `<html>` and persists in `localStorage`. Inline `<script>` in `<head>` applies the saved theme before first paint to prevent flash.
