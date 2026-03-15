package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
)

func newCampaignAuthorizer(
	config Config,
	participantClient statev1.ParticipantServiceClient,
	sessionClient statev1.SessionServiceClient,
	campaignClient statev1.CampaignServiceClient,
	authSessionClient webSessionAuthClient,
) wsAuthorizer {
	authBaseURL := strings.TrimSpace(config.AuthBaseURL)
	resourceSecret := strings.TrimSpace(config.OAuthResourceSecret)
	if (authBaseURL == "" || resourceSecret == "") && authSessionClient == nil && participantClient == nil && sessionClient == nil {
		return nil
	}
	var httpClient *http.Client
	if authBaseURL != "" && resourceSecret != "" {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}

	return &campaignAuthorizer{
		authBaseURL:         authBaseURL,
		oauthResourceSecret: resourceSecret,
		httpClient:          httpClient,
		authSessionClient:   authSessionClient,
		participantClient:   participantClient,
		sessionClient:       sessionClient,
		campaignClient:      campaignClient,
	}
}

func (a *campaignAuthorizer) Authenticate(ctx context.Context, accessToken string) (string, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return "", errors.New("access token is required")
	}
	if strings.HasPrefix(accessToken, webSessionTokenPrefix) {
		return a.authenticateWebSession(ctx, strings.TrimPrefix(accessToken, webSessionTokenPrefix))
	}
	if a == nil || a.httpClient == nil {
		return "", errors.New("auth is not configured")
	}

	endpoint := strings.TrimRight(a.authBaseURL, "/") + "/introspect"
	authCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(authCtx, http.MethodPost, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("build introspection request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Resource-Secret", a.oauthResourceSecret)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call auth introspection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth introspection status %d", resp.StatusCode)
	}

	var payload authIntrospectResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode introspection response: %w", err)
	}
	if !payload.Active {
		return "", errors.New("inactive access token")
	}

	userID := strings.TrimSpace(payload.UserID)
	if userID == "" {
		return "", errors.New("introspection returned empty user id")
	}
	return userID, nil
}

func (a *campaignAuthorizer) authenticateWebSession(ctx context.Context, sessionID string) (string, error) {
	if a == nil || a.authSessionClient == nil {
		return "", errors.New("web session auth is not configured")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", errors.New("web session id is required")
	}
	resp, err := a.authSessionClient.GetWebSession(ctx, &authv1.GetWebSessionRequest{SessionId: sessionID})
	if err != nil {
		return "", fmt.Errorf("lookup web session: %w", err)
	}
	if resp == nil || resp.GetSession() == nil {
		return "", errors.New("web session not found")
	}
	userID := strings.TrimSpace(resp.GetSession().GetUserId())
	if userID == "" {
		return "", errors.New("web session returned empty user id")
	}
	return userID, nil
}

func (a *campaignAuthorizer) ResolveJoinWelcome(ctx context.Context, campaignID string, sessionID string, userID string) (joinWelcome, error) {
	campaignID = strings.TrimSpace(campaignID)
	sessionID = strings.TrimSpace(sessionID)
	userID = strings.TrimSpace(userID)
	if campaignID == "" {
		return joinWelcome{}, errors.New("campaign id is required")
	}
	if sessionID == "" {
		return joinWelcome{}, errors.New("session id is required")
	}

	participant, err := a.findParticipantByUserID(ctx, campaignID, userID)
	if err != nil {
		return joinWelcome{}, err
	}
	if participant == nil {
		return joinWelcome{}, errCampaignParticipantRequired
	}

	session, err := a.getSession(ctx, campaignID, sessionID, userID)
	if err != nil {
		return joinWelcome{}, err
	}

	campaignName := campaignID
	if a.campaignClient != nil {
		callCtx, cancel := context.WithTimeout(grpcauthctx.WithUserID(ctx, userID), 3*time.Second)
		resp, err := a.campaignClient.GetCampaign(callCtx, &statev1.GetCampaignRequest{CampaignId: campaignID})
		cancel()
		if err != nil {
			return joinWelcome{}, fmt.Errorf("get campaign: %w", err)
		}
		if campaign := resp.GetCampaign(); campaign != nil {
			if name := strings.TrimSpace(campaign.GetName()); name != "" {
				campaignName = name
			}
		}
	}

	participantName := strings.TrimSpace(participant.GetName())
	if participantName == "" {
		participantName = userID
	}
	resolvedSessionID := strings.TrimSpace(session.GetId())
	if resolvedSessionID == "" {
		resolvedSessionID = sessionID
	}
	sessionName := strings.TrimSpace(session.GetName())
	if sessionName == "" {
		sessionName = resolvedSessionID
	}

	return joinWelcome{
		ParticipantID:   strings.TrimSpace(participant.GetId()),
		ParticipantName: participantName,
		CampaignName:    campaignName,
		SessionID:       resolvedSessionID,
		SessionName:     sessionName,
	}, nil
}

func (a *campaignAuthorizer) getSession(ctx context.Context, campaignID string, sessionID string, userID string) (*statev1.Session, error) {
	if a == nil || a.sessionClient == nil {
		return nil, errors.New("session client is not configured")
	}
	callCtx, cancel := context.WithTimeout(grpcauthctx.WithUserID(ctx, userID), 3*time.Second)
	defer cancel()

	resp, err := a.sessionClient.GetSession(callCtx, &statev1.GetSessionRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	})
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if resp == nil || resp.GetSession() == nil {
		return nil, errors.New("session not found")
	}
	return resp.GetSession(), nil
}

func (a *campaignAuthorizer) findParticipantByUserID(ctx context.Context, campaignID string, userID string) (*statev1.Participant, error) {
	if a == nil || a.participantClient == nil {
		return nil, errors.New("participant client is not configured")
	}
	pageToken := ""
	for {
		callCtx, cancel := context.WithTimeout(grpcauthctx.WithUserID(ctx, userID), 3*time.Second)
		resp, err := a.participantClient.ListParticipants(callCtx, &statev1.ListParticipantsRequest{
			CampaignId: campaignID,
			PageSize:   100,
			PageToken:  pageToken,
		})
		cancel()
		if err != nil {
			return nil, fmt.Errorf("list campaign participants: %w", err)
		}
		for _, participant := range resp.GetParticipants() {
			if strings.TrimSpace(participant.GetUserId()) == userID {
				return participant, nil
			}
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}
	return nil, nil
}
