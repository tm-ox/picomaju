# ui/

Active frontend: templui v1.10.0 + Tailwind CSS v4.2.4. `web/` is legacy reference only.

Components added via `templui add <name>`. Installed: button, alert, label, card, input, selectbox (+ icon, popover, aspectratio deps).

Icons: `github.com/bryanvaz/go-templ-lucide-icons` — import alias `icons`, usage `icons.Name(templ.Attributes{"class": "size-4"})`.

## Template packages

Templates are organised into sub-packages under `ui/templates/`. Each directory is one Go package. Cross-package imports use `shell.*` for shared components.

| Package | Go alias | Key exports |
|---|---|---|
| `shell/` | `shell` | `AppLayout`, `ChatAppLayout`, `Layout`, `ThemeToggle`, `NavData`, `AppBottomTabBar`, `SidebarItem`, `PageHeader`, `EmptyState`, `ListSection`, `RowList`, `RowItem`, `Badge`, `RowActions`, `FormCard`, `Field`, `ValueCatIcon`, `EmptySidebarNav`; helpers: `IncludesStr`, `FormTitle`, `FormAction`, `CountWord`, `CategoryLabel`, `CatLabel` |
| `staff/` | `stafftpl` | `StaffListPage`, `StaffFormPage`, `StaffDetailPage`, `StaffChatPage`; private: sidebar navs, icon picker, helpers |
| `values/` | `valuestpl` | `ValueListPage`, `ValueFormPage`, `ValidationFragment`; private: sidebar nav, `groupValuesByCat` |
| `tools/` | `toolstpl` | `ToolListPage`, `NewToolPage`, `ToolFormPage`; private: sidebar nav, `groupToolsByCat`, `configValue` |
| `tasks/` | `taskstpl` | `TaskListPage`, `TaskFormPage`; private: sidebar nav |
| `license/` | `licensetpl` | `LicensePage(l, nd, activated, formErr, dev)` — plan status, credits bar, credit pack + plan cards, dev activation panel |
| `settings/` | `settingstpl` | `SettingsPage` |
| `setup/` | `setuptpl` | `WelcomePage`, `DashboardPage`, `SetupStep1Page`, `SetupStep2Page`, `SetupStep3Page` |
| `workshop/` | `workshoptpl` | `WorkshopPage` |

## Layout shells

### AppLayout — standard pages

`shell.AppLayout(title string, nd shell.NavData, nav templ.Component)`

Content area: `max-w-2xl mx-auto px-4 py-6 flex flex-col gap-6 min-h-full` inside `flex-1 overflow-y-auto`. Includes `appContentFooter()` ("Powered by [logo] PicoMaju") at the bottom via `mt-auto`.

### ChatAppLayout — chat pages

`shell.ChatAppLayout(title string, nd shell.NavData, nav templ.Component)`

Content area: `max-w-2xl mx-auto flex-1 flex flex-col min-h-0 overflow-hidden` inside `flex-1 overflow-hidden flex flex-col`. Children must own their own padding and scroll: header (`shrink-0`), messages (`flex-1 overflow-y-auto`), input (`shrink-0`). No footer. Tab bar (`AppBottomTabBar`) is a sibling of the content area, naturally above it on mobile.

### Shared sub-components (private)

`shell/layout.templ` extracts `appHead`, `appStyles`, `appSidebarAside`, `appTopBar`, `appSidebarScript`, `appContentFooter` — both layouts compose from these.

**Sidebar** (`bg-sidebar`, `h-dvh`, dark):
- Header: `size-7` business avatar + name (`data-sidebar-label`) + desktop toggle (`hidden sm:flex`)
- Middle: `@nav` (page-specific)
- Footer: Plan & Credits (Zap icon, `/license`) + Settings + theme toggle (all as `SidebarItem`)

**In-content header** (`h-12 border-b`): mobile hamburger (`flex sm:hidden`) left; `hidden sm:flex` nav links (Staff / Values / Tools / Tasks) center.

**Bottom tab bar** (`AppBottomTabBar`, `sm:hidden`): `shrink-0` flex child inside `#sidebar-main` (not fixed) — slides with content when sidebar opens on mobile. Outer: `px-4 pb-4 pt-2`. Card: `bg-card rounded-xl border border-border shadow-sm`. 4 tabs: Staff / Values / Tools / Tasks — icon + label, active `text-foreground font-medium`.

