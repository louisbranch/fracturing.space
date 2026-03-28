package publicauth

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
)

// maxJSONBodyBytes caps auth JSON payload size for passkey endpoint inputs.
const maxJSONBodyBytes = 64 << 10

// passkeyCredentialInput carries parsed session/credential fields.
type passkeyCredentialInput struct {
	SessionID  string          `json:"session_id"`
	PendingID  string          `json:"pending_id,omitempty"`
	NextPath   string          `json:"next,omitempty"`
	Credential json.RawMessage `json:"credential"`
}

// passkeyRegisterStartInput carries parsed register-start fields.
type passkeyRegisterStartInput struct {
	Username string `json:"username"`
}

// usernameCheckInput carries parsed username-check fields.
type usernameCheckInput struct {
	Username string `json:"username"`
}

// passkeyLoginStartInput carries parsed login-start fields.
type passkeyLoginStartInput struct {
	Username string `json:"username"`
}

// recoveryStartInput carries parsed recovery-start fields.
type recoveryStartInput struct {
	Username     string `json:"username"`
	RecoveryCode string `json:"recovery_code"`
}

// recoveryFinishInput carries parsed recovery finish fields.
type recoveryFinishInput struct {
	RecoverySessionID string          `json:"recovery_session_id"`
	SessionID         string          `json:"session_id"`
	PendingID         string          `json:"pending_id,omitempty"`
	NextPath          string          `json:"next,omitempty"`
	Credential        json.RawMessage `json:"credential"`
}

// parsePasskeyCredentialInput decodes a passkey credential payload.
func parsePasskeyCredentialInput(r *http.Request) (passkeyCredentialInput, error) {
	var payload passkeyCredentialInput
	if err := decodeJSONBodyStrict(r, &payload); err != nil {
		return passkeyCredentialInput{}, err
	}
	return payload, nil
}

// parsePasskeyRegisterStartInput decodes passkey register-start payloads.
func parsePasskeyRegisterStartInput(r *http.Request) (passkeyRegisterStartInput, error) {
	var payload passkeyRegisterStartInput
	if err := decodeJSONBodyStrict(r, &payload); err != nil {
		return passkeyRegisterStartInput{}, err
	}
	return payload, nil
}

// parseUsernameCheckInput decodes username availability payloads.
func parseUsernameCheckInput(r *http.Request) (usernameCheckInput, error) {
	var payload usernameCheckInput
	if err := decodeJSONBodyStrict(r, &payload); err != nil {
		return usernameCheckInput{}, err
	}
	return payload, nil
}

// parsePasskeyLoginStartInput decodes passkey login-start payloads.
func parsePasskeyLoginStartInput(r *http.Request) (passkeyLoginStartInput, error) {
	var payload passkeyLoginStartInput
	if err := decodeJSONBodyStrict(r, &payload); err != nil {
		return passkeyLoginStartInput{}, err
	}
	return payload, nil
}

// parseRecoveryStartInput decodes recovery-start payloads.
func parseRecoveryStartInput(r *http.Request) (recoveryStartInput, error) {
	var payload recoveryStartInput
	if err := decodeJSONBodyStrict(r, &payload); err != nil {
		return recoveryStartInput{}, err
	}
	return payload, nil
}

// parseRecoveryFinishInput decodes recovery-finish payloads and trims next path.
func parseRecoveryFinishInput(r *http.Request) (recoveryFinishInput, error) {
	var payload recoveryFinishInput
	if err := decodeJSONBodyStrict(r, &payload); err != nil {
		return recoveryFinishInput{}, err
	}
	return recoveryFinishInput{
		RecoverySessionID: payload.RecoverySessionID,
		SessionID:         payload.SessionID,
		PendingID:         payload.PendingID,
		NextPath:          strings.TrimSpace(payload.NextPath),
		Credential:        payload.Credential,
	}, nil
}

// decodeJSONBodyStrict decodes one JSON object with strict field/size constraints.
func decodeJSONBodyStrict(r *http.Request, target any) error {
	return httpx.DecodeJSONStrictInvalidInput(r, target, maxJSONBodyBytes)
}
