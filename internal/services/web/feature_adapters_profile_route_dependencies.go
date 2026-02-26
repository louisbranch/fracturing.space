package web

import (
	"context"
	"net/http"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/i18n"
	featureprofile "github.com/louisbranch/fracturing.space/internal/services/web/feature/profile"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

func (h *handler) appProfileRouteDependencies(w http.ResponseWriter, r *http.Request) featureprofile.AppProfileHandlers {
	sess := sessionFromRequest(r, h.sessions)
	return featureprofile.AppProfileHandlers{
		Authenticate: func(req *http.Request) bool {
			return sessionFromRequest(req, h.sessions) != nil
		},
		HasAccountClient: func() bool {
			return h.accountClient != nil
		},
		ResolveProfileUserID: func(ctx context.Context) (string, error) {
			return h.resolveProfileUserID(ctx, sess)
		},
		LoadAccountLocale: func(ctx context.Context, userID string) (commonv1.Locale, error) {
			profile, err := h.fetchAccountProfile(ctx, userID)
			if err != nil {
				return commonv1.Locale_LOCALE_UNSPECIFIED, err
			}
			if profile == nil {
				return commonv1.Locale_LOCALE_UNSPECIFIED, nil
			}
			return i18n.NormalizeLocale(profile.Locale), nil
		},
		UpdateAccountLocale: func(ctx context.Context, userID string, locale commonv1.Locale) error {
			_, err := h.accountClient.UpdateProfile(ctx, &authv1.UpdateProfileRequest{
				UserId: userID,
				Locale: locale,
			})
			return err
		},
		CacheUserLocale: func(locale commonv1.Locale) {
			if sess != nil {
				sess.setCachedUserLocale(i18n.LocaleString(i18n.NormalizeLocale(locale)))
			}
		},
		RedirectToLogin: func(writer http.ResponseWriter, req *http.Request) {
			http.Redirect(writer, req, routepath.AuthLogin, http.StatusFound)
		},
		RenderErrorPage: h.renderErrorPage,
		PageContext: func(req *http.Request) webtemplates.PageContext {
			return h.pageContext(w, req)
		},
	}
}
