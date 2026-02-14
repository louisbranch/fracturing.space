package web

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const sessionCookieName = "fs_session"

// tokenCookieName is the domain-scoped cookie carrying the raw access token,
// readable by sibling subdomains (e.g. admin.{DOMAIN}).
const tokenCookieName = "fs_token"

// session holds data for an authenticated web session.
type session struct {
	accessToken string
	displayName string
	expiresAt   time.Time
}

// sessionStore is a thread-safe in-memory session store.
type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*session
}

// newSessionStore creates an empty session store.
func newSessionStore() *sessionStore {
	return &sessionStore{sessions: make(map[string]*session)}
}

// create stores a new session and returns its ID.
func (s *sessionStore) create(accessToken, displayName string, expiresAt time.Time) string {
	id := randomHex(16)
	s.mu.Lock()
	s.sessions[id] = &session{
		accessToken: accessToken,
		displayName: displayName,
		expiresAt:   expiresAt,
	}
	s.mu.Unlock()
	return id
}

// get returns a session by ID, or nil if missing or expired.
func (s *sessionStore) get(id string) *session {
	s.mu.RLock()
	sess, ok := s.sessions[id]
	s.mu.RUnlock()
	if !ok {
		return nil
	}
	if time.Now().After(sess.expiresAt) {
		s.delete(id)
		return nil
	}
	return sess
}

// delete removes a session by ID.
func (s *sessionStore) delete(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

// pendingFlow holds PKCE state for an in-flight OAuth login.
type pendingFlow struct {
	codeVerifier string
	createdAt    time.Time
}

// pendingFlowStore is a thread-safe store for in-flight PKCE flows.
type pendingFlowStore struct {
	mu    sync.Mutex
	flows map[string]*pendingFlow
	ttl   time.Duration
}

// newPendingFlowStore creates an empty pending flow store with a 10-minute TTL.
func newPendingFlowStore() *pendingFlowStore {
	return &pendingFlowStore{
		flows: make(map[string]*pendingFlow),
		ttl:   10 * time.Minute,
	}
}

// create stores a new pending flow and returns the state parameter.
func (s *pendingFlowStore) create(codeVerifier string) string {
	state := randomHex(16)
	s.mu.Lock()
	s.flows[state] = &pendingFlow{
		codeVerifier: codeVerifier,
		createdAt:    time.Now(),
	}
	s.mu.Unlock()
	return state
}

// consume retrieves and removes a pending flow by state.
// Returns nil if missing or expired.
func (s *pendingFlowStore) consume(state string) *pendingFlow {
	s.mu.Lock()
	flow, ok := s.flows[state]
	if ok {
		delete(s.flows, state)
	}
	s.mu.Unlock()
	if !ok {
		return nil
	}
	if time.Since(flow.createdAt) > s.ttl {
		return nil
	}
	return flow
}

// setSessionCookie writes the session cookie to the response.
func setSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearSessionCookie expires the session cookie.
func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// sessionFromRequest reads the session cookie and looks up the session.
func sessionFromRequest(r *http.Request, store *sessionStore) *session {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil
	}
	return store.get(cookie.Value)
}

// setTokenCookie writes a domain-scoped access token cookie to the response.
// The Domain attribute allows subdomains (e.g. admin.{DOMAIN}) to read it.
func setTokenCookie(w http.ResponseWriter, token, domain string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     tokenCookieName,
		Value:    token,
		Path:     "/",
		Domain:   domain,
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearTokenCookie expires the domain-scoped token cookie.
func clearTokenCookie(w http.ResponseWriter, domain string) {
	http.SetCookie(w, &http.Cookie{
		Name:     tokenCookieName,
		Value:    "",
		Path:     "/",
		Domain:   domain,
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// randomHex generates a cryptographically random hex string of n bytes.
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
