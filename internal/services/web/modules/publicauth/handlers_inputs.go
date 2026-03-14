package publicauth

import (
	"encoding/json"
	"net/http"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/jsoninput"
)

// maxJSONBodyBytes caps auth JSON payload size for passkey endpoint inputs.
const maxJSONBodyBytes = 64 << 10

// passkeyCredentialInput carries parsed session/credential fields.
type passkeyCredentialInput struct {
	SessionID  string          `json:"session_id"`
	PendingID  string          `json:"pending_id,omitempty"`
	Credential json.RawMessage `json:"credential"`
}

// passkeyRegisterStartInput carries parsed register-start fields.
type passkeyRegisterStartInput struct {
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
	Credential        json.RawMessage `json:"credential"`
}

// parsePasskeyCredentialInput parses and normalizes passkey credential payloads.
func parsePasskeyCredentialInput(r *http.Request) (passkeyCredentialInput, error) {
	var payload passkeyCredentialInput
	if err := decodeJSONBodyStrict(r, &payload); err != nil {
		return passkeyCredentialInput{}, err
	}
	return passkeyCredentialInput{
		SessionID:  strings.TrimSpace(payload.SessionID),
		PendingID:  strings.TrimSpace(payload.PendingID),
		Credential: payload.Credential,
	}, nil
}

// parsePasskeyRegisterStartInput parses and normalizes passkey register-start input.
func parsePasskeyRegisterStartInput(r *http.Request) (passkeyRegisterStartInput, error) {
	var payload passkeyRegisterStartInput
	if err := decodeJSONBodyStrict(r, &payload); err != nil {
		return passkeyRegisterStartInput{}, err
	}
	return passkeyRegisterStartInput{Username: strings.TrimSpace(payload.Username)}, nil
}

// parsePasskeyLoginStartInput parses and normalizes passkey login-start input.
func parsePasskeyLoginStartInput(r *http.Request) (passkeyLoginStartInput, error) {
	var payload passkeyLoginStartInput
	if err := decodeJSONBodyStrict(r, &payload); err != nil {
		return passkeyLoginStartInput{}, err
	}
	return passkeyLoginStartInput{Username: strings.TrimSpace(payload.Username)}, nil
}

// parseRecoveryStartInput parses and normalizes recovery start input.
func parseRecoveryStartInput(r *http.Request) (recoveryStartInput, error) {
	var payload recoveryStartInput
	if err := decodeJSONBodyStrict(r, &payload); err != nil {
		return recoveryStartInput{}, err
	}
	return recoveryStartInput{
		Username:     strings.TrimSpace(payload.Username),
		RecoveryCode: strings.TrimSpace(payload.RecoveryCode),
	}, nil
}

// parseRecoveryFinishInput parses and normalizes recovery finish input.
func parseRecoveryFinishInput(r *http.Request) (recoveryFinishInput, error) {
	var payload recoveryFinishInput
	if err := decodeJSONBodyStrict(r, &payload); err != nil {
		return recoveryFinishInput{}, err
	}
	return recoveryFinishInput{
		RecoverySessionID: strings.TrimSpace(payload.RecoverySessionID),
		SessionID:         strings.TrimSpace(payload.SessionID),
		PendingID:         strings.TrimSpace(payload.PendingID),
		Credential:        payload.Credential,
	}, nil
}

// decodeJSONBodyStrict decodes one JSON object with strict field/size constraints.
func decodeJSONBodyStrict(r *http.Request, target any) error {
	if r == nil || r.Body == nil {
		return invalidJSONBodyError()
	}
	if err := jsoninput.DecodeStrict(r, target, maxJSONBodyBytes); err != nil {
		return invalidJSONBodyError()
	}
	return nil
}

// invalidJSONBodyError returns a stable invalid-input error for malformed JSON.
func invalidJSONBodyError() error {
	return apperrors.E(apperrors.KindInvalidInput, "Invalid JSON body.")
}
