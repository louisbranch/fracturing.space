package publicauth

import (
	"encoding/json"

	publicauthapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// passkeyChallengeResponse defines the JSON contract returned by passkey
// challenge endpoints.
type passkeyChallengeResponse struct {
	SessionID string          `json:"session_id"`
	PublicKey json.RawMessage `json:"public_key"`
}

// passkeyLoginFinishResponse defines the JSON contract returned after login.
type passkeyLoginFinishResponse struct {
	RedirectURL string `json:"redirect_url"`
}

// passkeyRegisterFinishResponse defines the JSON contract returned after account
// registration and initial sign-in.
type passkeyRegisterFinishResponse struct {
	UserID      string `json:"user_id"`
	RedirectURL string `json:"redirect_url"`
}

// recoveryStartResponse defines the JSON contract returned when recovery begins.
type recoveryStartResponse struct {
	RecoverySessionID string          `json:"recovery_session_id"`
	SessionID         string          `json:"session_id"`
	PublicKey         json.RawMessage `json:"public_key"`
}

// usernameAvailabilityResponse defines the JSON contract for signup checks.
type usernameAvailabilityResponse struct {
	CanonicalUsername string `json:"canonical_username"`
	State             string `json:"state"`
}

// newPasskeyChallengeResponse maps app-layer challenge data to the transport contract.
func newPasskeyChallengeResponse(challenge publicauthapp.PasskeyChallenge) passkeyChallengeResponse {
	return passkeyChallengeResponse{
		SessionID: challenge.SessionID,
		PublicKey: challenge.PublicKey,
	}
}

// newPasskeyRegisterStartResponse maps app-layer register-start data to the transport contract.
func newPasskeyRegisterStartResponse(result publicauthapp.PasskeyRegisterResult) passkeyChallengeResponse {
	return passkeyChallengeResponse{
		SessionID: result.SessionID,
		PublicKey: result.PublicKey,
	}
}

// newPasskeyLoginFinishResponse maps the resolved post-auth redirect contract.
func newPasskeyLoginFinishResponse(redirectURL string) passkeyLoginFinishResponse {
	return passkeyLoginFinishResponse{RedirectURL: redirectURL}
}

// newPasskeyRegisterFinishResponse maps app-layer registration completion data
// to the transport contract.
func newPasskeyRegisterFinishResponse(finished publicauthapp.PasskeyFinish) passkeyRegisterFinishResponse {
	return passkeyRegisterFinishResponse{
		UserID:      finished.UserID,
		RedirectURL: routepath.LoginRecoveryCode,
	}
}

// newRecoveryStartResponse maps recovery enrollment begin state to the transport contract.
func newRecoveryStartResponse(challenge publicauthapp.RecoveryChallenge) recoveryStartResponse {
	return recoveryStartResponse{
		RecoverySessionID: challenge.RecoverySessionID,
		SessionID:         challenge.SessionID,
		PublicKey:         challenge.PublicKey,
	}
}

// newUsernameAvailabilityResponse maps signup validation state to transport JSON.
func newUsernameAvailabilityResponse(result publicauthapp.UsernameAvailability) usernameAvailabilityResponse {
	return usernameAvailabilityResponse{
		CanonicalUsername: result.CanonicalUsername,
		State:             string(result.State),
	}
}
