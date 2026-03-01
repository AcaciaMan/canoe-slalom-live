package handler

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"sync"
)

// SessionStore provides concurrent-safe session storage.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]bool
}

// NewSessionStore creates a new SessionStore.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]bool),
	}
}

// Set adds a session ID.
func (s *SessionStore) Set(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[id] = true
}

// Valid checks if a session ID exists.
func (s *SessionStore) Valid(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[id]
}

// generateSessionID creates a cryptographically random 32-byte hex-encoded session ID.
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// RequireAuth is middleware that protects routes with admin token authentication.
// If ADMIN_TOKEN is not set, all requests are allowed through.
// Otherwise, it checks for a valid session cookie or a matching ?token= query param.
func (d *Deps) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If no admin token configured, allow all requests (dev mode)
		if d.AdminToken == "" {
			next(w, r)
			return
		}

		// 1. Check for valid session cookie
		cookie, err := r.Cookie("admin_session")
		if err == nil && d.Sessions.Valid(cookie.Value) {
			next(w, r)
			return
		}

		// 2. Check for ?token= query param
		token := r.URL.Query().Get("token")
		if token == d.AdminToken {
			// Generate session ID and set cookie
			sessionID, err := generateSessionID()
			if err != nil {
				log.Printf("Error generating session ID: %v", err)
				d.renderError(w, 500, "Internal server error")
				return
			}
			d.Sessions.Set(sessionID)

			http.SetCookie(w, &http.Cookie{
				Name:     "admin_session",
				Value:    sessionID,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				MaxAge:   86400, // 24 hours
				Secure:   false, // false for localhost dev
			})

			// Redirect to same URL without ?token= param
			cleanURL := *r.URL
			q := cleanURL.Query()
			q.Del("token")
			cleanURL.RawQuery = q.Encode()
			http.Redirect(w, r, cleanURL.String(), http.StatusFound)
			return
		}

		// 3. No valid auth — 403
		d.renderError(w, 403, "Access Denied — Please use an authorized link to access the judge panel.")
	}
}
