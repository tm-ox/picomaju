package api

// Onboarding handlers for the 4-step setup wizard.
// Drop this file into internal/api/ and add the routes from router_onboarding_patch.go.

import (
	"net/http"
	"strings"

	"picomaju/internal/settings"
	"picomaju/internal/staff"
	"picomaju/web/templates"
)

// ─── Step 2 — Languages / Timezone / Hours ─────────────────────────────────

func (h *uiHandler) languagesPage(w http.ResponseWriter, r *http.Request) {
	cfg, _ := h.settings.Load()
	if cfg == nil {
		cfg = &settings.Settings{}
	}
	// Sensible defaults on first visit.
	langs := cfg.Languages
	if len(langs) == 0 {
		langs = []string{"id", "en"}
	}
	tz := cfg.Timezone
	if tz == "" {
		tz = "Asia/Jakarta"
	}
	templates.LanguagesPage(langs, tz, cfg.Hours, "").Render(r.Context(), w)
}

func (h *uiHandler) completeLanguages(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		templates.LanguagesPage(nil, "", "", err.Error()).Render(r.Context(), w)
		return
	}

	langs := r.Form["languages"]
	if langs == nil {
		langs = []string{}
	}
	tz := strings.TrimSpace(r.FormValue("timezone"))
	hours := strings.TrimSpace(r.FormValue("hours"))

	cfg, _ := h.settings.Load()
	if cfg == nil {
		cfg = &settings.Settings{}
	}
	cfg.Languages = langs
	cfg.Timezone = tz
	cfg.Hours = hours
	if err := h.settings.Save(cfg); err != nil {
		templates.LanguagesPage(langs, tz, hours, "Could not save: "+err.Error()).Render(r.Context(), w)
		return
	}

	http.Redirect(w, r, "/setup/first-staff", http.StatusSeeOther)
}

// ─── Step 3 — First staff profile ──────────────────────────────────────────

func (h *uiHandler) firstStaffPage(w http.ResponseWriter, r *http.Request) {
	templates.FirstStaffPage("", "", "").Render(r.Context(), w)
}

func (h *uiHandler) completeFirstStaff(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		templates.FirstStaffPage("", "", err.Error()).Render(r.Context(), w)
		return
	}
	label := strings.TrimSpace(r.FormValue("label"))
	role := strings.TrimSpace(r.FormValue("role"))
	if label == "" {
		templates.FirstStaffPage(label, role, "Please provide a profile name.").Render(r.Context(), w)
		return
	}

	// Slug-ish ID derived from the label. Staff.Store checks duplicates so
	// any collision will surface as an error the user can correct.
	id := slugify(label)

	m := &staff.Staff{
		ID:    id,
		Label: label,
	}
	// The current Staff struct has no "role/description" field; drop the
	// value silently for now. Once a Description field is added, persist it.
	_ = role

	if err := h.staff.Create(m); err != nil {
		// If ID clash, just continue to next step — the user already has staff.
		// Any other error is surfaced.
		if !strings.Contains(err.Error(), "already exists") {
			templates.FirstStaffPage(label, role, err.Error()).Render(r.Context(), w)
			return
		}
	}

	http.Redirect(w, r, "/setup/integrations", http.StatusSeeOther)
}

// slugify turns "Reception Desk" into "reception-desk".
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case r == ' ' || r == '-' || r == '_':
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := strings.TrimRight(b.String(), "-")
	if out == "" {
		return "staff"
	}
	return out
}
