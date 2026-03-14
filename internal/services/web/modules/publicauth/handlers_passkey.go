package publicauth

import (
	"net/http"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/weberror"
)

// handlePasskeyLoginStart handles this route in the module transport layer.
func (h handlers) handlePasskeyLoginStart(w http.ResponseWriter, r *http.Request) {
	input, err := parsePasskeyLoginStartInput(r)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	start, err := h.service.PasskeyLoginStart(r.Context(), input.Username)
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
	finished, err := h.service.PasskeyLoginFinish(r.Context(), input.SessionID, input.Credential, input.PendingID)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	h.writeSessionCookie(w, r, finished.SessionID)
	_ = httpx.WriteJSON(w, http.StatusOK, newPasskeyLoginFinishResponse(h.service.ResolvePostAuthRedirect(input.PendingID)))
}

// handlePasskeyRegisterStart handles this route in the module transport layer.
func (h handlers) handlePasskeyRegisterStart(w http.ResponseWriter, r *http.Request) {
	input, err := parsePasskeyRegisterStartInput(r)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	start, err := h.service.PasskeyRegisterStart(r.Context(), input.Username)
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
	finished, err := h.service.PasskeyRegisterFinish(r.Context(), input.SessionID, input.Credential)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	h.writeSessionCookie(w, r, finished.SessionID)
	h.writeRecoveryRevealState(w, r, recoveryRevealState{
		Code: finished.RecoveryCode,
		Mode: recoveryRevealModeSignup,
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
	result, err := h.service.CheckUsernameAvailability(r.Context(), input.Username)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	_ = httpx.WriteJSON(w, http.StatusOK, newUsernameAvailabilityResponse(result))
}

// writeJSONError centralizes this web behavior in one helper seam.
func (h handlers) writeJSONError(w http.ResponseWriter, r *http.Request, err error) {
	loc, lang := webi18n.ResolveLocalizer(w, r, nil)
	_ = httpx.WriteJSONError(w, apperrors.HTTPStatus(err), weberror.PublicMessage(loc, err, lang))
}
