package web

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const sessionCookieName = "fs_session"

// tokenCookieName is the domain-scoped cookie carrying the raw access token,
// readable by sibling subdomains (e.g. admin.{DOMAIN}).
const tokenCookieName = "fs_token"

type sessionPersistence interface {
	LoadSession(ctx context.Context, sessionID string) (accessToken string, displayName string, expiresAt time.Time, found bool, err error)
	SaveSession(ctx context.Context, sessionID, accessToken, displayName string, expiresAt time.Time) error
	DeleteSession(ctx context.Context, sessionID string) error
}

// session holds one in-process authenticated session mapping token identity to UI
// display context for subsequent web requests.
type session struct {
	accessToken            string
	displayName            string
	expiresAt              time.Time
	accessTokenFingerprint string

	mu                     sync.RWMutex
	cachedUserID           string
	cachedUserIDResolved   bool
	cachedUserAvatarURL    string
	cachedUserAvatarCached bool
}

// sessionStore keeps a process-local cache of active sessions and optionally reads
// and writes a persisted copy for restart recovery.
//
// The in-memory map is authoritative while this process is live. Persisted rows are
// used only to recover sessions when the process restarts.
type sessionStore struct {
	mu          sync.RWMutex
	sessions    map[string]*session
	persistence sessionPersistence
	loads       map[string]chan struct{}
}

// newSessionStore creates the session cache for authenticated sessions.
// The in-memory map stays authoritative for the current process and optional
// persistence is used for restart recovery.
func newSessionStore(persistence ...sessionPersistence) *sessionStore {
	var store sessionPersistence
	if len(persistence) > 0 {
		store = persistence[0]
	}
	return &sessionStore{
		sessions:    make(map[string]*session),
		persistence: store,
		loads:       make(map[string]chan struct{}),
	}
}

// cachedUserID resolves from session-local cache when available.
func (s *session) cachedUserIDValue() (string, bool) {
	if s == nil {
		return "", false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.cachedUserIDResolved {
		return "", false
	}
	return s.cachedUserID, true
}

func (s *session) setCachedUserID(userID string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.cachedUserID = userID
	s.cachedUserIDResolved = true
	s.mu.Unlock()
}

// cachedUserAvatarURL resolves the cached avatar URL for this session.
func (s *session) cachedUserAvatar() (string, bool) {
	if s == nil {
		return "", false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.cachedUserAvatarCached {
		return "", false
	}
	return s.cachedUserAvatarURL, true
}

func (s *session) setCachedUserAvatar(avatarURL string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.cachedUserAvatarURL = avatarURL
	s.cachedUserAvatarCached = true
	s.mu.Unlock()
}

// create stores a new web session and returns an opaque session identifier.
// IDs are random and intentionally not reused so replaying old cookie values stays safe.
func (s *sessionStore) create(accessToken, displayName string, expiresAt time.Time) string {
	id := randomHex(16)
	accessTokenFingerprint := sessionAccessTokenFingerprint(accessToken)
	sess := &session{
		accessToken:            accessToken,
		displayName:            displayName,
		expiresAt:              expiresAt,
		accessTokenFingerprint: accessTokenFingerprint,
	}
	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()
	if s.persistence != nil {
		if err := s.persistence.SaveSession(context.Background(), id, accessToken, displayName, expiresAt); err != nil {
			log.Printf("web: failed to persist session %q: %v", id, err)
		}
	}
	return id
}

// get resolves a session by ID and prunes expired entries eagerly.
// Expired sessions reduce auth confusion when users leave long-running tabs open.
func (s *sessionStore) get(id string, requestAccessToken string) *session {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	requestAccessToken = strings.TrimSpace(requestAccessToken)

	for {
		s.mu.RLock()
		sess, ok := s.sessions[id]
		if ok {
			s.mu.RUnlock()
			return s.authorizeSession(id, sess, requestAccessToken)
		}
		if s.persistence == nil || requestAccessToken == "" {
			s.mu.RUnlock()
			return nil
		}
		if done, loading := s.loads[id]; loading {
			s.mu.RUnlock()
			<-done
			continue
		}
		s.mu.RUnlock()

		done := make(chan struct{})
		s.mu.Lock()
		if existing, exists := s.sessions[id]; exists {
			s.mu.Unlock()
			return s.authorizeSession(id, existing, requestAccessToken)
		}
		if done, loading := s.loads[id]; loading {
			s.mu.Unlock()
			<-done
			continue
		}
		s.loads[id] = done
		s.mu.Unlock()

		persistedAccessTokenFingerprint, displayName, expiresAt, found, err := s.persistence.LoadSession(context.Background(), id)
		if err != nil || !found {
			s.finishSessionLoad(id, done)
			return nil
		}
		if time.Now().After(expiresAt) {
			s.delete(id)
			s.finishSessionLoad(id, done)
			return nil
		}
		requestAccessTokenFingerprint := sessionAccessTokenFingerprint(requestAccessToken)
		if requestAccessTokenFingerprint != persistedAccessTokenFingerprint {
			s.delete(id)
			s.finishSessionLoad(id, done)
			return nil
		}
		sess = &session{
			accessToken:            requestAccessToken,
			displayName:            displayName,
			accessTokenFingerprint: persistedAccessTokenFingerprint,
			expiresAt:              expiresAt,
		}
		s.mu.Lock()
		if existing, exists := s.sessions[id]; exists {
			sess = existing
		} else {
			s.sessions[id] = sess
		}
		s.mu.Unlock()
		s.finishSessionLoad(id, done)

		return s.authorizeSession(id, sess, requestAccessToken)
	}
}

func (s *sessionStore) authorizeSession(id string, sess *session, requestAccessToken string) *session {
	if sess == nil {
		return nil
	}
	if requestAccessToken != "" && sess.accessTokenFingerprint != "" {
		requestAccessTokenFingerprint := sessionAccessTokenFingerprint(requestAccessToken)
		if requestAccessTokenFingerprint != sess.accessTokenFingerprint {
			s.delete(id)
			return nil
		}
		if strings.TrimSpace(sess.accessToken) == "" {
			sess.accessToken = requestAccessToken
		}
	}
	if time.Now().After(sess.expiresAt) {
		s.delete(id)
		return nil
	}
	return sess
}

func (s *sessionStore) finishSessionLoad(id string, done chan struct{}) {
	s.mu.Lock()
	delete(s.loads, id)
	s.mu.Unlock()
	close(done)
}

// delete removes session state when sign-out or rotation occurs.
func (s *sessionStore) delete(id string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
	if s.persistence == nil {
		return
	}
	if err := s.persistence.DeleteSession(context.Background(), id); err != nil {
		log.Printf("web: failed to delete persisted session %q: %v", id, err)
	}
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
	var tokenCookie string
	if token, err := r.Cookie(tokenCookieName); err == nil {
		tokenCookie = token.Value
	}
	return store.get(cookie.Value, tokenCookie)
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

func sessionAccessTokenFingerprint(accessToken string) string {
	accessToken = strings.TrimSpace(accessToken)
	sum := sha256.Sum256([]byte(accessToken))
	return hex.EncodeToString(sum[:])
}

// randomHex generates a cryptographically random hex string used as session state.
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