No FAB — page header buttons handle create actions on all screen sizes.

## Sidebar state

| State | Width | Position |
|---|---|---|
| Mobile closed | `0` | static |
| Mobile open | `14rem` | `absolute z-50`; main `translateX(14rem)` |
| Desktop collapsed | `3.5rem` | static, icon-only |
| Desktop expanded | `14rem` | static, with labels |

`localStorage` key `"sidebar"`. Default: open desktop / closed mobile. `data-sidebar-label` on elements hidden when collapsed. `white-space: nowrap` on all `[data-sidebar-label]` prevents text wrapping during transition.

`SidebarItem(href, lbl, key, active string)` — `mx-2 px-1 py-0.5`, icon in `size-8` container. Active: `bg-sidebar-accent text-sidebar-foreground`. Inactive: `text-sidebar-foreground/60`.

`--sidebar-accent` kept close to `--sidebar` to prevent expanded state appearing lighter (active items cover more surface): light `oklch(0.21)`, dark `oklch(0.13)`.

## Sidebar nav components

| Page | Component | Filter/content |
|---|---|---|
| Staff home | `staffListSidebarNav(members)` | staff member links, icon at `size-4` (UserRound fallback) |
| Staff detail | `staffSidebarNav(m, section, chats []chat.Chat)` | Overview / Profile / Values / Tools / Tasks + divider + New Chat button + chat list + back |
| Values | `valueSidebarNav(cats, activeCat)` | All + 5 categories via `?cat=<id>` |
| Tools | `toolSidebarNav(activeCat)` | All + messaging/commerce/payments/utilities via `?cat=` |
| Tasks | `taskSidebarNav(activeCat)` | All + same 4 catalog categories via `?tool_cat=` |

**New Chat button** in `staffSidebarNav`: `<form class="px-2">` + `<button class="w-full ... bg-primary text-primary-foreground">` — matches `SidebarItem` lateral padding (`px-2` on form ≡ `mx-2` on link).

`catLabel(cat string) string` → display label for catalog category IDs; `""` for unknown (caller provides section default).

Backend handlers filter before passing filtered slice + `activeCat` to template. Tasks filter: keep tasks with ≥1 tool in requested category (via `CatalogByType()[tool.Type].Category`).

## Grouped "All" views (Values + Tools)

When `activeCat == ""`, Values and Tools list pages render items in `listSection` groups by category instead of a flat `rowList`. Filtered views remain flat.

- `groupValuesByCat(cats, vals)` → `[]valCatGroup{Cat, Values}` — ordered by `DefaultCategories`, empty cats omitted
- `groupToolsByCat(tools)` → `[]toolCatGroup{Label, Tools}` — ordered messaging/commerce/payments/utilities, empty cats omitted
- `listSection(label string)` in `shared.templ` — renders `<h5>` category heading + `rowList` wrapper

## Staff pages

**Home `/`**: `StaffListPage` renders `staffDashboard` — 2-col card grid, each card links to `/staff/:id`.

**Detail `/staff/:id`**: `StaffDetailPage(m, tasks, tools, values, cats, nd, section, formErr, chats, compiled bool)`. Sections: overview / profile / values / tools / tasks. Overview stat cards ordered Status → Values → Tools → Tasks; each is `staffStatCardLink` linking to `?s=profile/values/tools/tasks`. Directives card has "Compile & Deploy" button (`POST /staff/:id/compile`); shows "Deployed" + "Recompile" after `?compiled=1`.

**Chat `/staff/:id/chats/:chatId`**: `StaffChatPage(m, c, chats, nd, licensed bool)` using `ChatAppLayout`. When `licensed=false`: empty state copy changes to "Activate to chat"; input panel replaced with a `/license` upgrade prompt card. `POST .../messages` redirects to `/license` if not active. Three `shrink-0`/`flex-1`/`shrink-0` children: header with inline rename form + delete button, messages scroll area, input panel. Send button uses `ArrowUp` icon. Rename: text input styled as plain text, checkmark button on `group-focus-within:opacity-100`, auto-submits on blur if changed.

## UI conventions

- Checkboxes `sr-only`; wrapping `<label>` is visual toggle via `has-[:checked]:border-primary has-[:checked]:bg-primary/5`
- `bg-popover` = `--background` (not `--card`) in both modes
- Staff avatars / icon containers: `size-9 rounded-lg bg-primary text-primary-foreground` (not `rounded-full`)
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
