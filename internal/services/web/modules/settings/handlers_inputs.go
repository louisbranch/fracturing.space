package settings

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/jsoninput"
)

const maxJSONBodyBytes = 64 << 10

// passkeyCredentialInput captures one authenticated WebAuthn completion payload.
type passkeyCredentialInput struct {
	SessionID  string          `json:"session_id"`
	Credential json.RawMessage `json:"credential"`
}

// parseProfileInput maps profile form values and preserves avatar catalog IDs.
func parseProfileInput(form url.Values, existingProfile SettingsProfile) SettingsProfile {
	return SettingsProfile{
		Username:      strings.TrimSpace(existingProfile.Username),
		Name:          strings.TrimSpace(form.Get("name")),
		AvatarSetID:   existingProfile.AvatarSetID,
		AvatarAssetID: existingProfile.AvatarAssetID,
		Pronouns:      strings.TrimSpace(form.Get("pronouns")),
		Bio:           strings.TrimSpace(form.Get("bio")),
	}
}

// parseLocaleInput maps locale form values.
func parseLocaleInput(form url.Values) string {
	return strings.TrimSpace(form.Get("locale"))
}

// parseAIKeyCreateInput maps create-key form values.
func parseAIKeyCreateInput(form url.Values) (label string, secret string) {
	return strings.TrimSpace(form.Get("label")), strings.TrimSpace(form.Get("secret"))
}

// parseAIAgentCredentialSelectionInput maps the selected credential query value.
func parseAIAgentCredentialSelectionInput(values url.Values) string {
	return strings.TrimSpace(values.Get("credential_id"))
}

// parseAIAgentCreateInput maps create-agent form values.
func parseAIAgentCreateInput(form url.Values) CreateAIAgentInput {
	return CreateAIAgentInput{
		Name:         strings.TrimSpace(form.Get("name")),
		CredentialID: strings.TrimSpace(form.Get("credential_id")),
		Model:        strings.TrimSpace(form.Get("model")),
		Instructions: strings.TrimSpace(form.Get("instructions")),
	}
}

// parsePasskeyCredentialInput parses and normalizes one settings passkey credential payload.
func parsePasskeyCredentialInput(r *http.Request) (passkeyCredentialInput, error) {
	var payload passkeyCredentialInput
	if err := jsoninput.DecodeStrict(r, &payload, maxJSONBodyBytes); err != nil {
		return passkeyCredentialInput{}, apperrors.E(apperrors.KindInvalidInput, "Invalid JSON body.")
	}
	return passkeyCredentialInput{
		SessionID:  strings.TrimSpace(payload.SessionID),
		Credential: payload.Credential,
	}, nil
}
