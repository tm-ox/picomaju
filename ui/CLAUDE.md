# ui/

Active frontend: templui v1.10.0 + Tailwind CSS v4.2.4. `web/` is legacy reference only.

Components added via `templui add <name>`. Installed: button, alert, label, card, input, selectbox (+ icon, popover, aspectratio deps).

Icons: `github.com/bryanvaz/go-templ-lucide-icons` — import alias `icons`, usage `icons.Name(templ.Attributes{"class": "size-4"})`.

## Template map

| File | Key exports |
|---|---|
| `layout.templ` | `AppLayout`, `ThemeToggle` |
| `nav.templ` | `NavData`, `appNavLink`, `AppBottomTabBar`, `navInitials` |
| `shared.templ` | `SidebarItem`, `emptySidebarNav`, `pageHeader`, `emptyState`, `rowList`, `rowItem`, `badge`, `rowActions`, `field` |
| `helpers.go` | `catLabel`, `countWord`, `categoryLabel`, `configValue`, `staffInitials`, `staffToolCount`, `staffIconOptions`, `includesStr` |
| `staff.templ` | `StaffListPage`, `StaffFormPage`, `StaffDetailPage`, sidebar navs, icon picker |
| `values.templ` | `valueSidebarNav`, `ValueListPage`, `ValueFormPage`, `ValidationFragment` |
| `tools.templ` | `toolSidebarNav`, `ToolListPage`, `NewToolPage`, `ToolFormPage` |
| `tasks.templ` | `taskSidebarNav`, `TaskListPage`, `TaskFormPage` |
| `settings.templ` | `SettingsPage` |
| `setup.templ` | `SetupStep1/2/3Page` |
| `welcome.templ` | `WelcomePage` |

## AppLayout shell

`AppLayout(title string, nd NavData, nav templ.Component)`

**Sidebar** (`bg-sidebar`, `h-dvh`, dark):
- Header: `size-7` business avatar + name (`data-sidebar-label`) + desktop toggle (`hidden sm:flex`)
- Middle: `@nav` (page-specific)
- Footer: Settings + theme toggle (both as `SidebarItem`)

**In-content header** (`h-12 border-b`): mobile hamburger (`flex sm:hidden`) left; `hidden sm:flex` nav links (Staff / Values / Tools / Tasks) center.

**Content area**: `max-w-2xl mx-auto px-4 py-6 flex flex-col gap-6` — set in AppLayout, not per page.

**Bottom tab bar** (`AppBottomTabBar`, `sm:hidden`): `shrink-0` flex child inside `#sidebar-main` (not fixed) — slides with content when sidebar opens on mobile. Outer: `px-4 pb-4 pt-2`. Card: `bg-card rounded-xl border border-border shadow-sm`. 4 tabs: Staff / Values / Tools / Tasks — icon + label, active `text-foreground font-medium`.

No FAB — page header buttons handle create actions on all screen sizes.

## Sidebar state

| State | Width | Position |
|---|---|---|
| Mobile closed | `0` | static |
| Mobile open | `14rem` | `absolute z-50`; main `translateX(14rem)` |
| Desktop collapsed | `3.5rem` | static, icon-only |
| Desktop expanded | `14rem` | static, with labels |

`localStorage` key `"sidebar"`. Default: open desktop / closed mobile. `data-sidebar-label` on elements hidden when collapsed.

`SidebarItem(href, lbl, key, active string)` — `mx-2 px-1 py-0.5`, icon in `size-8` container. Active: `bg-sidebar-accent text-sidebar-foreground`. Inactive: `text-sidebar-foreground/60`.

`--sidebar-accent` kept close to `--sidebar` to prevent expanded state appearing lighter (active items cover more surface): light `oklch(0.21)`, dark `oklch(0.13)`.

## Sidebar nav components

| Page | Component | Filter mechanism |
|---|---|---|
| Staff home | `staffListSidebarNav(members)` | staff member links with `size-7` avatar |
| Staff detail | `staffSidebarNav(m, section)` | Overview / Profile / Values / Tools / Tasks + back |
| Values | `valueSidebarNav(cats, activeCat)` | All + 5 categories via `?cat=<id>` |
| Tools | `toolSidebarNav(activeCat)` | All + messaging/commerce/payments/utilities via `?cat=` |
| Tasks | `taskSidebarNav(activeCat)` | All + same 4 catalog categories via `?tool_cat=` |

`catLabel(cat string) string` → display label for catalog category IDs; `""` for unknown (caller provides section default).

Backend handlers filter before passing filtered slice + `activeCat` to template. Tasks filter: keep tasks with ≥1 tool in requested category (via `CatalogByType()[tool.Type].Category`).

## Staff pages

**Home `/`**: `StaffListPage` renders `staffDashboard` — 2-col card grid, each card links to `/staff/:id`.

**Detail `/staff/:id`**: `StaffDetailPage(m, tasks, tools, values, cats, nd, section, formErr)`. Sections: overview / profile / values / tools / tasks. Overview stat cards ordered Status → Values → Tools → Tasks; each is `staffStatCardLink` linking to `?s=profile/values/tools/tasks`.

## UI conventions

- Checkboxes `sr-only`; wrapping `<label>` is visual toggle via `has-[:checked]:border-primary has-[:checked]:bg-primary/5`
- `bg-popover` = `--background` (not `--card`) in both modes
- Staff avatars: `size-9 rounded-full bg-primary text-primary-foreground`
- Typography base styles in `@layer base` in `input.css`; `p` is large (marketing scale) — use `<div class="text-muted-foreground">` for dense UI

## Key brand tokens (full set in `ui/assets/css/input.css`)

| Token | Light | Dark |
|---|---|---|
| `--primary` | `oklch(0.41 0.22 18.5)` | same |
| `--background` | `oklch(0.972 0.006 240)` | `oklch(0.115 0.042 258)` |
| `--card` | `oklch(1 0 0)` | `oklch(0.183 0.057 258)` |
| `--sidebar` | `oklch(0.167 0.058 258)` | `oklch(0.082 0.04 258)` |
| `--sidebar-accent` | `oklch(0.21 0.058 258)` | `oklch(0.13 0.05 258)` |

Theme: `.dark` class on `<html>` (`@custom-variant dark`). `localStorage` key `theme`.
