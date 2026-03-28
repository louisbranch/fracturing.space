package app

import (
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/shared/httpx"
)

const playSessionCookieName = "play_session"

func readPlaySessionCookie(r *http.Request) (string, bool) {
	if r == nil {
		return "", false
	}
	cookie, err := r.Cookie(playSessionCookieName)
	if err != nil || cookie == nil {
		return "", false
	}
	value := strings.TrimSpace(cookie.Value)
	if value == "" {
		return "", false
	}
	return value, true
}

func writePlaySessionCookie(w http.ResponseWriter, r *http.Request, sessionID string, policy httpx.SchemePolicy) {
	if w == nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     playSessionCookieName,
		Value:    strings.TrimSpace(sessionID),
		Path:     "/",
		HttpOnly: true,
		Secure:   httpx.IsHTTPSWithPolicy(r, policy),
		SameSite: http.SameSiteLaxMode,
	})
}

func clearPlaySessionCookie(w http.ResponseWriter, r *http.Request, policy httpx.SchemePolicy) {
	if w == nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     playSessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   httpx.IsHTTPSWithPolicy(r, policy),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
