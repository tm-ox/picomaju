package license

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	PlanFree       = ""
	PlanCredits    = "credits"
	PlanStarter    = "starter"
	PlanPro        = "pro"
)

// License holds the local activation state issued after payment.
type License struct {
	Active           bool   `json:"active"`
	Plan             string `json:"plan,omitempty"`
	CreditsRemaining int    `json:"credits_remaining,omitempty"`
	Token            string `json:"token,omitempty"`
	ExpiresAt        int64  `json:"expires_at,omitempty"` // unix; 0 = no expiry (credits plan)
}

// IsActive returns true if the license is active and not expired.
func (l *License) IsActive() bool {
	if l == nil || !l.Active {
		return false
	}
	if l.ExpiresAt > 0 && time.Now().Unix() > l.ExpiresAt {
		return false
	}
	if l.Plan == PlanCredits {
		return l.CreditsRemaining > 0
	}
	return true
}

// PlanLabel returns a display string for the current plan.
func (l *License) PlanLabel() string {
	switch l.Plan {
	case PlanCredits:
		return "Credits"
	case PlanStarter:
		return "Starter"
	case PlanPro:
		return "Pro"
	default:
		return "Free"
	}
}

type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Load() (*License, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &License{}, nil
		}
		return nil, err
	}
	var l License
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

func (s *Store) Save(l *License) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0644)
}

// DeductCredit decrements credits_remaining by 1 and saves.
// Returns false if no credits remain.
func (s *Store) DeductCredit() (bool, error) {
	l, err := s.Load()
	if err != nil {
		return false, err
	}
	if l.Plan != PlanCredits || l.CreditsRemaining <= 0 {
		return false, nil
	}
	l.CreditsRemaining--
	if l.CreditsRemaining == 0 {
		l.Active = false
	}
	return true, s.Save(l)
}
