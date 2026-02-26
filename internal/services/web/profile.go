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
	"github.com/louisbranch/fracturing.space/internal/services/shared/authctx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type accountProfileView struct {
	Locale commonv1.Locale
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
		profile.Locale = resp.GetProfile().GetLocale()
	}
	return profile, nil
}
