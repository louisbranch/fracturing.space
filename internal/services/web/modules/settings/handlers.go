package settings

import (
	"context"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	flashnotice "github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// settingsService defines the service contract used by settings handlers.
type settingsService = settingsapp.Service

// DashboardSync exposes dashboard cache refresh hooks needed by settings mutations.
type DashboardSync interface {
	ProfileSaved(context.Context, string)
}

// handlers defines an internal contract used at this web package boundary.
type handlers struct {
	modulehandler.Base
	service   settingsService
	flashMeta requestmeta.SchemePolicy
	sync      DashboardSync
}

// newHandlers builds package wiring for this web seam.
func newHandlers(s settingsService, base modulehandler.Base, policy requestmeta.SchemePolicy, sync DashboardSync) handlers {
	return handlers{Base: base, service: s, flashMeta: policy, sync: sync}
}

// settingsMainHeader centralizes this web behavior in one helper seam.
func settingsMainHeader(loc webtemplates.Localizer) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{Title: webtemplates.T(loc, "layout.settings")}
}

// redirectSettingsRoot centralizes this web behavior in one helper seam.
func (h handlers) redirectSettingsRoot(w http.ResponseWriter, r *http.Request) {
	httpx.WriteRedirect(w, r, routepath.AppSettingsProfile)
}

