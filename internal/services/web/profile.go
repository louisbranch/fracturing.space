package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/shared/authctx"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type accountProfileView struct {
	Name   string
	Locale commonv1.Locale
}

func (h *handler) handleAppProfile(w http.ResponseWriter, r *http.Request) {
	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}
	if h.accountClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Profile unavailable", "account service client is not configured")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleAppProfileGet(w, r, sess)
	case http.MethodPost:
		h.handleAppProfilePost(w, r, sess)
	default:
		w.Header().Set("Allow", "GET, POST")
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
	}
}

func (h *handler) handleAppProfileGet(w http.ResponseWriter, r *http.Request, sess *session) {
	requestCtx := r.Context()
	userID, err := h.resolveProfileUserID(requestCtx, sess)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Profile unavailable", "failed to resolve current user")
		return
	}
	if strings.TrimSpace(userID) == "" {
		h.renderErrorPage(w, r, http.StatusUnauthorized, "Authentication required", "no user identity was resolved for this session")
		return
	}

	profileResp, err := h.fetchAccountProfile(requestCtx, userID)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Profile unavailable", "failed to load profile")
		return
	}
	page := h.pageContext(w, r)
	if strings.TrimSpace(profileResp.Name) == "" {
		profileResp.Name = page.UserName
	}
	locale := profileResp.Locale
	locale = platformi18n.NormalizeLocale(locale)

	renderAppProfilePage(w, r, page, profileResp.Name, locale)
}

func (h *handler) handleAppProfilePost(w http.ResponseWriter, r *http.Request, sess *session) {
	requestCtx := r.Context()
	userID, err := h.resolveProfileUserID(requestCtx, sess)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Profile update failed", "failed to resolve current user")
		return
	}
	if strings.TrimSpace(userID) == "" {
		h.renderErrorPage(w, r, http.StatusUnauthorized, "Authentication required", "no user identity was resolved for this session")
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Profile update failed", "failed to parse profile form")
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	locale := strings.TrimSpace(r.FormValue("locale"))
	parsedLocale, ok := platformi18n.ParseLocale(locale)
	if !ok {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Profile update failed", "failed to parse profile locale")
		return
	}

	_, err = h.accountClient.UpdateProfile(requestCtx, &authv1.UpdateProfileRequest{
		UserId: userID,
		Name:   name,
		Locale: parsedLocale,
	})
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Profile update failed", "failed to update profile")
		return
	}
	http.Redirect(w, r, "/profile", http.StatusFound)
}

func (h *handler) resolveProfileUserID(ctx context.Context, sess *session) (string, error) {
	if sess == nil {
		return "", errors.New("session is not available")
	}
	userID, err := h.sessionUserIDForSession(ctx, sess)
	if err == nil {
		return userID, nil
	}

	resolvedID, resolveErr := h.resolveProfileUserIDFromToken(ctx, sess.accessToken)
	if resolveErr != nil {
		return "", resolveErr
	}
	sess.setCachedUserID(resolvedID)
	return resolvedID, nil
}

func (h *handler) resolveProfileUserIDFromToken(ctx context.Context, accessToken string) (string, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return "", nil
	}

	authBaseURL := strings.TrimSpace(h.config.AuthBaseURL)
	resourceSecret := strings.TrimSpace(h.config.OAuthResourceSecret)
	if authBaseURL == "" || resourceSecret == "" {
		return "", nil
	}

	introspectCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	introspectEndpoint := strings.TrimRight(authBaseURL, "/") + "/introspect"
	resp, err := authctx.NewHTTPIntrospector(introspectEndpoint, resourceSecret, http.DefaultClient).Introspect(introspectCtx, accessToken)
	if err != nil {
		return "", fmt.Errorf("call auth introspection: %w", err)
	}
	if !resp.Active {
		return "", nil
	}
	return strings.TrimSpace(resp.UserID), nil
}

func renderAppProfilePage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, name string, locale commonv1.Locale) {
	if err := writePage(
		w,
		r,
		webtemplates.ProfilePage(page, webtemplates.ProfileFormState{
			Name:          strings.TrimSpace(name),
			LocaleOptions: profileLocaleOptions(page, locale),
		}),
		composeHTMXTitleForPage(page, "layout.profile"),
	); err != nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
	}
}

func (h *handler) fetchAccountProfile(ctx context.Context, userID string) (*accountProfileView, error) {
	resp, err := h.accountClient.GetProfile(ctx, &authv1.GetProfileRequest{UserId: userID})
	if err != nil {
		if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.NotFound {
			return &accountProfileView{}, nil
		}
		return nil, err
	}
	profile := &accountProfileView{}
	if resp != nil && resp.GetProfile() != nil {
		profile.Name = resp.GetProfile().GetName()
		profile.Locale = platformi18n.NormalizeLocale(resp.GetProfile().GetLocale())
	}
	return profile, nil
}

func profileLocaleOptions(page webtemplates.PageContext, selectedLocale commonv1.Locale) []webtemplates.LanguageOption {
	selectedTag := platformi18n.LocaleString(platformi18n.NormalizeLocale(selectedLocale))
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
