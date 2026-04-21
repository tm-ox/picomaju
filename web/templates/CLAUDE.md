# web/templates/

## Files

```
layout.templ        — base HTML shell; topnav (avatar | Home+Values+Tools+Tasks+Staff | theme icon + settings);
                      bottomTabs (5-tab mobile bar); fab (section floating action button);
                      tab/fab icon glyphs; initials() helper; logo-type.svg during onboarding
dashboard.templ     — DashboardPage: centered logo-symbol.svg home screen
shared.templ        — rowActions(editHref, deleteHref, noun): shared edit-link + delete-form used in all list tables
sidebar.templ       — contextual sidebar (sidebarHeader with toggle; switches per active section)
empty_state.templ   — EmptyState component (illustrated card + CTA); EmptyIconValues/Tools/Tasks/Staff glyphs
icons.templ         — toolIcon(type) brand SVGs; categoryIcon/taskItemIcon/staffItemIcon sidebar glyphs;
                      tabIconHome/Values/Tools/Tasks/Staff tab bar glyphs;
                      iconSun/iconMoon theme toggle; iconEdit/iconDelete action buttons
values.templ        — Value list page (EmptyState when empty), Value form, ValidationFragment
tools.templ         — Tool list (EmptyState), NewToolPage (catalog radio picker), ToolFormPage
tasks.templ         — Task list (EmptyState), Task form (with tool picker)
staff.templ         — Staff list (EmptyState), Staff form (task + value picker)
settings.templ      — Settings page (business info + data dir)
setup.templ         — Welcome screen (WelcomePage) + three-step onboarding: SetupPage, FirstStaffPage, IntegrationsPage;
                      setupProgress component; welcomeLangOption/tzOption helpers; itoa() helper
helpers.go          — SidebarData type, includesStr()
```

`SidebarData` is built by `uiHandler.sidebarData(r, section)` and passed to every page render.

## Onboarding flow

On first launch, if no data directory is configured, **all routes redirect to `/welcome`**. Onboarding renders without a sidebar (`HideSidebar: true`, minimal header with logo-type.svg, full-width layout):

0. **`/welcome`** — Language picker (English / Bahasa Indonesia). Saves `languages` to settings, redirects to `/setup`.
1. **`/setup`** — Business Name + Data Directory (pre-filled `~/picomaju`), Timezone, Operating Hours. Defaults: `Asia/Jakarta`.
2. **`/setup/first-staff`** — First staff profile (name + optional description). Skip button goes directly to step 3.
3. **`/setup/integrations`** — Tool picker: select from the catalog to auto-create Tool entries; credentials configured later.

After step 3, user lands at `/values`. No restart required.

## Navigation

**Top nav** (sticky, `var(--topnav-h): 3.25rem`):
- Left: crimson avatar (business initials via `initials()`) + business name
- Center: Home | Values | Tools | Tasks | Staff (section links, `.active` on current section)
- Right: theme toggle icon button (moon/sun, `2rem` square) + settings gear
- During onboarding: minimal header with logo-type.svg only

**Sidebar** (contextual, collapsible — hidden during onboarding):
- Header row: collapse/expand chevron + section heading
- Collapsed state: `3rem` wide icon strip (both `.sidebar` and `.sidebar-inner` collapse to `3rem` so flex centering keeps icons visible)
- Expanded state: `13.75rem`
- Content switches per `ActiveSection`: `values` → category filter + New Value; `tools` → tool list + Add Tool; `tasks` → task list + New Task; `staff` → staff list + New Staff; `home`/empty → empty nav
- Collapsed state persists in `localStorage` (`sidebar-collapsed`)

**Mobile (≤640px):**
- Sidebar hidden; top nav center links hidden
- Fixed bottom tab bar: 5 tabs — Home | Values | Tools | Tasks | Staff (`grid-template-columns: repeat(5, 1fr)`)
- Floating action button (crimson circle, `3.25rem`) above tab bar for the active section's primary create action
- `viewport-fit=cover` + `env(safe-area-inset-bottom)` for iOS home bar
- Tables render as compact flex-row cards: label left (`flex: 1`), secondary columns hidden, action icon buttons right

**Footer**: logo-type.svg (`color: var(--color-text-muted)`)

## Icons (`icons.templ`)

**Sidebar / section:**
- `toolIcon(toolType string)` — switch over all 8 catalog types; brand SVG paths from Simple Icons; Midtrans → generic credit card; unknown → stacked-layers
- `categoryIcon()` — tag glyph for value category items
- `taskItemIcon()` — document glyph for task items
- `staffItemIcon()` — person glyph for staff items

**Tab bar** (stroke, 1.8px, `viewBox="0 0 24 24"`):
- `tabIconHome()` — house glyph
- `tabIconValues()` — star glyph
- `tabIconTools()` — wrench glyph
- `tabIconTasks()` — rounded rectangle + checkmark glyph
- `tabIconStaff()` — person circle glyph

**Theme toggle** (stroke, `viewBox="0 0 24 24"`):
- `iconSun()` — `class="icon-sun"` — shown in dark mode (click → light); hidden in light mode via CSS
- `iconMoon()` — `class="icon-moon"` — shown in light mode (click → dark); hidden in dark mode via CSS

**Action buttons** (stroke, `viewBox="0 0 16 16"`):
- `iconEdit()` — pencil glyph; used in `.btn-icon` edit links
- `iconDelete()` — trash glyph; used in `.btn-icon.btn-icon-danger` delete buttons

**FAB:** `fabPlus()` — plus glyph

All icons use `stroke="currentColor"` or `fill="currentColor"` and inherit color from CSS.

