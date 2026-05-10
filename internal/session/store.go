package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const CookieName = "pm_session"

type contextKey struct{}

type entry struct {
	UserID    string
	CreatedAt time.Time
}

type Store struct {
	mu       sync.RWMutex
	sessions map[string]entry
}

func NewStore() *Store {
	return &Store{sessions: make(map[string]entry)}
}

func (s *Store) Create(userID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	id := hex.EncodeToString(b)
	s.mu.Lock()
	s.sessions[id] = entry{UserID: userID, CreatedAt: time.Now()}
	s.mu.Unlock()
	return id, nil
}

func (s *Store) Lookup(sessionID string) (userID string, ok bool) {
	s.mu.RLock()
	e, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	return e.UserID, ok
}

func (s *Store) Delete(sessionID string) {
	s.mu.Lock()
	delete(s.sessions, sessionID)
	s.mu.Unlock()
}

func (s *Store) SetCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 30,
	})
}

func (s *Store) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

func (s *Store) FromRequest(r *http.Request) (userID string, ok bool) {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return "", false
	}
	return s.Lookup(c.Value)
}

// WithUser stores the user ID in the request context.
func WithUser(r *http.Request, userID string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), contextKey{}, userID))
}

// CurrentUser retrieves the user ID from the request context.
func CurrentUser(r *http.Request) (string, bool) {
	v, ok := r.Context().Value(contextKey{}).(string)
	return v, ok && v != ""
}
