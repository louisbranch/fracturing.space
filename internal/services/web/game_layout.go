package web

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/platform/branding"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	sharedhtmx "github.com/louisbranch/fracturing.space/internal/services/shared/htmx"
	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/i18n"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

const gamePageContentType = "text/html; charset=utf-8"
const unreadNotificationProbeTTL = 20 * time.Second
const unreadNotificationProbeTimeout = 350 * time.Millisecond

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
		Lang:             lang,
		Loc:              printer,
		CurrentPath:      r.URL.Path,
		CurrentQuery:     r.URL.RawQuery,
		ChatFallbackPort: chatFallbackPort(h.config.ChatHTTPAddr),
		UserName:         "",
		UserAvatarURL:    "",
		AppName:          h.resolvedAppName(),
	}

	sess := sessionFromRequest(r, h.sessions)
	if sess != nil {
		if accountLang := h.pageContextLanguage(r.Context(), sess, lang); strings.TrimSpace(accountLang) != "" {
			page.Lang = accountLang
			if tag, ok := platformi18n.ParseTag(accountLang); ok {
				page.Loc = webi18n.Printer(tag)
				if shouldSetLanguageCookie(r, tag.String()) {
					webi18n.SetLanguageCookie(w, tag)
				}
			}
		}
		page.UserName = strings.TrimSpace(sess.displayName)
		if page.UserName == "" {
			page.UserName = webtemplates.T(page.Loc, "web.dashboard.user_name_fallback")
		}
		page.UserAvatarURL = h.pageContextUserAvatar(r.Context(), sess)
		page.HasUnreadNotifications = h.pageContextHasUnreadNotifications(r.Context(), sess)
	}

	return page
}

func (h *handler) pageContextLanguage(ctx context.Context, sess *session, fallback string) string {
	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		fallback = platformi18n.DefaultTag().String()
	}
	if h == nil || sess == nil || h.accountClient == nil {
		return fallback
	}

	if cached, ok := sess.cachedLocaleTag(); ok {
		cached = strings.TrimSpace(cached)
		if cached == "" {
			return fallback
		}
		return cached
	}

	userID, err := h.resolveProfileUserID(ctx, sess)
	if err != nil || strings.TrimSpace(userID) == "" {
		return fallback
	}

	profile, err := h.fetchAccountProfile(ctx, userID)
	if err != nil || profile == nil {
		return fallback
	}
	if profile.Locale == commonv1.Locale_LOCALE_UNSPECIFIED {
		return fallback
	}

	accountLang := platformi18n.LocaleString(platformi18n.NormalizeLocale(profile.Locale))
	sess.setCachedUserLocale(accountLang)
	return accountLang
}

func shouldSetLanguageCookie(r *http.Request, expected string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return false
	}
	if r == nil {
		return true
	}
	cookie, err := r.Cookie(webi18n.LangCookieName)
	if err != nil {
		return true
	}
	return strings.TrimSpace(cookie.Value) != expected
}

func chatFallbackPort(rawAddr string) string {
	trimmed := strings.TrimSpace(rawAddr)
	if trimmed == "" {
		return ""
	}
	_, port, err := net.SplitHostPort(trimmed)
	if err == nil {
		return sanitizePort(port)
	}

	if strings.Count(trimmed, ":") <= 1 {
		if idx := strings.LastIndex(trimmed, ":"); idx >= 0 {
			return sanitizePort(trimmed[idx+1:])
		}
	}

	return sanitizePort(trimmed)
}

func sanitizePort(raw string) string {
	port := strings.TrimSpace(raw)
	if port == "" {
		return ""
	}
	n, err := strconv.Atoi(port)
	if err != nil {
		return ""
	}
	if n < 1 || n > 65535 {
		return ""
	}
	return port
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

func (h *handler) pageContextHasUnreadNotifications(ctx context.Context, sess *session) bool {
	if h == nil || sess == nil || h.notificationClient == nil {
		return false
	}

	if hasUnread, ok := sess.cachedUnreadNotifications(unreadNotificationProbeTTL); ok {
		return hasUnread
	}

	userID, err := h.sessionUserIDForSession(ctx, sess)
	if err != nil {
		return staleUnreadStateOrDefault(sess)
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return staleUnreadStateOrDefault(sess)
	}

	probeCtx, cancel := context.WithTimeout(ctx, unreadNotificationProbeTimeout)
	defer cancel()

	resp, err := h.notificationClient.GetUnreadNotificationStatus(grpcauthctx.WithUserID(probeCtx, userID), &notificationsv1.GetUnreadNotificationStatusRequest{})
	if err != nil {
		return staleUnreadStateOrDefault(sess)
	}

	hasUnread := resp.GetHasUnread() || resp.GetUnreadCount() > 0
	sess.setCachedUnreadNotifications(hasUnread)
	return hasUnread
}

func staleUnreadStateOrDefault(sess *session) bool {
	if sess == nil {
		return false
	}
	hasUnread, ok := sess.cachedUnreadNotifications(0)
	if !ok {
		return false
	}
	return hasUnread
}

func hasUnreadNotifications(notifications []*notificationsv1.Notification) bool {
	for _, notification := range notifications {
		if notification == nil {
			continue
		}
		if notification.GetReadAt() == nil {
			return true
		}
	}
	return false
}

func (h *handler) pageContextForCampaign(w http.ResponseWriter, r *http.Request, campaignID string) webtemplates.PageContext {
	page := h.pageContext(w, r)
	page.CampaignName = h.campaignDisplayName(r.Context(), campaignID)
	page.CampaignCoverImageURL = h.campaignCoverImage(r.Context(), campaignID)
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
