package publicauth

import (
	"net/http"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/weberror"
)

// handlePasskeyLoginStart handles this route in the module transport layer.
func (h handlers) handlePasskeyLoginStart(w http.ResponseWriter, r *http.Request) {
	input, err := parsePasskeyLoginStartInput(r)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	start, err := h.passkeys.PasskeyLoginStart(r.Context(), input.Username)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	_ = httpx.WriteJSON(w, http.StatusOK, newPasskeyChallengeResponse(start))
}

// handlePasskeyLoginFinish handles this route in the module transport layer.
func (h handlers) handlePasskeyLoginFinish(w http.ResponseWriter, r *http.Request) {
	input, err := parsePasskeyCredentialInput(r)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	finished, err := h.passkeys.PasskeyLoginFinish(r.Context(), input.SessionID, input.Credential, input.PendingID)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	h.writeSessionCookie(w, r, finished.SessionID)
	_ = httpx.WriteJSON(w, http.StatusOK, newPasskeyLoginFinishResponse(h.session.ResolvePostAuthRedirect(input.PendingID, input.NextPath)))
}

// handlePasskeyRegisterStart handles this route in the module transport layer.
func (h handlers) handlePasskeyRegisterStart(w http.ResponseWriter, r *http.Request) {
	input, err := parsePasskeyRegisterStartInput(r)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	start, err := h.passkeys.PasskeyRegisterStart(r.Context(), input.Username)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	_ = httpx.WriteJSON(w, http.StatusOK, newPasskeyRegisterStartResponse(start))
}

// handlePasskeyRegisterFinish handles this route in the module transport layer.
func (h handlers) handlePasskeyRegisterFinish(w http.ResponseWriter, r *http.Request) {
	input, err := parsePasskeyCredentialInput(r)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	finished, err := h.passkeys.PasskeyRegisterFinish(r.Context(), input.SessionID, input.Credential)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	h.writeRecoveryRevealState(w, r, recoveryRevealState{
		Code:      finished.RecoveryCode,
		SessionID: input.SessionID,
		PendingID: input.PendingID,
		Next:      input.NextPath,
		Mode:      recoveryRevealModeSignup,
	})
	_ = httpx.WriteJSON(w, http.StatusOK, newPasskeyRegisterFinishResponse(finished))
}

// handleUsernameCheck handles this route in the module transport layer.
func (h handlers) handleUsernameCheck(w http.ResponseWriter, r *http.Request) {
	input, err := parseUsernameCheckInput(r)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	result, err := h.passkeys.CheckUsernameAvailability(r.Context(), input.Username)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	_ = httpx.WriteJSON(w, http.StatusOK, newUsernameAvailabilityResponse(result))
}

// writeJSONError centralizes this web behavior in one helper seam.
func (h handlers) writeJSONError(w http.ResponseWriter, r *http.Request, err error) {
	page := requestresolver.ResolveLocalizedPage(w, r, nil)
	_ = httpx.WriteJSONError(w, apperrors.HTTPStatus(err), weberror.PublicMessage(page.Localizer, err, page.Language))
}
