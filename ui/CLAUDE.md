# ui/

Active frontend: templui v1.10.0 + Tailwind CSS v4.2.4.

Components added via `templui add <name>`. Installed: button, alert, label, card, input, selectbox (+ icon, popover, aspectratio deps).

Icons: `github.com/bryanvaz/go-templ-lucide-icons` — import alias `icons`, usage `icons.Name(templ.Attributes{"class": "size-4"})`.

## Template packages

Templates are organised into sub-packages under `ui/templates/`. Each directory is one Go package. Cross-package imports use `shell.*` for shared components.

| Package | Go alias | Key exports |
|---|---|---|
| `shell/` | `shell` | `AppLayout`, `ChatAppLayout`, `Layout`, `ThemeToggle`, `NavData`, `AppBottomTabBar`, `SidebarItem`, `PageHeader`, `EmptyState`, `ListSection`, `RowList`, `RowItem`, `Badge`, `RowActions`, `FormCard`, `Field`, `ValueCatIcon`, `EmptySidebarNav`, `SettingsTabNav`; helpers: `IncludesStr`, `FormTitle`, `FormAction`, `CountWord`, `CategoryLabel`, `CatLabel` |
| `home/` | `hometpl` | `HomePage(nd, tab)` — placeholder dashboard; greeting, stat cards, tabs Overview/Activity/Agents via `?t=` |
| `staff/` | `stafftpl` | `StaffListPage`, `StaffFormPage`, `StaffDetailPage`, `StaffChatPage`; private: sidebar navs, icon picker, `currentUserCard`, helpers |
| `values/` | `valuestpl` | `ValueListPage`, `ValueFormPage`, `ValidationFragment`; private: sidebar nav, `groupValuesByCat` |
| `tools/` | `toolstpl` | `ToolListPage`, `NewToolPage`, `ToolFormPage`; private: sidebar nav, `groupToolsByCat`, `configValue` |
| `tasks/` | `taskstpl` | `TaskListPage`, `TaskFormPage`; private: sidebar nav |
| `license/` | `licensetpl` | `LicensePage(l, nd, activated, formErr, dev)` — uses `SettingsTabNav("plan")` |
| `settings/` | `settingstpl` | `SettingsPage` — uses `SettingsTabNav("general")` |
| `users/` | `userstpl` | `UserListPage(users, nd)`, `UserFormPage(u, staffOptions, nd, isNew, formErr)`, `ProfilePage(u, nd, formErr)` — `UserListPage` uses `SettingsTabNav("users")` |
| `login/` | `logintpl` | `LoginPage(users []UserEntry, selectedID, formErr)` |
| `setup/` | `setuptpl` | `WelcomePage`, `SetupStep1Page`, `SetupOwnerPage`, `SetupStep2Page`, `SetupStep3Page` |

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
- No footer — all user/settings access moved to top-bar avatar menu

**In-content header** (`h-12 border-b`): mobile hamburger left; nav links center (Home / Values / Tools / Tasks / Staff); avatar menu button right.

**Top-bar avatar menu**: `size-7 rounded-md bg-primary text-primary-foreground font-brand font-bold text-[0.625rem] tracking-wide` inside `size-8` container — matches business avatar exactly. Click → `w-72 rounded-xl mt-4` dropdown panel. Items: user name+role header, Profile, Settings, theme toggle, Sign out. All items `whitespace-nowrap`. JS: `toggleUserMenu(btn)` + outside-click listener, IIFE in `userMenuButton` component.

**Bottom tab bar** (`AppBottomTabBar`, `sm:hidden`): 5 tabs — Home / Values / Tools / Tasks / Staff. `shrink-0` flex child, not fixed. Card: `bg-card rounded-xl border border-border shadow-sm`.

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

`--sidebar-accent` kept close to `--sidebar` to prevent expanded state appearing lighter (active items cover more surface).

## Sidebar nav components

