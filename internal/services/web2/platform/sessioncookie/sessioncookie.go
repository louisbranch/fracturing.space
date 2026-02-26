// Package sessioncookie centralizes web2 session cookie behavior.
package sessioncookie

import (
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web2/platform/requestmeta"
)

// Name is the canonical web2 session cookie name.
const Name = "web2_session"

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
	if w == nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     Name,
		Value:    strings.TrimSpace(sessionID),
		Path:     "/",
		HttpOnly: true,
		Secure:   requestmeta.IsHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	})
}

// Clear expires the session cookie for the current request context.
func Clear(w http.ResponseWriter, r *http.Request) {
	if w == nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     Name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   requestmeta.IsHTTPS(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
