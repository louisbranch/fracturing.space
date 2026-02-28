package settings

import (
	"context"
	"net/http"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	flashnotice "github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// settingsService defines the service operations used by settings handlers.
type settingsService interface {
	loadProfile(ctx context.Context, userID string) (SettingsProfile, error)
	saveProfile(ctx context.Context, userID string, profile SettingsProfile) error
	loadLocale(ctx context.Context, userID string) (string, error)
	saveLocale(ctx context.Context, userID string, value string) error
	listAIKeys(ctx context.Context, userID string) ([]SettingsAIKey, error)
	createAIKey(ctx context.Context, userID string, label string, secret string) error
	revokeAIKey(ctx context.Context, userID string, credentialID string) error
}

type handlers struct {
	modulehandler.Base
	service   settingsService
	flashMeta requestmeta.SchemePolicy
}

func newHandlers(s service, base modulehandler.Base, policy requestmeta.SchemePolicy) handlers {
	return handlers{Base: base, service: s, flashMeta: policy}
}

func settingsMainHeader(loc webtemplates.Localizer) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{Title: webtemplates.T(loc, "layout.settings")}
}

func (h handlers) redirectSettingsRoot(w http.ResponseWriter, r *http.Request) {
	httpx.WriteRedirect(w, r, routepath.AppSettingsProfile)
}

