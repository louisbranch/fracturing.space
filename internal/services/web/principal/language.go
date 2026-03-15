package principal

import (
	"context"
	"net/http"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/userid"
)

// ResolveLanguage returns the effective request language, preferring the
// account locale for authenticated requests.
func (r Resolver) ResolveLanguage(request *http.Request) string {
	if snapshot := snapshotFromRequest(request); snapshot != nil {
		snapshot.languageOnce.Do(func() {
			snapshot.language = r.resolveLanguageUncached(request)
		})
		return snapshot.language
	}
	return r.resolveLanguageUncached(request)
}

// ResolveRequestLanguage adapts the production resolver to the shared page
// contract used by transport helpers.
func (r Resolver) ResolveRequestLanguage(request *http.Request) string {
	return r.ResolveLanguage(request)
}

// resolveLanguageUncached prefers the authenticated account locale before
// falling back to transport language negotiation.
func (r Resolver) resolveLanguageUncached(request *http.Request) string {
	fallback := webi18n.ResolveTag(request, nil).String()
	if request == nil || r.language.accountProfile.accountClient == nil {
		return fallback
	}
	userID := userid.Normalize(r.ResolveUserID(request))
	if userID == "" {
		return fallback
	}
	profile := r.loadAccountProfile(request.Context(), userID)
	if profile == nil {
		return fallback
	}
	locale := profile.GetLocale()
	if locale == commonv1.Locale_LOCALE_UNSPECIFIED {
		return fallback
	}
	return platformi18n.LocaleString(platformi18n.NormalizeLocale(locale))
}

// loadAccountProfile returns auth-owned profile data and memoizes it inside the
// request snapshot so viewer and language resolution share one lookup.
func (r Resolver) loadAccountProfile(ctx context.Context, userID string) *authv1.AccountProfile {
	if r.language.accountProfile.accountClient == nil {
		return nil
	}
	if snapshot := snapshotFromContext(ctx); snapshot != nil {
		snapshot.accountProfileOnce.Do(func() {
			snapshot.accountProfile = r.language.accountProfile.load(ctx, userID)
		})
		return snapshot.accountProfile
	}
	return r.language.accountProfile.load(ctx, userID)
}

// load fetches auth-owned profile data without request snapshot caching.
func (r accountProfileResolver) load(ctx context.Context, userID string) *authv1.AccountProfile {
	if r.accountClient == nil {
		return nil
	}
	resp, err := r.accountClient.GetProfile(ctx, &authv1.GetProfileRequest{UserId: userID})
	if err != nil || resp == nil {
		return nil
	}
	return resp.GetProfile()
}
