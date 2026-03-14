package settings

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	flashnotice "github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/routeparam"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// settingsMainHeader centralizes this web behavior in one helper seam.
func settingsMainHeader(loc webtemplates.Localizer) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{Title: webtemplates.T(loc, "layout.settings")}
}

// redirectSettingsRoot centralizes this web behavior in one helper seam.
func (h handlers) redirectSettingsRoot(w http.ResponseWriter, r *http.Request) {
	target := h.availability.defaultPath()
	if target == "" {
		h.WriteNotFound(w, r)
		return
	}
	httpx.WriteRedirect(w, r, target)
}

// writeFlashNotice centralizes this web behavior in one helper seam.
func (h handlers) writeFlashNotice(w http.ResponseWriter, r *http.Request, notice flashnotice.Notice) {
	flashnotice.WriteWithPolicy(w, r, notice, h.flashMeta)
}

// writeJSONError writes a localized JSON error payload for settings endpoints.
func (h handlers) writeJSONError(w http.ResponseWriter, r *http.Request, err error) {
	loc, _ := h.PageLocalizer(w, r)
	_ = httpx.WriteJSONError(w, apperrors.HTTPStatus(err), webi18n.LocalizeError(loc, err))
}

// hasSameOriginProof reports whether this package condition is satisfied.
func (h handlers) hasSameOriginProof(r *http.Request) bool {
	return requestmeta.HasSameOriginProofWithPolicy(r, h.flashMeta)
}

// withCredentialID extracts the credential ID path param and delegates to fn,
// returning 404 when the param is missing.
func (h handlers) withCredentialID(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return routeparam.WithRequired("credentialID", h.WriteNotFound, fn)
}

// writeSettingsPage centralizes common settings page shell rendering.
func (h handlers) writeSettingsPage(
	w http.ResponseWriter,
	r *http.Request,
	loc webtemplates.Localizer,
	statusCode int,
	activePath string,
	title string,
	body templ.Component,
) {
	layout := webtemplates.AppMainLayoutOptions{SideMenu: settingsSideMenu(activePath, loc, h.availability)}
	h.WritePage(w, r, title, statusCode, settingsMainHeader(loc), layout, body)
}

// loadAIKeyRows resolves settings AI key rows for template rendering.
func (h handlers) loadAIKeyRows(ctx context.Context, userID string) ([]webtemplates.SettingsAIKeyRow, error) {
	keys, err := h.aiKeys.ListAIKeys(ctx, userID)
	if err != nil {
		return nil, err
	}
	return mapAIKeyTemplateRows(keys), nil
}

// loadAIAgentCredentialOptions resolves active credential options for template rendering.
func (h handlers) loadAIAgentCredentialOptions(ctx context.Context, userID string) ([]webtemplates.SettingsAICredentialOption, error) {
	options, err := h.aiAgents.ListAIAgentCredentials(ctx, userID)
	if err != nil {
		return nil, err
	}
	return mapAIAgentCredentialTemplateOptions(options), nil
}

// loadAIAgentRows resolves settings AI agent rows for template rendering.
func (h handlers) loadAIAgentRows(ctx context.Context, userID string) ([]webtemplates.SettingsAIAgentRow, error) {
	agents, err := h.aiAgents.ListAIAgents(ctx, userID)
	if err != nil {
		return nil, err
	}
	return mapAIAgentTemplateRows(agents), nil
}