func (h handlers) handleProfileGet(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.RequestContextAndUserID(r)
	profile, err := h.service.loadProfile(ctx, userID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.renderProfilePage(w, r, http.StatusOK, profile, "", settingsProfileNoticeCode(r))
}

func (h handlers) handleProfilePost(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.RequestContextAndUserID(r)
	if err := r.ParseForm(); err != nil {
		h.WriteError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_parse_profile_form", "failed to parse profile form"))
		return
	}
	existingProfile, err := h.service.loadProfile(ctx, userID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	profile := SettingsProfile{
		Username:      strings.TrimSpace(r.FormValue("username")),
		Name:          strings.TrimSpace(r.FormValue("name")),
		AvatarSetID:   existingProfile.AvatarSetID,
		AvatarAssetID: existingProfile.AvatarAssetID,
		Pronouns:      strings.TrimSpace(r.FormValue("pronouns")),
		Bio:           strings.TrimSpace(r.FormValue("bio")),
	}
	if err := h.service.saveProfile(ctx, userID, profile); err != nil {
		if apperrors.HTTPStatus(err) == http.StatusBadRequest {
			loc, _ := h.PageLocalizer(w, r)
			h.renderProfilePage(w, r, http.StatusBadRequest, profile, webi18n.LocalizeError(loc, err), "")
			return
		}
		h.WriteError(w, r, err)
		return
	}
	h.writeFlashNotice(w, r, flashnotice.NoticeSuccess("web.settings.user_profile.notice_saved"))
	httpx.WriteRedirect(w, r, routepath.AppSettingsProfile)
}

func (h handlers) handleLocaleGet(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.RequestContextAndUserID(r)
	locale, err := h.service.loadLocale(ctx, userID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.renderLocalePage(w, r, http.StatusOK, locale, "")
}

func (h handlers) handleLocalePost(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.RequestContextAndUserID(r)
	if err := r.ParseForm(); err != nil {
		h.WriteError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_parse_locale_form", "failed to parse locale form"))
		return
	}
	selectedLocale := strings.TrimSpace(r.FormValue("locale"))
	if err := h.service.saveLocale(ctx, userID, selectedLocale); err != nil {
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

func (h handlers) handleAIKeysGet(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.RequestContextAndUserID(r)
	h.renderAIKeysPage(w, r, ctx, userID, http.StatusOK, "", "")
}

func (h handlers) handleAIKeysCreate(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.RequestContextAndUserID(r)
	if err := r.ParseForm(); err != nil {
		h.WriteError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_parse_ai_key_form", "failed to parse ai key form"))
		return
	}
	label := strings.TrimSpace(r.FormValue("label"))
	secret := strings.TrimSpace(r.FormValue("secret"))
	if err := h.service.createAIKey(ctx, userID, label, secret); err != nil {
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

func (h handlers) handleAIKeyRevoke(w http.ResponseWriter, r *http.Request, credentialID string) {
	ctx, userID := h.RequestContextAndUserID(r)
	if err := h.service.revokeAIKey(ctx, userID, credentialID); err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.writeFlashNotice(w, r, flashnotice.NoticeSuccess("web.settings.ai_keys.notice_revoked"))
	httpx.WriteRedirect(w, r, routepath.AppSettingsAIKeys)
}

func (h handlers) writeFlashNotice(w http.ResponseWriter, r *http.Request, notice flashnotice.Notice) {
	flashnotice.WriteWithPolicy(w, r, notice, h.flashMeta)
}

func (h handlers) handleAIKeyRevokeRoute(w http.ResponseWriter, r *http.Request) {
	credentialID := strings.TrimSpace(r.PathValue("credentialID"))
	if credentialID == "" {
		h.WriteNotFound(w, r)
		return
	}
	h.handleAIKeyRevoke(w, r, credentialID)
}

func (h handlers) renderProfilePage(w http.ResponseWriter, r *http.Request, statusCode int, profile SettingsProfile, errorMessage string, noticeCode string) {
	loc, _ := h.PageLocalizer(w, r)
	layout := webtemplates.AppMainLayoutOptions{SideMenu: settingsSideMenu(routepath.AppSettingsProfile, loc)}
	h.WritePage(
		w, r,
		webtemplates.T(loc, "web.settings.page_profile_title"),
		statusCode,
		settingsMainHeader(loc),
		layout,
		webtemplates.SettingsProfileFragment(webtemplates.SettingsProfileForm{
			Username:      profile.Username,
			Name:          profile.Name,
			AvatarSetID:   profile.AvatarSetID,
			AvatarAssetID: profile.AvatarAssetID,
			Pronouns:      profile.Pronouns,
			Bio:           profile.Bio,
			NoticeMessage: settingsProfileNoticeMessage(noticeCode, loc),
			ErrorMessage:  errorMessage,
		}, loc),
	)
}

func (h handlers) renderLocalePage(w http.ResponseWriter, r *http.Request, statusCode int, selectedLocale string, errorMessage string) {
	loc, _ := h.PageLocalizer(w, r)
	layout := webtemplates.AppMainLayoutOptions{SideMenu: settingsSideMenu(routepath.AppSettingsLocale, loc)}
	h.WritePage(
		w, r,
		webtemplates.T(loc, "web.settings.page_locale_title"),
		statusCode,
		settingsMainHeader(loc),
		layout,
		webtemplates.SettingsLocaleFragment(webtemplates.SettingsLocaleForm{
			SelectedLocale: selectedLocale,
			ErrorMessage:   errorMessage,
		}, loc),
	)
}

func (h handlers) renderAIKeysPage(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string, statusCode int, label string, errorMessage string) {
	loc, _ := h.PageLocalizer(w, r)
	keys, err := h.service.listAIKeys(ctx, userID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	rows := make([]webtemplates.SettingsAIKeyRow, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, webtemplates.SettingsAIKeyRow{
			ID:        key.ID,
			Label:     key.Label,
			Provider:  key.Provider,
			Status:    key.Status,
			CreatedAt: key.CreatedAt,
			RevokedAt: key.RevokedAt,
			CanRevoke: key.CanRevoke,
		})
	}
	layout := webtemplates.AppMainLayoutOptions{SideMenu: settingsSideMenu(routepath.AppSettingsAIKeys, loc)}
	h.WritePage(
		w, r,
		webtemplates.T(loc, "web.settings.page_ai_keys_title"),
		statusCode,
		settingsMainHeader(loc),
		layout,
		webtemplates.SettingsAIKeysFragment(webtemplates.SettingsAIKeysForm{
			Label:        label,
			Provider:     "OpenAI",
			ErrorMessage: errorMessage,
		}, rows, loc),
	)
}

func settingsProfileNoticeCode(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	return strings.TrimSpace(r.URL.Query().Get(routepath.SettingsNoticeQueryKey))
}

func settingsProfileNoticeMessage(code string, loc webtemplates.Localizer) string {
	switch strings.TrimSpace(code) {
	case routepath.SettingsNoticePublicProfileRequired:
		return webtemplates.T(loc, "web.settings.user_profile.notice_public_profile_required")
	default:
		return ""
	}
}
