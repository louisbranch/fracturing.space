package publicauth

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// maxJSONBodyBytes caps auth JSON payload size for passkey endpoint inputs.
const maxJSONBodyBytes = 64 << 10

// passkeyCredentialInput carries parsed session/credential fields.
type passkeyCredentialInput struct {
	SessionID  string          `json:"session_id"`
	Credential json.RawMessage `json:"credential"`
}

// passkeyRegisterStartInput carries parsed register-start fields.
type passkeyRegisterStartInput struct {
	Email string `json:"email"`
}

// parsePasskeyCredentialInput parses and normalizes passkey credential payloads.
func parsePasskeyCredentialInput(r *http.Request) (passkeyCredentialInput, error) {
	var payload passkeyCredentialInput
	if err := decodeJSONBodyStrict(r, &payload); err != nil {
		return passkeyCredentialInput{}, err
	}
	return passkeyCredentialInput{
		SessionID:  strings.TrimSpace(payload.SessionID),
		Credential: payload.Credential,
	}, nil
}

// parsePasskeyRegisterStartInput parses and normalizes passkey register-start input.
func parsePasskeyRegisterStartInput(r *http.Request) (passkeyRegisterStartInput, error) {
	var payload passkeyRegisterStartInput
	if err := decodeJSONBodyStrict(r, &payload); err != nil {
		return passkeyRegisterStartInput{}, err
	}
	return passkeyRegisterStartInput{Email: strings.TrimSpace(payload.Email)}, nil
}

// decodeJSONBodyStrict decodes one JSON object with strict field/size constraints.
func decodeJSONBodyStrict(r *http.Request, target any) error {
	if r == nil || r.Body == nil {
		return invalidJSONBodyError()
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxJSONBodyBytes+1))
	if err != nil {
		return invalidJSONBodyError()
	}
	if len(body) == 0 || len(body) > maxJSONBodyBytes {
		return invalidJSONBodyError()
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return invalidJSONBodyError()
	}
	// Reject trailing JSON tokens after the first payload.
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return invalidJSONBodyError()
	}
	return nil
}

// invalidJSONBodyError returns a stable invalid-input error for malformed JSON.
func invalidJSONBodyError() error {
	return apperrors.E(apperrors.KindInvalidInput, "invalid json body")
}
