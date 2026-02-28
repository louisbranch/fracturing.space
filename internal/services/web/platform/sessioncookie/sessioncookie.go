// Package sessioncookie centralizes web session cookie behavior.
package sessioncookie

import (
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// Name is the canonical web session cookie name.
const Name = "web_session"

// Read returns the trimmed session cookie value when present.
func Read(r *http.Request) (string, bool) {
	if r == nil {
		return "", false
	}
	cookie, err := r.Cookie(Name)
	if err != nil || cookie == nil {
		return "", false
	}
	value := strings.TrimSpace(cookie.Value)
	if value == "" {
		return "", false
	}
	return value, true
}

// Write sets the session cookie for the current request context.
func Write(w http.ResponseWriter, r *http.Request, sessionID string) {
	WriteWithPolicy(w, r, sessionID, requestmeta.SchemePolicy{})
}

// WriteWithPolicy sets the session cookie for the current request context.
func WriteWithPolicy(w http.ResponseWriter, r *http.Request, sessionID string, policy requestmeta.SchemePolicy) {
	if w == nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     Name,
		Value:    strings.TrimSpace(sessionID),
		Path:     "/",
		HttpOnly: true,
		Secure:   requestmeta.IsHTTPSWithPolicy(r, policy),
		SameSite: http.SameSiteLaxMode,
	})
}

// Clear expires the session cookie for the current request context.
func Clear(w http.ResponseWriter, r *http.Request) {
	ClearWithPolicy(w, r, requestmeta.SchemePolicy{})
}

// ClearWithPolicy expires the session cookie for the current request context.
func ClearWithPolicy(w http.ResponseWriter, r *http.Request, policy requestmeta.SchemePolicy) {
	if w == nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     Name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   requestmeta.IsHTTPSWithPolicy(r, policy),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
