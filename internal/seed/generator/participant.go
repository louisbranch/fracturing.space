package generator

import (
	"context"
	"fmt"

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

		resp, err := g.participants.CreateParticipant(callCtx, &statev1.CreateParticipantRequest{
			CampaignId:  campaignID,
			DisplayName: g.wb.ParticipantName(),
			Role:        role,
			Controller:  controller,
		})
		if err != nil {
			return nil, fmt.Errorf("CreateParticipant %d: %w", i+1, err)
		}

		participants = append(participants, resp.Participant)
	}

	return participants, nil
}
