package generator

import (
	"context"
	"fmt"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc/metadata"
)

// gmModes is the list of valid GM modes to vary across.
var gmModes = []statev1.GmMode{
	statev1.GmMode_HUMAN,
	statev1.GmMode_AI,
	statev1.GmMode_HYBRID,
}

// pickGmMode selects a GM mode based on configuration.
func (g *Generator) pickGmMode(vary bool, index int) statev1.GmMode {
	if !vary {
		return statev1.GmMode_HUMAN
	}
	return gmModes[index%len(gmModes)]
}

// createCampaign creates a new campaign with the given GM mode.
func (g *Generator) createCampaign(ctx context.Context, gmMode statev1.GmMode) (*statev1.Campaign, string, error) {
	creatorName := g.wb.ParticipantName()
	userResp, err := g.authClient.CreateUser(ctx, &authv1.CreateUserRequest{DisplayName: creatorName})
	if err != nil {
		return nil, "", fmt.Errorf("CreateUser: %w", err)
	}
	userID := userResp.GetUser().GetId()
	if userID == "" {
		return nil, "", fmt.Errorf("CreateUser: missing user id in response")
	}
	callCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(grpcmeta.UserIDHeader, userID))
	resp, err := g.campaigns.CreateCampaign(callCtx, &statev1.CreateCampaignRequest{
		Name:        g.wb.CampaignName(),
		System:      g.gameSystem(),
		GmMode:      gmMode,
		ThemePrompt: g.wb.ThemePrompt(),
	})
	if err != nil {
		return nil, "", fmt.Errorf("CreateCampaign: %w", err)
	}
	if resp == nil {
		return nil, "", fmt.Errorf("CreateCampaign: missing response")
	}
	if resp.OwnerParticipant == nil {
		return nil, "", fmt.Errorf("CreateCampaign: missing owner participant in response")
	}
	ownerParticipantID := resp.OwnerParticipant.GetId()
	if ownerParticipantID == "" {
		return nil, "", fmt.Errorf("CreateCampaign: empty owner participant ID in response")
	}
	return resp.Campaign, ownerParticipantID, nil
}

// transitionCampaignStatus moves a campaign through status transitions.
// Uses the index to determine final status for variety.
func (g *Generator) transitionCampaignStatus(ctx context.Context, campaignID string, index int) error {
	// Status distribution: DRAFT(0), ACTIVE(1), COMPLETED(2), ARCHIVED(3)
	// Pattern: index % 4 determines final status
	targetStatus := index % 4

	switch targetStatus {
	case 0:
		// Keep as DRAFT (initial state) - no action needed
		return nil

	case 1:
		// ACTIVE - campaigns become active when they have a session started
		// This is handled implicitly during session creation
		return nil

	case 2:
		// COMPLETED - end all active sessions first, then end the campaign
		if err := g.endAllActiveSessions(ctx, campaignID); err != nil {
			return err
		}
		_, err := g.campaigns.EndCampaign(ctx, &statev1.EndCampaignRequest{
			CampaignId: campaignID,
		})
		if err != nil {
			return fmt.Errorf("EndCampaign: %w", err)
		}

	case 3:
		// ARCHIVED - end all active sessions, end campaign, then archive
		if err := g.endAllActiveSessions(ctx, campaignID); err != nil {
			return err
		}
		_, err := g.campaigns.EndCampaign(ctx, &statev1.EndCampaignRequest{
			CampaignId: campaignID,
		})
		if err != nil {
			return fmt.Errorf("EndCampaign: %w", err)
		}
		_, err = g.campaigns.ArchiveCampaign(ctx, &statev1.ArchiveCampaignRequest{
			CampaignId: campaignID,
		})
		if err != nil {
			return fmt.Errorf("ArchiveCampaign: %w", err)
		}
	}

	return nil
}

// endAllActiveSessions ends all active sessions for a campaign.
func (g *Generator) endAllActiveSessions(ctx context.Context, campaignID string) error {
	pageToken := ""
	for {
		resp, err := g.sessions.ListSessions(ctx, &statev1.ListSessionsRequest{
			CampaignId: campaignID,
			PageSize:   100,
			PageToken:  pageToken,
		})
		if err != nil {
			return fmt.Errorf("ListSessions: %w", err)
		}

		for _, session := range resp.Sessions {
			if session.Status == statev1.SessionStatus_SESSION_ACTIVE {
				_, err := g.sessions.EndSession(ctx, &statev1.EndSessionRequest{
					CampaignId: campaignID,
					SessionId:  session.Id,
				})
				if err != nil {
					return fmt.Errorf("EndSession %s: %w", session.Id, err)
				}
			}
		}

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	return nil
}
