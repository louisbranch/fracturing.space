package web

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/platform/branding"
	sharedhtmx "github.com/louisbranch/fracturing.space/internal/services/shared/htmx"
	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

const gamePageContentType = "text/html; charset=utf-8"

var errNoWebPageComponent = errors.New("web: no page component provided")

func (h *handler) resolvedAppName() string {
	if h == nil {
		return branding.AppName
	}
	appName := strings.TrimSpace(h.config.AppName)
	if appName == "" {
		return branding.AppName
	}
	return appName
}

func (h *handler) pageContext(w http.ResponseWriter, r *http.Request) webtemplates.PageContext {
	printer, lang := localizer(w, r)
	page := webtemplates.PageContext{
		Lang:          lang,
		Loc:           printer,
		CurrentPath:   r.URL.Path,
		CurrentQuery:  r.URL.RawQuery,
		UserName:      "",
		UserAvatarURL: "",
		AppName:       h.resolvedAppName(),
	}

	sess := sessionFromRequest(r, h.sessions)
	if sess != nil {
		page.UserName = strings.TrimSpace(sess.displayName)
		if page.UserName == "" {
			page.UserName = webtemplates.T(page.Loc, "web.dashboard.user_name_fallback")
		}
		page.UserAvatarURL = h.pageContextUserAvatar(r.Context(), sess)
	}

	return page
}

func (h *handler) pageContextUserAvatar(ctx context.Context, sess *session) string {
	if h == nil {
		return ""
	}
	if sess == nil || h.campaignAccess == nil {
		return ""
	}

	if avatar, ok := sess.cachedUserAvatar(); ok {
		return avatar
	}

	userID, err := h.sessionUserIDForSession(ctx, sess)
	if err != nil {
		return ""
	}
	if strings.TrimSpace(userID) == "" {
		sess.setCachedUserAvatar("")
		return ""
	}

	avatarURL := avatarImageURL(h.config, catalog.AvatarRoleUser, userID, "", "")
	sess.setCachedUserAvatar(avatarURL)
	return avatarURL
}

func (h *handler) pageContextForCampaign(w http.ResponseWriter, r *http.Request, campaignID string) webtemplates.PageContext {
	page := h.pageContext(w, r)
	page.CampaignName = h.campaignDisplayName(r.Context(), campaignID)
	return page
}

func (h *handler) writePage(w http.ResponseWriter, r *http.Request, page templ.Component, htmxTitle string) error {
	return writePage(w, r, page, htmxTitle)
}

func writePage(w http.ResponseWriter, r *http.Request, page templ.Component, htmxTitle string) error {
	writeGameContentType(w)
	if page == nil {
		return errNoWebPageComponent
	}
	if sharedhtmx.IsHTMXRequest(r) {
		sharedhtmx.RenderPage(w, r, page, page, htmxTitle)
		return nil
	}
	return page.Render(r.Context(), w)
}

func composeHTMXTitle(loc webtemplates.Localizer, title string, args ...any) string {
	if loc == nil {
		return sharedhtmx.TitleTag(sharedtemplates.ComposePageTitle(title))
	}
	return sharedhtmx.TitleTag(sharedtemplates.ComposePageTitle(webtemplates.T(loc, title, args...)))
}

func composeHTMXTitleForPage(page webtemplates.PageContext, title string, args ...any) string {
	return composeHTMXTitle(page.Loc, title, args...)
}

func writeGameContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", gamePageContentType)
}
