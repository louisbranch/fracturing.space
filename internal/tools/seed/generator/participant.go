package generator

import (
	"context"
	"fmt"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc/metadata"
)

// createParticipants creates the specified number of participants for a campaign.
// The first participant is always a GM, the rest are players.
func (g *Generator) createParticipants(ctx context.Context, campaignID, ownerParticipantID string, count int) ([]*statev1.Participant, error) {
	if count < 1 {
		count = 1 // At minimum, we need a GM
	}

	participants := make([]*statev1.Participant, 0, count)
	callCtx := ctx
	if ownerParticipantID != "" {
		callCtx = metadata.AppendToOutgoingContext(ctx, grpcmeta.ParticipantIDHeader, ownerParticipantID)
	}

	for i := 0; i < count; i++ {
		role := statev1.ParticipantRole_PLAYER
		if i == 0 {
			role = statev1.ParticipantRole_GM
		}

		// Vary controller type: mostly human, occasionally AI
		controller := statev1.Controller_CONTROLLER_HUMAN
		if g.rng.Float32() < 0.2 { // 20% chance of AI controller
			controller = statev1.Controller_CONTROLLER_AI
		}

		displayName := g.uniqueDisplayName(g.wb.ParticipantName())
		resp, err := g.participants.CreateParticipant(callCtx, &statev1.CreateParticipantRequest{
			CampaignId:  campaignID,
			DisplayName: displayName,
			Role:        role,
			Controller:  controller,
		})
		if err != nil {
			return nil, fmt.Errorf("CreateParticipant %d: %w", i+1, err)
		}

		created := resp.Participant
		participants = append(participants, created)

		if controller == statev1.Controller_CONTROLLER_HUMAN {
			userResp, err := g.authClient.CreateUser(ctx, &authv1.CreateUserRequest{
				Username: displayName,
			})
			if err != nil {
				return nil, fmt.Errorf("CreateUser for participant %d: %w", i+1, err)
			}
			userID := userResp.GetUser().GetId()
			if userID == "" {
				return nil, fmt.Errorf("CreateUser for participant %d: missing user id", i+1)
			}

			inviteRecipient := ""
			claimInvite := false
			switch g.rng.Intn(4) {
			case 0:
				inviteRecipient = ""
				claimInvite = false
			case 1:
				inviteRecipient = ""
				claimInvite = true
			case 2:
				inviteRecipient = userID
				claimInvite = false
			case 3:
				inviteRecipient = userID
				claimInvite = true
			}

			inviteResp, err := g.invites.CreateInvite(callCtx, &statev1.CreateInviteRequest{
				CampaignId:      campaignID,
				ParticipantId:   created.GetId(),
				RecipientUserId: inviteRecipient,
			})
			if err != nil {
				return nil, fmt.Errorf("CreateInvite for participant %d: %w", i+1, err)
			}
			inviteID := inviteResp.GetInvite().GetId()
			if inviteID == "" {
				return nil, fmt.Errorf("CreateInvite for participant %d: missing invite id", i+1)
			}

			if claimInvite {
				grantResp, err := g.authClient.IssueJoinGrant(ctx, &authv1.IssueJoinGrantRequest{
					UserId:        userID,
					CampaignId:    campaignID,
					InviteId:      inviteID,
					ParticipantId: created.GetId(),
				})
				if err != nil {
					return nil, fmt.Errorf("IssueJoinGrant for participant %d: %w", i+1, err)
				}
				joinGrant := grantResp.GetJoinGrant()
				if joinGrant == "" {
					return nil, fmt.Errorf("IssueJoinGrant for participant %d: missing join grant", i+1)
				}

				claimCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(grpcmeta.UserIDHeader, userID))
				_, err = g.invites.ClaimInvite(claimCtx, &statev1.ClaimInviteRequest{
					CampaignId: campaignID,
					InviteId:   inviteID,
					JoinGrant:  joinGrant,
				})
				if err != nil {
					return nil, fmt.Errorf("ClaimInvite for participant %d: %w", i+1, err)
				}
			}
		}
	}

	return participants, nil
}
