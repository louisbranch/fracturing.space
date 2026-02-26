package profile

import (
	"context"
	"net/http"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/i18n"
	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/web/support"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

type AppProfileHandlers struct {
	Authenticate         func(*http.Request) bool
	HasAccountClient     func() bool
	ResolveProfileUserID func(context.Context) (string, error)
	LoadAccountLocale    func(context.Context, string) (commonv1.Locale, error)
	UpdateAccountLocale  func(context.Context, string, commonv1.Locale) error
	CacheUserLocale      func(commonv1.Locale)
	RedirectToLogin      func(http.ResponseWriter, *http.Request)
	RenderErrorPage      func(http.ResponseWriter, *http.Request, int, string, string)
	PageContext          func(*http.Request) webtemplates.PageContext
}

// HandleAppProfile handles /app/profile.
func HandleAppProfile(h AppProfileHandlers, w http.ResponseWriter, r *http.Request) {
	if h.Authenticate == nil ||
		h.HasAccountClient == nil ||
		h.ResolveProfileUserID == nil ||
		h.LoadAccountLocale == nil ||
		h.UpdateAccountLocale == nil ||
		h.CacheUserLocale == nil ||
		h.RedirectToLogin == nil ||
		h.RenderErrorPage == nil ||
		h.PageContext == nil {
		http.NotFound(w, r)
		return
	}

	if !h.Authenticate(r) {
		h.RedirectToLogin(w, r)
		return
	}
	if !h.HasAccountClient() {
		h.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Profile unavailable", "account service client is not configured")
		return
	}

	switch r.Method {
	case http.MethodGet:
		handleAppProfileGet(h, w, r)
	case http.MethodPost:
		handleAppProfilePost(h, w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		support.LocalizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
	}
}

func handleAppProfileGet(h AppProfileHandlers, w http.ResponseWriter, r *http.Request) {
	userID, err := h.ResolveProfileUserID(r.Context())
	if err != nil {
		h.RenderErrorPage(w, r, http.StatusBadGateway, "Profile unavailable", "failed to resolve current user")
		return
	}
	if strings.TrimSpace(userID) == "" {
		h.RenderErrorPage(w, r, http.StatusUnauthorized, "Authentication required", "no user identity was resolved for this session")
		return
	}

	locale, err := h.LoadAccountLocale(r.Context(), userID)
	if err != nil {
		h.RenderErrorPage(w, r, http.StatusBadGateway, "Profile unavailable", "failed to load profile")
		return
	}
	locale = i18n.NormalizeLocale(locale)

	RenderAppProfilePage(w, r, h.PageContext(r), locale)
}

func handleAppProfilePost(h AppProfileHandlers, w http.ResponseWriter, r *http.Request) {
	userID, err := h.ResolveProfileUserID(r.Context())
	if err != nil {
		h.RenderErrorPage(w, r, http.StatusBadGateway, "Profile update failed", "failed to resolve current user")
		return
	}
	if strings.TrimSpace(userID) == "" {
		h.RenderErrorPage(w, r, http.StatusUnauthorized, "Authentication required", "no user identity was resolved for this session")
		return
	}
	if err := r.ParseForm(); err != nil {
		h.RenderErrorPage(w, r, http.StatusBadRequest, "Profile update failed", "failed to parse profile form")
		return
	}

	locale := strings.TrimSpace(r.FormValue("locale"))
	parsedLocale, ok := i18n.ParseLocale(locale)
	if !ok {
		h.RenderErrorPage(w, r, http.StatusBadRequest, "Profile update failed", "failed to parse profile locale")
		return
	}

	if err := h.UpdateAccountLocale(r.Context(), userID, parsedLocale); err != nil {
		h.RenderErrorPage(w, r, http.StatusBadGateway, "Profile update failed", "failed to update profile")
		return
	}

	h.CacheUserLocale(parsedLocale)
	http.Redirect(w, r, routepath.AppProfile, http.StatusFound)
}

func RenderAppProfilePage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, locale commonv1.Locale) {
	if err := support.WritePage(
		w,
		r,
		webtemplates.ProfilePage(page, webtemplates.ProfileFormState{
			LocaleOptions: ProfileLocaleOptions(page, locale),
		}),
		support.ComposeHTMXTitleForPage(page, "layout.profile"),
	); err != nil {
		support.LocalizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
	}
}

func ProfileLocaleOptions(page webtemplates.PageContext, selectedLocale commonv1.Locale) []webtemplates.LanguageOption {
	selectedTag := i18n.LocaleString(i18n.NormalizeLocale(selectedLocale))
	displayPage := webtemplates.PageContext{Loc: page.Loc, Lang: selectedTag}
	options := webtemplates.LanguageOptions(displayPage)
	for idx, option := range options {
		options[idx] = webtemplates.LanguageOption{
			Tag:    option.Tag,
			Label:  option.Label,
			Active: strings.TrimSpace(option.Tag) == selectedTag,
		}
	}
	return options
}
