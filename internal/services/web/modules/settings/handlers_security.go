package settings

import (
	"context"
	"net/http"

	flashnotice "github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// handleSecurityGet handles this route in the module transport layer.
func (h handlers) handleSecurityGet(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.RequestContextAndUserID(r)
	h.renderSecurityPage(w, r, ctx, userID, http.StatusOK)
}

// handleSecurityPasskeyStart handles this route in the module transport layer.
func (h handlers) handleSecurityPasskeyStart(w http.ResponseWriter, r *http.Request) {
	if !h.hasSameOriginProof(r) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	ctx, userID := h.RequestContextAndUserID(r)
	start, err := h.security.BeginPasskeyRegistration(ctx, userID)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	_ = httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"session_id": start.SessionID,
		"public_key": start.PublicKey,
	})
}

// handleSecurityPasskeyFinish handles this route in the module transport layer.
func (h handlers) handleSecurityPasskeyFinish(w http.ResponseWriter, r *http.Request) {
	if !h.hasSameOriginProof(r) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	input, err := parsePasskeyCredentialInput(r)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.security.FinishPasskeyRegistration(ctx, input.SessionID, input.Credential); err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	h.writeFlashNotice(w, r, flashnotice.NoticeSuccess("web.settings.security.notice_added"))
	_ = httpx.WriteJSON(w, http.StatusOK, map[string]any{"redirect_url": routepath.AppSettingsSecurity})
}

// renderSecurityPage centralizes this web behavior in one helper seam.
func (h handlers) renderSecurityPage(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string, statusCode int) {
	loc, _ := h.PageLocalizer(w, r)
	passkeys, err := h.loadPasskeyRows(ctx, userID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.writeSettingsPage(
		w,
		r,
		loc,
		statusCode,
		routepath.AppSettingsSecurity,
		webtemplates.T(loc, "web.settings.page_security_title"),
		SettingsSecurityFragment(passkeys, loc),
	)
}

// loadPasskeyRows keeps the security page mapper separate from passkey gateway calls.
func (h handlers) loadPasskeyRows(ctx context.Context, userID string) ([]SettingsPasskeyRow, error) {
	passkeys, err := h.security.ListPasskeys(ctx, userID)
	if err != nil {
		return nil, err
	}
	return mapPasskeyTemplateRows(passkeys), nil
}
