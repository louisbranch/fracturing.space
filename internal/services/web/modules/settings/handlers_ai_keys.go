package settings

import (
	"context"
	"net/http"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	flashnotice "github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// handleAIKeysGet handles this route in the module transport layer.
func (h handlers) handleAIKeysGet(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.RequestContextAndUserID(r)
	h.renderAIKeysPage(w, r, ctx, userID, http.StatusOK, "", "")
}

// handleAIKeysCreate handles this route in the module transport layer.
func (h handlers) handleAIKeysCreate(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.RequestContextAndUserID(r)
	if err := r.ParseForm(); err != nil {
		h.WriteError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_parse_ai_key_form", "failed to parse ai key form"))
		return
	}
	label, secret := parseAIKeyCreateInput(r.PostForm)
	if err := h.aiKeys.CreateAIKey(ctx, userID, label, secret); err != nil {
		statusCode := apperrors.HTTPStatus(err)
		if statusCode == http.StatusBadRequest || statusCode == http.StatusConflict {
			loc, lang := h.PageLocalizer(w, r)
			h.renderAIKeysPage(w, r, ctx, userID, statusCode, label, webi18n.LocalizeError(loc, err, lang))
			return
		}
		h.WriteError(w, r, err)
		return
	}
	h.writeFlashNotice(w, r, flashnotice.NoticeSuccess("web.settings.ai_keys.notice_created"))
	httpx.WriteRedirect(w, r, routepath.AppSettingsAIKeys)
}

// handleAIKeyRevoke handles this route in the module transport layer.
func (h handlers) handleAIKeyRevoke(w http.ResponseWriter, r *http.Request, credentialID string) {
	ctx, userID := h.RequestContextAndUserID(r)
	if err := h.aiKeys.RevokeAIKey(ctx, userID, credentialID); err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.writeFlashNotice(w, r, flashnotice.NoticeSuccess("web.settings.ai_keys.notice_revoked"))
	httpx.WriteRedirect(w, r, routepath.AppSettingsAIKeys)
}

// renderAIKeysPage centralizes this web behavior in one helper seam.
func (h handlers) renderAIKeysPage(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string, statusCode int, label string, errorMessage string) {
	loc, _ := h.PageLocalizer(w, r)
	rows, err := h.loadAIKeyRows(ctx, userID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.writeSettingsPage(
		w,
		r,
		loc,
		statusCode,
		routepath.AppSettingsAIKeys,
		webtemplates.T(loc, "web.settings.page_ai_keys_title"),
		webtemplates.SettingsAIKeysFragment(webtemplates.SettingsAIKeysForm{
			Label:        label,
			Provider:     "OpenAI",
			ErrorMessage: errorMessage,
		}, rows, loc),
	)
}
