package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

type campaignAccessChecker interface {
	IsCampaignParticipant(ctx context.Context, campaignID string, accessToken string) (bool, error)
}

type campaignAccessService struct {
	authBaseURL         string
	oauthResourceSecret string
	httpClient          *http.Client
	participantClient   statev1.ParticipantServiceClient
}

type introspectResponse struct {
	Active bool   `json:"active"`
	UserID string `json:"user_id"`
}

func newCampaignAccessChecker(config Config, participantClient statev1.ParticipantServiceClient) campaignAccessChecker {
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

func (s *campaignAccessService) IsCampaignParticipant(ctx context.Context, campaignID string, accessToken string) (bool, error) {
	campaignID = strings.TrimSpace(campaignID)
	accessToken = strings.TrimSpace(accessToken)
	if campaignID == "" || accessToken == "" {
		return false, nil
	}

	userID, err := s.introspectUserID(ctx, accessToken)
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

func (s *campaignAccessService) introspectUserID(ctx context.Context, accessToken string) (string, error) {
	if s == nil || s.httpClient == nil {
		return "", fmt.Errorf("campaign access checker is not configured")
	}
	endpoint := strings.TrimRight(s.authBaseURL, "/") + "/introspect"
	introspectCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(introspectCtx, http.MethodPost, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("build introspection request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Resource-Secret", s.oauthResourceSecret)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call auth introspection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth introspection status %d", resp.StatusCode)
	}

	var payload introspectResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode introspection response: %w", err)
	}
	if !payload.Active {
		return "", nil
	}
	return strings.TrimSpace(payload.UserID), nil
}
