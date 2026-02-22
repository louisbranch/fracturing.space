package web

import (
	"context"
	"errors"
	"net/http"
	"strings"

	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *handler) renderInviteRecipientLookupError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, errRecipientUsernameRequired):
		h.renderErrorPage(w, r, http.StatusBadRequest, "Invite action unavailable", "recipient username is required")
	case errors.Is(err, errRecipientUsernameFormat):
		h.renderErrorPage(w, r, http.StatusBadRequest, "Invite action unavailable", "recipient username must start with @")
	case errors.Is(err, errConnectionsUnavailable):
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Invite action unavailable", "connections service is not configured")
	case status.Code(err) == codes.InvalidArgument:
		h.renderErrorPage(w, r, http.StatusBadRequest, "Invite action unavailable", "recipient username is invalid")
	case status.Code(err) == codes.NotFound:
		h.renderErrorPage(w, r, http.StatusBadRequest, "Invite action unavailable", "recipient username was not found")
	default:
		h.renderErrorPage(w, r, grpcErrorHTTPStatus(err, http.StatusBadGateway), "Invite action unavailable", "failed to resolve invite recipient")
	}
}

func (h *handler) lookupInviteRecipientVerification(ctx context.Context, recipientUserID string) (webtemplates.CampaignInviteVerification, error) {
	recipientUserID = strings.TrimSpace(recipientUserID)
	if recipientUserID == "" {
		return webtemplates.CampaignInviteVerification{}, errRecipientUsernameRequired
	}
	if !strings.HasPrefix(recipientUserID, "@") {
		return webtemplates.CampaignInviteVerification{}, errRecipientUsernameFormat
	}
	username := strings.TrimSpace(strings.TrimPrefix(recipientUserID, "@"))
	if username == "" {
		return webtemplates.CampaignInviteVerification{}, errRecipientUsernameRequired
	}
	if h == nil || h.connectionsClient == nil {
		return webtemplates.CampaignInviteVerification{}, errConnectionsUnavailable
	}
	resp, err := h.connectionsClient.LookupUserProfile(ctx, &connectionsv1.LookupUserProfileRequest{
		Username: username,
	})
	if err != nil {
		return webtemplates.CampaignInviteVerification{}, err
	}
	profileRecord := resp.GetUserProfile()
	if profileRecord == nil {
		return webtemplates.CampaignInviteVerification{}, status.Error(codes.NotFound, "username not found")
	}
	resolvedUserID := strings.TrimSpace(profileRecord.GetUserId())
	if resolvedUserID == "" {
		return webtemplates.CampaignInviteVerification{}, status.Error(codes.NotFound, "username not found")
	}
	verification := webtemplates.CampaignInviteVerification{
		HasResult: true,
		Username:  strings.TrimSpace(profileRecord.GetUsername()),
		UserID:    resolvedUserID,
	}
	if verification.Username == "" {
		verification.Username = username
	}
	verification.Name = strings.TrimSpace(profileRecord.GetName())
	verification.AvatarSetID = strings.TrimSpace(profileRecord.GetAvatarSetId())
	verification.AvatarAssetID = strings.TrimSpace(profileRecord.GetAvatarAssetId())
	verification.Bio = strings.TrimSpace(profileRecord.GetBio())
	return verification, nil
}

func (h *handler) resolveInviteRecipientUserID(ctx context.Context, recipientUserID string) (string, error) {
	recipientUserID = strings.TrimSpace(recipientUserID)
	if recipientUserID == "" {
		return "", nil
	}
	if !strings.HasPrefix(recipientUserID, "@") {
		return recipientUserID, nil
	}
	username := strings.TrimSpace(strings.TrimPrefix(recipientUserID, "@"))
	if username == "" {
		return "", errRecipientUsernameRequired
	}
	if h == nil || h.connectionsClient == nil {
		return "", errConnectionsUnavailable
	}
	resp, err := h.connectionsClient.LookupUserProfile(ctx, &connectionsv1.LookupUserProfileRequest{
		Username: username,
	})
	if err != nil {
		return "", err
	}
	record := resp.GetUserProfile()
	if record == nil {
		return "", status.Error(codes.NotFound, "username not found")
	}
	resolvedUserID := strings.TrimSpace(record.GetUserId())
	if resolvedUserID == "" {
		return "", status.Error(codes.NotFound, "username not found")
	}
	return resolvedUserID, nil
}