// handleProfileGet handles this route in the module transport layer.
func (h handlers) handleProfileGet(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.RequestContextAndUserID(r)
	profile, err := h.service.LoadProfile(ctx, userID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.renderProfilePage(w, r, http.StatusOK, profile, "")
}

// handleProfilePost handles this route in the module transport layer.
func (h handlers) handleProfilePost(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.RequestContextAndUserID(r)
	if err := r.ParseForm(); err != nil {
		h.WriteError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_parse_profile_form", "failed to parse profile form"))
		return
	}
	existingProfile, err := h.service.LoadProfile(ctx, userID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	profile := parseProfileInput(r.PostForm, existingProfile)
	if err := h.service.SaveProfile(ctx, userID, profile); err != nil {
		if apperrors.HTTPStatus(err) == http.StatusBadRequest {
			loc, _ := h.PageLocalizer(w, r)
			h.renderProfilePage(w, r, http.StatusBadRequest, profile, webi18n.LocalizeError(loc, err))
			return
		}
		h.WriteError(w, r, err)
		return
	}
	if h.sync != nil {
		h.sync.ProfileSaved(ctx, userID)
	}
	h.writeFlashNotice(w, r, flashnotice.NoticeSuccess("web.settings.user_profile.notice_saved"))
	httpx.WriteRedirect(w, r, routepath.AppSettingsProfile)
}

// handleLocaleGet handles this route in the module transport layer.
func (h handlers) handleLocaleGet(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.RequestContextAndUserID(r)
	locale, err := h.service.LoadLocale(ctx, userID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.renderLocalePage(w, r, http.StatusOK, locale, "")
}

// handleLocalePost handles this route in the module transport layer.
func (h handlers) handleLocalePost(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.RequestContextAndUserID(r)
	if err := r.ParseForm(); err != nil {
		h.WriteError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_parse_locale_form", "failed to parse locale form"))
		return
	}
	selectedLocale := parseLocaleInput(r.PostForm)
	if err := h.service.SaveLocale(ctx, userID, selectedLocale); err != nil {
		if apperrors.HTTPStatus(err) == http.StatusBadRequest {
			loc, _ := h.PageLocalizer(w, r)
			h.renderLocalePage(w, r, http.StatusBadRequest, selectedLocale, webi18n.LocalizeError(loc, err))
			return
		}
		h.WriteError(w, r, err)
		return
	}
	h.writeFlashNotice(w, r, flashnotice.NoticeSuccess("web.settings.locale.notice_saved"))
	httpx.WriteRedirect(w, r, routepath.AppSettingsLocale)
}

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
	start, err := h.service.BeginPasskeyRegistration(ctx, userID)
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
	if err := h.service.FinishPasskeyRegistration(ctx, input.SessionID, input.Credential); err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	h.writeFlashNotice(w, r, flashnotice.NoticeSuccess("web.settings.security.notice_added"))
	_ = httpx.WriteJSON(w, http.StatusOK, map[string]any{"redirect_url": routepath.AppSettingsSecurity})
}

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
	if err := h.service.CreateAIKey(ctx, userID, label, secret); err != nil {
		if apperrors.HTTPStatus(err) == http.StatusBadRequest {
			loc, _ := h.PageLocalizer(w, r)
			h.renderAIKeysPage(w, r, ctx, userID, http.StatusBadRequest, label, webi18n.LocalizeError(loc, err))
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
	if err := h.service.RevokeAIKey(ctx, userID, credentialID); err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.writeFlashNotice(w, r, flashnotice.NoticeSuccess("web.settings.ai_keys.notice_revoked"))
	httpx.WriteRedirect(w, r, routepath.AppSettingsAIKeys)
}

// handleAIAgentsGet handles this route in the module transport layer.
func (h handlers) handleAIAgentsGet(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.RequestContextAndUserID(r)
	form := webtemplates.SettingsAIAgentsForm{
		CredentialID: parseAIAgentCredentialSelectionInput(r.URL.Query()),
	}
	if form.CredentialID != "" {
		models, err := h.service.ListAIProviderModels(ctx, userID, form.CredentialID)
		if err != nil {
			statusCode := apperrors.HTTPStatus(err)
			if statusCode == http.StatusBadRequest || statusCode == http.StatusPreconditionFailed {
				loc, _ := h.PageLocalizer(w, r)
				h.renderAIAgentsPage(w, r, ctx, userID, statusCode, form, webi18n.LocalizeError(loc, err))
				return
			}
			h.WriteError(w, r, err)
			return
		}
		form.ModelOptions = mapAIModelTemplateOptions(models)
	}
	h.renderAIAgentsPage(w, r, ctx, userID, http.StatusOK, form, "")
}

// handleAIAgentsCreate handles this route in the module transport layer.
func (h handlers) handleAIAgentsCreate(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.RequestContextAndUserID(r)
	if err := r.ParseForm(); err != nil {
		h.WriteError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_parse_ai_agent_form", "failed to parse ai agent form"))
		return
	}
	input := parseAIAgentCreateInput(r.PostForm)
	if err := h.service.CreateAIAgent(ctx, userID, input); err != nil {
		statusCode := apperrors.HTTPStatus(err)
		if statusCode == http.StatusBadRequest || statusCode == http.StatusPreconditionFailed {
			loc, _ := h.PageLocalizer(w, r)
			form := webtemplates.SettingsAIAgentsForm{
				Name:         input.Name,
				CredentialID: input.CredentialID,
				Model:        input.Model,
				Instructions: input.Instructions,
			}
			if form.CredentialID != "" {
				models, modelErr := h.service.ListAIProviderModels(ctx, userID, form.CredentialID)
				if modelErr == nil {
					form.ModelOptions = mapAIModelTemplateOptions(models)
				}
			}
			h.renderAIAgentsPage(w, r, ctx, userID, statusCode, form, webi18n.LocalizeError(loc, err))
			return
		}
		h.WriteError(w, r, err)
		return
	}
	h.writeFlashNotice(w, r, flashnotice.NoticeSuccess("web.settings.ai_agents.notice_created"))
	httpx.WriteRedirect(w, r, routepath.AppSettingsAIAgents)
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

// routeCredentialID extracts the canonical settings credential route parameter.
func (h handlers) routeCredentialID(r *http.Request) (string, bool) {
	credentialID := strings.TrimSpace(r.PathValue("credentialID"))
	if credentialID == "" {
		return "", false
	}
	return credentialID, true
}

// withCredentialID extracts the credential ID path param and delegates to fn,
// returning 404 when the param is missing.
func (h handlers) withCredentialID(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		credentialID, ok := h.routeCredentialID(r)
		if !ok {
			h.WriteNotFound(w, r)
			return
		}
		fn(w, r, credentialID)
	}
}

// renderProfilePage centralizes this web behavior in one helper seam.
func (h handlers) renderProfilePage(w http.ResponseWriter, r *http.Request, statusCode int, profile SettingsProfile, errorMessage string) {
	loc, _ := h.PageLocalizer(w, r)
	h.writeSettingsPage(
		w,
		r,
		loc,
		statusCode,
		routepath.AppSettingsProfile,
		webtemplates.T(loc, "web.settings.page_profile_title"),
		webtemplates.SettingsProfileFragment(webtemplates.SettingsProfileForm{
			Username:      profile.Username,
			Name:          profile.Name,
			AvatarSetID:   profile.AvatarSetID,
			AvatarAssetID: profile.AvatarAssetID,
			Pronouns:      profile.Pronouns,
			Bio:           profile.Bio,
			ErrorMessage:  errorMessage,
		}, loc),
	)
}

// renderLocalePage centralizes this web behavior in one helper seam.
func (h handlers) renderLocalePage(w http.ResponseWriter, r *http.Request, statusCode int, selectedLocale string, errorMessage string) {
	loc, _ := h.PageLocalizer(w, r)
	h.writeSettingsPage(
		w,
		r,
		loc,
		statusCode,
		routepath.AppSettingsLocale,
		webtemplates.T(loc, "web.settings.page_locale_title"),
		webtemplates.SettingsLocaleFragment(webtemplates.SettingsLocaleForm{
			SelectedLocale: selectedLocale,
			ErrorMessage:   errorMessage,
		}, loc),
	)
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
		webtemplates.SettingsSecurityFragment(passkeys, loc),
	)
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

// renderAIAgentsPage centralizes this web behavior in one helper seam.
func (h handlers) renderAIAgentsPage(
	w http.ResponseWriter,
	r *http.Request,
	ctx context.Context,
	userID string,
	statusCode int,
	form webtemplates.SettingsAIAgentsForm,
	errorMessage string,
) {
	loc, _ := h.PageLocalizer(w, r)
	credentialOptions, err := h.loadAIAgentCredentialOptions(ctx, userID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	agentRows, err := h.loadAIAgentRows(ctx, userID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	form.ErrorMessage = errorMessage
	form.CredentialOptions = credentialOptions
	h.writeSettingsPage(
		w,
		r,
		loc,
		statusCode,
		routepath.AppSettingsAIAgents,
		webtemplates.T(loc, "web.settings.page_ai_agents_title"),
		webtemplates.SettingsAIAgentsFragment(form, agentRows, loc),
	)
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
	layout := webtemplates.AppMainLayoutOptions{SideMenu: settingsSideMenu(activePath, loc)}
	h.WritePage(w, r, title, statusCode, settingsMainHeader(loc), layout, body)
}

// loadAIKeyRows resolves settings AI key rows for template rendering.
func (h handlers) loadAIKeyRows(ctx context.Context, userID string) ([]webtemplates.SettingsAIKeyRow, error) {
	keys, err := h.service.ListAIKeys(ctx, userID)
	if err != nil {
		return nil, err
	}
	return mapAIKeyTemplateRows(keys), nil
}

// loadPasskeyRows resolves settings passkey rows for template rendering.
func (h handlers) loadPasskeyRows(ctx context.Context, userID string) ([]webtemplates.SettingsPasskeyRow, error) {
	passkeys, err := h.service.ListPasskeys(ctx, userID)
	if err != nil {
		return nil, err
	}
	return mapPasskeyTemplateRows(passkeys), nil
}

// loadAIAgentCredentialOptions resolves active credential options for template rendering.
func (h handlers) loadAIAgentCredentialOptions(ctx context.Context, userID string) ([]webtemplates.SettingsAICredentialOption, error) {
	options, err := h.service.ListAIAgentCredentials(ctx, userID)
	if err != nil {
		return nil, err
	}
	return mapAIAgentCredentialTemplateOptions(options), nil
}

// loadAIAgentRows resolves settings AI agent rows for template rendering.
func (h handlers) loadAIAgentRows(ctx context.Context, userID string) ([]webtemplates.SettingsAIAgentRow, error) {
	agents, err := h.service.ListAIAgents(ctx, userID)
	if err != nil {
		return nil, err
	}
	return mapAIAgentTemplateRows(agents), nil
}
