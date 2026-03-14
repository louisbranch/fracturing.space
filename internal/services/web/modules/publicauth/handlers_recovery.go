package publicauth

import (
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// handleRecoveryStart handles this route in the module transport layer.
func (h handlers) handleRecoveryStart(w http.ResponseWriter, r *http.Request) {
	input, err := parseRecoveryStartInput(r)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	start, err := h.service.RecoveryStart(r.Context(), input.Username, input.RecoveryCode)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	_ = httpx.WriteJSON(w, http.StatusOK, newRecoveryStartResponse(start))
}

// handleRecoveryFinish handles this route in the module transport layer.
func (h handlers) handleRecoveryFinish(w http.ResponseWriter, r *http.Request) {
	input, err := parseRecoveryFinishInput(r)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	finished, err := h.service.RecoveryFinish(r.Context(), input.RecoverySessionID, input.SessionID, input.Credential, input.PendingID)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	h.writeSessionCookie(w, r, finished.SessionID)
	h.writeRecoveryRevealState(w, r, recoveryRevealState{
		Code:      finished.RecoveryCode,
		PendingID: input.PendingID,
		Next:      input.NextPath,
		Mode:      recoveryRevealModeRecovery,
	})
	_ = httpx.WriteJSON(w, http.StatusOK, newPasskeyLoginFinishResponse(routepath.LoginRecoveryCode))
}

// handleRecoveryCodeGet handles this route in the module transport layer.
func (h handlers) handleRecoveryCodeGet(w http.ResponseWriter, r *http.Request) {
	state, ok := h.readAndClearRecoveryRevealState(w, r)
	if !ok {
		if h.redirectAuthenticatedToApp(w, r) {
			return
		}
		httpx.WriteRedirect(w, r, routepath.Login)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	langTag := h.resolveAuthTag(w, r)
	copy := webi18n.Auth(langTag)
	h.writeAuthPage(
		w,
		r,
		copy.RecoveryCodePageTitle,
		copy.MetaDescription,
		langTag.String(),
		RecoveryCodePage(RecoveryCodePageParams{
			Copy:       copy,
			Code:       state.Code,
			PendingID:  state.PendingID,
			Next:       state.Next,
			IsRecovery: state.Mode == recoveryRevealModeRecovery,
		}),
	)
}

// handleRecoveryCodeAcknowledge handles this route in the module transport layer.
func (h handlers) handleRecoveryCodeAcknowledge(w http.ResponseWriter, r *http.Request) {
	if !h.hasSameOriginProof(r) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(r.FormValue("acknowledged")) == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	httpx.WriteRedirect(w, r, h.service.ResolvePostAuthRedirect(strings.TrimSpace(r.FormValue("pending_id")), strings.TrimSpace(r.FormValue("next"))))
}

// writeRecoveryRevealState stores one-time recovery-code display state.
func (h handlers) writeRecoveryRevealState(w http.ResponseWriter, r *http.Request, state recoveryRevealState) {
	writeRecoveryRevealState(w, r, h.requestMeta, state)
}

// readAndClearRecoveryRevealState reads and clears one-time recovery-code display state.
func (h handlers) readAndClearRecoveryRevealState(w http.ResponseWriter, r *http.Request) (recoveryRevealState, bool) {
	state, ok := readRecoveryRevealState(r)
	if !ok {
		return recoveryRevealState{}, false
	}
	clearRecoveryRevealState(w, r, h.requestMeta)
	return state, true
}
