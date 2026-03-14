package publicauth

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

const recoveryRevealCookieName = "fs_recovery_reveal"

// recoveryRevealMode distinguishes why the one-time recovery code is being shown.
type recoveryRevealMode string

const (
	recoveryRevealModeSignup   recoveryRevealMode = "signup"
	recoveryRevealModeRecovery recoveryRevealMode = "recovery"
)

// recoveryRevealState carries the one-time recovery code between auth handlers.
type recoveryRevealState struct {
	Code      string             `json:"code"`
	PendingID string             `json:"pending_id,omitempty"`
	Mode      recoveryRevealMode `json:"mode"`
}

// writeRecoveryRevealState stores reveal state in a short-lived, path-scoped cookie.
func writeRecoveryRevealState(w http.ResponseWriter, r *http.Request, policy requestmeta.SchemePolicy, state recoveryRevealState) {
	if w == nil {
		return
	}
	normalized, ok := normalizeRecoveryRevealState(state)
	if !ok {
		return
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     recoveryRevealCookieName,
		Value:    base64.RawURLEncoding.EncodeToString(payload),
		Path:     routepath.LoginRecoveryCode,
		HttpOnly: true,
		Secure:   requestmeta.IsHTTPSWithPolicy(r, policy),
		SameSite: http.SameSiteLaxMode,
	})
}

// readRecoveryRevealState loads and validates the current one-time reveal cookie.
func readRecoveryRevealState(r *http.Request) (recoveryRevealState, bool) {
	if r == nil {
		return recoveryRevealState{}, false
	}
	cookie, err := r.Cookie(recoveryRevealCookieName)
	if err != nil || cookie == nil {
		return recoveryRevealState{}, false
	}
	raw := strings.TrimSpace(cookie.Value)
	if raw == "" {
		return recoveryRevealState{}, false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return recoveryRevealState{}, false
	}
	var state recoveryRevealState
	if err := json.Unmarshal(decoded, &state); err != nil {
		return recoveryRevealState{}, false
	}
	return normalizeRecoveryRevealState(state)
}

// clearRecoveryRevealState expires the one-time reveal cookie after use.
func clearRecoveryRevealState(w http.ResponseWriter, r *http.Request, policy requestmeta.SchemePolicy) {
	if w == nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     recoveryRevealCookieName,
		Value:    "",
		Path:     routepath.LoginRecoveryCode,
		HttpOnly: true,
		Secure:   requestmeta.IsHTTPSWithPolicy(r, policy),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// normalizeRecoveryRevealState trims transport fields and rejects invalid modes.
func normalizeRecoveryRevealState(state recoveryRevealState) (recoveryRevealState, bool) {
	state.Code = strings.TrimSpace(state.Code)
	state.PendingID = strings.TrimSpace(state.PendingID)
	switch state.Mode {
	case recoveryRevealModeSignup, recoveryRevealModeRecovery:
	default:
		return recoveryRevealState{}, false
	}
	if state.Code == "" {
		return recoveryRevealState{}, false
	}
	return state, true
}