| Page | Component | Filter/content |
|---|---|---|
| Home | `EmptySidebarNav()` | — |
| Staff list | `staffAccordionNav(members, "", "", nil)` | all collapsed |
| Staff detail | `staffAccordionNav(members, m.ID, section, chats)` | active staff expanded |
| Staff chat | `staffAccordionNav(members, m.ID, c.ID, chats)` | active staff + chat highlighted |
| Staff form | `staffAccordionNav(members, "", "", nil)` | all collapsed |
| Values | `valueSidebarNav(cats, activeCat)` | All + 5 categories via `?cat=<id>` |
| Tools | `toolSidebarNav(activeCat)` | All + messaging/commerce/payments/utilities via `?cat=` |
| Tasks | `taskSidebarNav(activeCat)` | All + same 4 catalog categories via `?tool_cat=` |
| Settings/Users/License | `EmptySidebarNav()` | tab nav rendered in-content via `SettingsTabNav` |

**New Chat button** in `staffSidebarNav`: `<form class="px-2">` + `<button class="w-full ... bg-primary text-primary-foreground">` — matches `SidebarItem` lateral padding (`px-2` on form ≡ `mx-2` on link).

`catLabel(cat string) string` → display label for catalog category IDs; `""` for unknown (caller provides section default).

Backend handlers filter before passing filtered slice + `activeCat` to template. Tasks filter: keep tasks with ≥1 tool in requested category (via `CatalogByType()[tool.Type].Category`).

## Grouped "All" views (Values + Tools)

When `activeCat == ""`, Values and Tools list pages render items in `listSection` groups by category instead of a flat `rowList`. Filtered views remain flat.

- `groupValuesByCat(cats, vals)` → `[]valCatGroup{Cat, Values}` — ordered by `DefaultCategories`, empty cats omitted
- `groupToolsByCat(tools)` → `[]toolCatGroup{Label, Tools}` — ordered messaging/commerce/payments/utilities, empty cats omitted
- `listSection(label string)` in `shared.templ` — renders `<h5>` category heading + `rowList` wrapper

## Settings area

`SettingsTabNav(active string, nd NavData)` in `shell/shared.templ`. Tabs: General (`/settings`) | Users (`/users`, owner only — checks `nd.CurrentUserRole == "owner"`) | Plan (`/license`). Accepts children for right-slot action. Used in place of `PageHeader` on all three pages. `UserListPage` passes `+` button as child.

## Staff pages

**List `/staff`**: `StaffListPage` renders `currentUserCard(nd)` (links to `/profile`) above the staff grid when logged in, then `staffDashboard` — 2-col card grid, each card links to `/staff/:id`.

**Detail `/staff/:id`**: `StaffDetailPage(members, m, tasks, tools, values, cats, nd, section, formErr, chats, compiled, licensed, running)`. Sections: overview / profile / values / tools / tasks. Overview stat cards Status → Values → Tools → Tasks. Directives card + Runtime card.

**Chat `/staff/:id/chats/:chatId`**: `StaffChatPage(m, c, chats, nd, licensed bool)` using `ChatAppLayout`. When `licensed=false`: empty state copy changes to "Activate to chat"; input panel replaced with a `/license` upgrade prompt card. `POST .../messages` redirects to `/license` if not active. Three `shrink-0`/`flex-1`/`shrink-0` children: header with inline rename form + delete button, messages scroll area, input panel. Send button uses `ArrowUp` icon. Rename: text input styled as plain text, checkmark button on `group-focus-within:opacity-100`, auto-submits on blur if changed.

## UI conventions

- Checkboxes `sr-only`; wrapping `<label>` is visual toggle via `has-[:checked]:border-primary has-[:checked]:bg-primary/5`
- `bg-popover` = `--background` (not `--card`) in both modes
- Staff avatars / icon containers: `size-9 rounded-lg bg-primary text-primary-foreground` (not `rounded-full`)
- Typography base styles in `@layer base` in `input.css`; `p` is large (marketing scale) — use `<div class="text-muted-foreground">` for dense UI

## Key brand tokens (full set in `ui/assets/css/input.css`)

All tokens are HSL — editable directly.

| Token | Light | Dark |
|---|---|---|
| `--primary` | `hsl(348, 91%, 39%)` | same |
| `--background` | `hsl(210, 33%, 97%)` | `hsl(212, 52%, 10%)` |
| `--card` | `hsl(0, 0%, 100%)` | `hsl(216, 52%, 16%)` |
| `--sidebar` | `hsl(217, 54%, 16%)` | same |
| `--sidebar-accent` | `hsl(217, 45%, 21%)` | `hsl(217, 35%, 13%)` |

Theme: `.dark` class on `<html>` (`@custom-variant dark`). `localStorage` key `theme`.
