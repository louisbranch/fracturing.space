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

// session holds one in-process authenticated session mapping token identity to UI
// display context for subsequent web requests.
type session struct {
	accessToken string
	displayName string
	expiresAt   time.Time
}

// sessionStore stores ephemeral web sessions for this process.
//
// The map is intentionally local to the web process because the session token is
// already authoritative and persisted in the signed access token cookie.
type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*session
}

// newSessionStore creates the in-memory cache for authenticated sessions.
func newSessionStore() *sessionStore {
	return &sessionStore{sessions: make(map[string]*session)}
}

// create stores a new web session and returns an opaque session identifier.
// IDs are random and intentionally not reused so replaying old cookie values stays safe.
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

// get resolves a session by ID and prunes expired entries eagerly.
// Expired sessions reduce auth confusion when users leave long-running tabs open.
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

// delete removes session state when sign-out or rotation occurs.
func (s *sessionStore) delete(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

// pendingFlow tracks OAuth PKCE state for one in-flight login/registration flow.
type pendingFlow struct {
	codeVerifier string
	createdAt    time.Time
}

// pendingFlowStore is a thread-safe cache of temporary OAuth PKCE state.
//
// Entries are short-lived by design because they only need to bridge browser auth
// redirects to token exchange and should not become a replay surface.
type pendingFlowStore struct {
	mu    sync.Mutex
	flows map[string]*pendingFlow
	ttl   time.Duration
}

// newPendingFlowStore creates a PKCE flow store with a short-lived default TTL.
// A small TTL keeps open OAuth handshakes from staying valid after user abandonment.
func newPendingFlowStore() *pendingFlowStore {
	return &pendingFlowStore{
		flows: make(map[string]*pendingFlow),
		ttl:   10 * time.Minute,
	}
}

// create stores a new pending flow and returns the state token.
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
// Returns nil if missing, already consumed, or expired.
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

// setSessionCookie writes the session cookie that maps browser requests to this
// process-specific session map.
func setSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearSessionCookie expires the session cookie, forcing re-auth on subsequent requests.
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

// sessionFromRequest resolves request-local session state from the session cookie.
// This is the first auth gate used by almost all app routes.
func sessionFromRequest(r *http.Request, store *sessionStore) *session {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil
	}
	return store.get(cookie.Value)
}

// setTokenCookie writes an access-token cookie for sibling web surfaces.
// The domain scope is intentional so admin and game web pages can share one session.
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

// clearTokenCookie expires token-bearing cookie material across related hosts.
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

// randomHex generates a cryptographically random hex string used as session state.
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
