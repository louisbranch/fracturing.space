package settings

import (
	"context"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	module "github.com/louisbranch/fracturing.space/internal/services/web2/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web2/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web2/platform/httpx"
	web2i18n "github.com/louisbranch/fracturing.space/internal/services/web2/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web2/platform/pagerender"
	"github.com/louisbranch/fracturing.space/internal/services/web2/platform/webctx"
	"github.com/louisbranch/fracturing.space/internal/services/web2/platform/weberror"
	"github.com/louisbranch/fracturing.space/internal/services/web2/routepath"
	web2templates "github.com/louisbranch/fracturing.space/internal/services/web2/templates"
)

type handlers struct {
	service service
	deps    runtimeDependencies
}

type runtimeDependencies struct {
	resolveUserID   module.ResolveUserID
	resolveLanguage module.ResolveLanguage
	resolveViewer   module.ResolveViewer
}

func newRuntimeDependencies(deps module.Dependencies) runtimeDependencies {
	return runtimeDependencies{
		resolveUserID:   deps.ResolveUserID,
		resolveLanguage: deps.ResolveLanguage,
		resolveViewer:   deps.ResolveViewer,
	}
}

func (d runtimeDependencies) moduleDependencies() module.Dependencies {
	return module.Dependencies{
		ResolveViewer:   d.resolveViewer,
		ResolveLanguage: d.resolveLanguage,
	}
}

func settingsMainHeader(loc web2templates.Localizer) *web2templates.AppMainHeader {
	return &web2templates.AppMainHeader{Title: web2templates.T(loc, "layout.settings")}
}

func newHandlers(s service, deps module.Dependencies) handlers {
	return handlers{service: s, deps: newRuntimeDependencies(deps)}
}

func (h handlers) redirectSettingsRoot(w http.ResponseWriter, r *http.Request) {
	if httpx.IsHTMXRequest(r) {
		httpx.WriteHXRedirect(w, routepath.AppSettingsProfile)
		return
	}
	http.Redirect(w, r, routepath.AppSettingsProfile, http.StatusFound)
}

