package auth

import (
	"context"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type joinGrantIssuer struct {
	store storage.UserStore
	clock func() time.Time
}

func newJoinGrantIssuer(service *AuthService) joinGrantIssuer {
	issuer := joinGrantIssuer{store: service.store, clock: service.clock}
	if issuer.clock == nil {
		issuer.clock = time.Now
	}
	return issuer
}

func (j joinGrantIssuer) issue(ctx context.Context, in *authv1.IssueJoinGrantRequest) (joinGrantResult, error) {
	if j.store == nil {
		return joinGrantResult{}, status.Error(codes.Internal, "User store is not configured.")
	}

	userID := strings.TrimSpace(in.GetUserId())
	if userID == "" {
		return joinGrantResult{}, status.Error(codes.InvalidArgument, "User ID is required.")
	}
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return joinGrantResult{}, status.Error(codes.InvalidArgument, "Campaign ID is required.")
	}
	inviteID := strings.TrimSpace(in.GetInviteId())
	if inviteID == "" {
		return joinGrantResult{}, status.Error(codes.InvalidArgument, "Invite ID is required.")
	}
	participantID := strings.TrimSpace(in.GetParticipantId())

	if _, err := j.store.GetUser(ctx, userID); err != nil {
		return joinGrantResult{}, err
	}

	config, err := loadJoinGrantConfigFromEnv()
	if err != nil {
		return joinGrantResult{}, status.Errorf(codes.Internal, "Join grant config: %v", err)
	}

	issuedAt := j.clock().UTC()
	expiresAt := issuedAt.Add(config.ttl)
	jti, err := id.NewID()
	if err != nil {
		return joinGrantResult{}, status.Errorf(codes.Internal, "Generate join grant ID: %v", err)
	}

	payload := map[string]any{
		"iss":         config.issuer,
		"aud":         config.audience,
		"sub":         userID,
		"exp":         expiresAt.Unix(),
		"iat":         issuedAt.Unix(),
		"jti":         jti,
		"campaign_id": campaignID,
		"invite_id":   inviteID,
		"user_id":     userID,
	}
	if participantID != "" {
		payload["participant_id"] = participantID
	}

	grant, err := encodeJoinGrant(config, payload)
	if err != nil {
		return joinGrantResult{}, status.Errorf(codes.Internal, "Sign join grant: %v", err)
	}

	return joinGrantResult{
		grant:     grant,
		jti:       jti,
		expiresAt: expiresAt,
	}, nil
}

type joinGrantResult struct {
	grant     string
	jti       string
	expiresAt time.Time
}
