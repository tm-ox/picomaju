package api

// Onboarding handlers for the welcome screen + 3-step setup wizard.

import (
	"net/http"
	"strings"

	"picomaju/internal/settings"
	"picomaju/internal/staff"
	setuptpl "picomaju/ui/templates/setup"
)

func (h *uiHandler) completeWelcome(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		setuptpl.WelcomePage("", err.Error()).Render(r.Context(), w)
		return
	}
	lang := strings.TrimSpace(r.FormValue("language"))
	if lang == "" {
		lang = "en"
	}

	cfg, _ := h.settings.Load()
	if cfg == nil {
		cfg = &settings.Settings{}
	}
	cfg.Languages = []string{lang}
	if err := h.settings.Save(cfg); err != nil {
		setuptpl.WelcomePage(lang, "Could not save: "+err.Error()).Render(r.Context(), w)
		return
	}

	http.Redirect(w, r, "/setup", http.StatusSeeOther)
}

// ─── Step 2 — First staff profile ──────────────────────────────────────────

func (h *uiHandler) firstStaffPage(w http.ResponseWriter, r *http.Request) {
	setuptpl.SetupStep2Page("", "", "").Render(r.Context(), w)
}

func (h *uiHandler) completeFirstStaff(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		setuptpl.SetupStep2Page("", "", err.Error()).Render(r.Context(), w)
		return
	}
	label := strings.TrimSpace(r.FormValue("label"))
	role := strings.TrimSpace(r.FormValue("role"))
	if label == "" {
		setuptpl.SetupStep2Page(label, role, "Please provide a profile name.").Render(r.Context(), w)
		return
	}

	id := slugify(label)

	m := &staff.Staff{
		ID:    id,
		Label: label,
	}
	_ = role

	if err := h.staff.Create(m); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			setuptpl.SetupStep2Page(label, role, err.Error()).Render(r.Context(), w)
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
