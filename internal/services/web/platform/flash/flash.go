// Package flash provides one-time web notices persisted across redirects.
package flash

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// CookieName is the canonical cookie used for one-time web notices.
const CookieName = "fs_flash"

// Kind classifies flash notice presentation.
type Kind string

const (
	KindSuccess Kind = "success"
	KindInfo    Kind = "info"
	KindWarning Kind = "warning"
	KindError   Kind = "error"
)

// Notice stores one flash message reference.
type Notice struct {
	Kind Kind   `json:"kind"`
	Key  string `json:"key"`
}

// NoticeSuccess creates a success notice for the provided localization key.
func NoticeSuccess(key string) Notice {
	return Notice{Kind: KindSuccess, Key: key}
}

// Write stores a flash notice cookie for the next page render.
func Write(w http.ResponseWriter, r *http.Request, notice Notice) {
	WriteWithPolicy(w, r, notice, requestmeta.SchemePolicy{})
}

// WriteWithPolicy stores a flash notice cookie for the next page render.
func WriteWithPolicy(w http.ResponseWriter, r *http.Request, notice Notice, policy requestmeta.SchemePolicy) {
	if w == nil {
		return
	}
	normalized, ok := normalizeNotice(notice)
	if !ok {
		return
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    base64.RawURLEncoding.EncodeToString(payload),
		Path:     "/",
		HttpOnly: true,
		Secure:   requestmeta.IsHTTPSWithPolicy(r, policy),
		SameSite: http.SameSiteLaxMode,
	})
}

// ReadAndClear reads and clears the flash notice cookie.
func ReadAndClear(w http.ResponseWriter, r *http.Request) (Notice, bool) {
	if r == nil {
		return Notice{}, false
	}
	cookie, err := r.Cookie(CookieName)
	if err != nil || cookie == nil {
		return Notice{}, false
	}
	if w != nil {
		Clear(w, r)
	}
	notice, ok := decodeNotice(cookie.Value)
	if !ok {
		return Notice{}, false
	}
	return notice, true
}

// Clear expires any flash notice cookie.
func Clear(w http.ResponseWriter, r *http.Request) {
	ClearWithPolicy(w, r, requestmeta.SchemePolicy{})
}

// ClearWithPolicy expires any flash notice cookie.
func ClearWithPolicy(w http.ResponseWriter, r *http.Request, policy requestmeta.SchemePolicy) {
	if w == nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   requestmeta.IsHTTPSWithPolicy(r, policy),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func decodeNotice(raw string) (Notice, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return Notice{}, false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return Notice{}, false
	}
	var notice Notice
	if err := json.Unmarshal(decoded, &notice); err != nil {
		return Notice{}, false
	}
	normalized, ok := normalizeNotice(notice)
	if !ok {
		return Notice{}, false
	}
	return normalized, true
}

func normalizeNotice(notice Notice) (Notice, bool) {
	notice.Key = strings.TrimSpace(notice.Key)
	if notice.Key == "" {
		return Notice{}, false
	}
	notice.Kind = Kind(strings.ToLower(strings.TrimSpace(string(notice.Kind))))
	switch notice.Kind {
	case KindSuccess, KindInfo, KindWarning, KindError:
		return notice, true
	default:
		return Notice{}, false
	}
}
