package web

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/authctx"
)

// campaignAccessChecker answers "can this access token claim campaign access?"
//
// It forms the boundary between authenticated identity and campaign membership.
type campaignAccessChecker interface {
	IsCampaignParticipant(ctx context.Context, campaignID string, accessToken string) (bool, error)
	ResolveUserID(ctx context.Context, accessToken string) (string, error)
}

// campaignAccessService adapts auth introspection + participant reads into a
// campaign membership decision that web routes can enforce before invoking domain
// services.
type campaignAccessService struct {
	authBaseURL         string
	oauthResourceSecret string
	httpClient          *http.Client
	participantClient   statev1.ParticipantServiceClient
}

type introspectResponse = authctx.IntrospectionResult

// newCampaignAccessChecker wires the identity-based membership gate when both auth
// base URL and resource secret are configured.
//
// If either input is missing, membership checks are intentionally disabled to
// allow partial deployments to continue with auth/session gating only.
func newCampaignAccessChecker(config Config, participantClient statev1.ParticipantServiceClient) campaignAccessChecker {
	// If either identity base URL or resource secret is missing, we intentionally
	// disable extra server-side access checks and let upstream auth/session gating
	// control overall trust decisions.
	if participantClient == nil {
		return nil
	}
	authBaseURL := strings.TrimSpace(config.AuthBaseURL)
	resourceSecret := strings.TrimSpace(config.OAuthResourceSecret)
	if authBaseURL == "" || resourceSecret == "" {
		return nil
	}
	return &campaignAccessService{
		authBaseURL:         authBaseURL,
		oauthResourceSecret: resourceSecret,
		httpClient:          http.DefaultClient,
		participantClient:   participantClient,
	}
}

// IsCampaignParticipant resolves token identity and confirms campaign membership
// by walking current participants, which protects campaign routes from being read by
// non-members before state mutation operations.
func (s *campaignAccessService) IsCampaignParticipant(ctx context.Context, campaignID string, accessToken string) (bool, error) {
	campaignID = strings.TrimSpace(campaignID)
	accessToken = strings.TrimSpace(accessToken)
	if campaignID == "" || accessToken == "" {
		return false, nil
	}

	userID, err := s.ResolveUserID(ctx, accessToken)
	if err != nil {
		return false, err
	}
	if userID == "" {
		return false, nil
	}

	pageToken := ""
	for {
		resp, err := s.participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
			CampaignId: campaignID,
			PageSize:   10,
			PageToken:  pageToken,
		})
		if err != nil {
			return false, fmt.Errorf("list campaign participants: %w", err)
		}
		for _, p := range resp.GetParticipants() {
			if strings.TrimSpace(p.GetUserId()) == userID {
				return true, nil
			}
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}
	return false, nil
}

func (s *campaignAccessService) ResolveUserID(ctx context.Context, accessToken string) (string, error) {
	if s == nil || s.httpClient == nil {
		return "", fmt.Errorf("campaign access checker is not configured")
	}
	endpoint := strings.TrimRight(s.authBaseURL, "/") + "/introspect"
	introspectCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resp, err := authctx.NewHTTPIntrospector(endpoint, s.oauthResourceSecret, s.httpClient).Introspect(introspectCtx, accessToken)
	if err != nil {
		return "", fmt.Errorf("call auth introspection: %w", err)
	}
	if !resp.Active {
		return "", nil
	}
	return strings.TrimSpace(resp.UserID), nil
}

func (s *campaignAccessService) introspectParticipantID(ctx context.Context, accessToken string) (string, error) {
	// introspectParticipantID is the auth boundary for web-gate decisions that are
	// participant-specific and should avoid user-level indirection.
	if s == nil || s.httpClient == nil {
		return "", fmt.Errorf("campaign access checker is not configured")
	}
	endpoint := strings.TrimRight(s.authBaseURL, "/") + "/introspect"
	introspectCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resp, err := authctx.NewHTTPIntrospector(endpoint, s.oauthResourceSecret, s.httpClient).Introspect(introspectCtx, accessToken)
	if err != nil {
		return "", fmt.Errorf("call auth introspection: %w", err)
	}
	if !resp.Active {
		return "", nil
	}
	return strings.TrimSpace(resp.ParticipantID), nil
}