func (h handlers) handleProfileGet(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.requestContextAndUserID(r)
	profile, err := h.service.loadProfile(ctx, userID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.renderProfilePage(w, r, http.StatusOK, profile, "")
}

func (h handlers) handleProfilePost(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.requestContextAndUserID(r)
	if err := r.ParseForm(); err != nil {
		h.writeError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_parse_profile_form", "failed to parse profile form"))
		return
	}
	existingProfile, err := h.service.loadProfile(ctx, userID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	profile := SettingsProfile{
		Username:      strings.TrimSpace(r.FormValue("username")),
		Name:          strings.TrimSpace(r.FormValue("name")),
		AvatarSetID:   existingProfile.AvatarSetID,
		AvatarAssetID: existingProfile.AvatarAssetID,
		Bio:           strings.TrimSpace(r.FormValue("bio")),
	}
	if err := h.service.saveProfile(ctx, userID, profile); err != nil {
		if apperrors.HTTPStatus(err) == http.StatusBadRequest {
			loc, _ := h.pageLocalizer(w, r)
			h.renderProfilePage(w, r, http.StatusBadRequest, profile, web2i18n.LocalizeError(loc, err))
			return
		}
		h.writeError(w, r, err)
		return
	}
	if httpx.IsHTMXRequest(r) {
		httpx.WriteHXRedirect(w, routepath.AppSettingsProfile)
		return
	}
	http.Redirect(w, r, routepath.AppSettingsProfile, http.StatusFound)
}

func (h handlers) handleLocaleGet(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.requestContextAndUserID(r)
	locale, err := h.service.loadLocale(ctx, userID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.renderLocalePage(w, r, http.StatusOK, platformi18n.LocaleString(locale), "")
}

func (h handlers) handleLocalePost(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.requestContextAndUserID(r)
	if err := r.ParseForm(); err != nil {
		h.writeError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_parse_locale_form", "failed to parse locale form"))
		return
	}
	selectedLocale := strings.TrimSpace(r.FormValue("locale"))
	if err := h.service.saveLocale(ctx, userID, selectedLocale); err != nil {
		if apperrors.HTTPStatus(err) == http.StatusBadRequest {
			loc, _ := h.pageLocalizer(w, r)
			h.renderLocalePage(w, r, http.StatusBadRequest, selectedLocale, web2i18n.LocalizeError(loc, err))
			return
		}
		h.writeError(w, r, err)
		return
	}
	if httpx.IsHTMXRequest(r) {
		httpx.WriteHXRedirect(w, routepath.AppSettingsLocale)
		return
	}
	http.Redirect(w, r, routepath.AppSettingsLocale, http.StatusFound)
}

func (h handlers) handleAIKeysGet(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.requestContextAndUserID(r)
	h.renderAIKeysPage(w, r, ctx, userID, http.StatusOK, "", "")
}

func (h handlers) handleAIKeysCreate(w http.ResponseWriter, r *http.Request) {
	ctx, userID := h.requestContextAndUserID(r)
	if err := r.ParseForm(); err != nil {
		h.writeError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_parse_ai_key_form", "failed to parse ai key form"))
		return
	}
	label := strings.TrimSpace(r.FormValue("label"))
	secret := strings.TrimSpace(r.FormValue("secret"))
	if err := h.service.createAIKey(ctx, userID, label, secret); err != nil {
		if apperrors.HTTPStatus(err) == http.StatusBadRequest {
			loc, _ := h.pageLocalizer(w, r)
			h.renderAIKeysPage(w, r, ctx, userID, http.StatusBadRequest, label, web2i18n.LocalizeError(loc, err))
			return
		}
		h.writeError(w, r, err)
		return
	}
	if httpx.IsHTMXRequest(r) {
		httpx.WriteHXRedirect(w, routepath.AppSettingsAIKeys)
		return
	}
	http.Redirect(w, r, routepath.AppSettingsAIKeys, http.StatusFound)
}

func (h handlers) handleAIKeyRevoke(w http.ResponseWriter, r *http.Request, credentialID string) {
	ctx, userID := h.requestContextAndUserID(r)
	if err := h.service.revokeAIKey(ctx, userID, credentialID); err != nil {
		h.writeError(w, r, err)
		return
	}
	if httpx.IsHTMXRequest(r) {
		httpx.WriteHXRedirect(w, routepath.AppSettingsAIKeys)
		return
	}
	http.Redirect(w, r, routepath.AppSettingsAIKeys, http.StatusFound)
}

func (h handlers) handleAIKeyRevokeRoute(w http.ResponseWriter, r *http.Request) {
	credentialID := strings.TrimSpace(r.PathValue("credentialID"))
	if credentialID == "" {
		h.handleNotFound(w, r)
		return
	}
	h.handleAIKeyRevoke(w, r, credentialID)
}

func (h handlers) handleNotFound(w http.ResponseWriter, r *http.Request) {
	weberror.WriteAppError(w, r, http.StatusNotFound, h.deps.moduleDependencies())
}

func (h handlers) renderProfilePage(w http.ResponseWriter, r *http.Request, statusCode int, profile SettingsProfile, errorMessage string) {
	loc, _ := h.pageLocalizer(w, r)
	layout := web2templates.AppMainLayoutOptions{SideMenu: settingsSideMenu(routepath.AppSettingsProfile, loc)}
	h.writePage(
		w,
		r,
		web2templates.T(loc, "web.settings.page_profile_title"),
		statusCode,
		settingsMainHeader(loc),
		layout,
		web2templates.SettingsProfileFragment(web2templates.SettingsProfileForm{
			Username:      profile.Username,
			Name:          profile.Name,
			AvatarSetID:   profile.AvatarSetID,
			AvatarAssetID: profile.AvatarAssetID,
			Bio:           profile.Bio,
			ErrorMessage:  errorMessage,
		}, loc),
	)
}

func (h handlers) renderLocalePage(w http.ResponseWriter, r *http.Request, statusCode int, selectedLocale string, errorMessage string) {
	loc, _ := h.pageLocalizer(w, r)
	layout := web2templates.AppMainLayoutOptions{SideMenu: settingsSideMenu(routepath.AppSettingsLocale, loc)}
	h.writePage(
		w,
		r,
		web2templates.T(loc, "web.settings.page_locale_title"),
		statusCode,
		settingsMainHeader(loc),
		layout,
		web2templates.SettingsLocaleFragment(web2templates.SettingsLocaleForm{
			SelectedLocale: selectedLocale,
			ErrorMessage:   errorMessage,
		}, loc),
	)
}

func (h handlers) renderAIKeysPage(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string, statusCode int, label string, errorMessage string) {
	loc, _ := h.pageLocalizer(w, r)
	keys, err := h.service.listAIKeys(ctx, userID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	rows := make([]web2templates.SettingsAIKeyRow, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, web2templates.SettingsAIKeyRow{
			ID:        key.ID,
			Label:     key.Label,
			Provider:  key.Provider,
			Status:    key.Status,
			CreatedAt: key.CreatedAt,
			RevokedAt: key.RevokedAt,
			CanRevoke: key.CanRevoke,
		})
	}
	layout := web2templates.AppMainLayoutOptions{SideMenu: settingsSideMenu(routepath.AppSettingsAIKeys, loc)}
	h.writePage(
		w,
		r,
		web2templates.T(loc, "web.settings.page_ai_keys_title"),
		statusCode,
		settingsMainHeader(loc),
		layout,
		web2templates.SettingsAIKeysFragment(web2templates.SettingsAIKeysForm{
			Label:        label,
			Provider:     "OpenAI",
			ErrorMessage: errorMessage,
		}, rows, loc),
	)
}

func (h handlers) writePage(
	w http.ResponseWriter,
	r *http.Request,
	title string,
	statusCode int,
	header *web2templates.AppMainHeader,
	layout web2templates.AppMainLayoutOptions,
	fragment templ.Component,
) {
	if err := pagerender.WriteModulePage(w, r, h.deps.moduleDependencies(), pagerender.ModulePage{
		Title:      title,
		StatusCode: statusCode,
		Header:     header,
		Layout:     layout,
		Fragment:   fragment,
	}); err != nil {
		h.writeError(w, r, err)
	}
}

func (h handlers) pageLocalizer(w http.ResponseWriter, r *http.Request) (web2templates.Localizer, string) {
	loc, lang := web2i18n.ResolveLocalizer(w, r, h.deps.resolveLanguage)
	return loc, lang
}

func (h handlers) writeError(w http.ResponseWriter, r *http.Request, err error) {
	weberror.WriteModuleError(w, r, err, h.deps.moduleDependencies())
}

func (h handlers) requestUserID(r *http.Request) string {
	if r == nil || h.deps.resolveUserID == nil {
		return ""
	}
	return strings.TrimSpace(h.deps.resolveUserID(r))
}

func (h handlers) requestContextAndUserID(r *http.Request) (context.Context, string) {
	ctx := webctx.WithResolvedUserID(r, h.deps.resolveUserID)
	return ctx, h.requestUserID(r)
}

func settingsSideMenu(currentPath string, loc web2templates.Localizer) *web2templates.AppSideMenu {
	return &web2templates.AppSideMenu{
		CurrentPath: currentPath,
		Items: []web2templates.AppSideMenuItem{
			{
				Label:      web2templates.T(loc, "layout.settings_user_profile"),
				URL:        routepath.AppSettingsProfile,
				MatchExact: true,
				IconID:     commonv1.IconId_ICON_ID_PROFILE,
			},
			{
				Label:      web2templates.T(loc, "layout.locale"),
				URL:        routepath.AppSettingsLocale,
				MatchExact: true,
				IconID:     commonv1.IconId_ICON_ID_SETTINGS,
			},
			{
				Label:      web2templates.T(loc, "layout.settings_ai_keys"),
				URL:        routepath.AppSettingsAIKeys,
				MatchExact: true,
				IconID:     commonv1.IconId_ICON_ID_AI,
			},
		},
	}
}
