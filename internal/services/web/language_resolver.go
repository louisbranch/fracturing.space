package web

import (
	"context"
	"net/http"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"google.golang.org/grpc"
)

// PrincipalAccountClient is the narrow account surface needed by locale resolution.
type PrincipalAccountClient interface {
	GetProfile(context.Context, *authv1.GetProfileRequest, ...grpc.CallOption) (*authv1.GetProfileResponse, error)
}

// languageResolver resolves user locale preferences for request rendering.
type languageResolver struct {
	accountClient PrincipalAccountClient
	resolveUserID func(*http.Request) string
}

func newLanguageResolver(client PrincipalAccountClient, resolveUserID func(*http.Request) string) languageResolver {
	return languageResolver{accountClient: client, resolveUserID: resolveUserID}
}

func (r languageResolver) resolveRequestLanguageUncached(request *http.Request) string {
	fallback := webi18n.ResolveTag(request, nil).String()
	if r.accountClient == nil {
		return fallback
	}
	userID := r.resolveUserID(request)
	if userID == "" {
		return fallback
	}
	resp, err := r.accountClient.GetProfile(request.Context(), &authv1.GetProfileRequest{UserId: userID})
	if err != nil || resp == nil || resp.GetProfile() == nil {
		return fallback
	}
	locale := resp.GetProfile().GetLocale()
	if locale == commonv1.Locale_LOCALE_UNSPECIFIED {
		return fallback
	}
	return platformi18n.LocaleString(platformi18n.NormalizeLocale(locale))
}

func (r languageResolver) resolveRequestLanguage(request *http.Request) string {
	if state := requestPrincipalStateFromRequest(request); state != nil {
		state.languageOnce.Do(func() {
			state.language = r.resolveRequestLanguageUncached(request)
		})
		return state.language
	}
	return r.resolveRequestLanguageUncached(request)
}
