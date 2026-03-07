package game

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/fork"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func shouldCopyForkEvent(evt event.Event, copyParticipants bool) (bool, error) {
	switch evt.Type {
	case eventTypeCampaignCreated, eventTypeCampaignForked:
		return false, nil
	case eventTypeCampaignAIBound, eventTypeCampaignAIUnbound:
		return false, nil
	case eventTypeParticipantJoined, eventTypeParticipantUpdated, eventTypeParticipantLeft:
		return copyParticipants, nil
	case eventTypeCharacterUpdated:
		if copyParticipants {
			return true, nil
		}
		var payload character.UpdatePayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return false, fmt.Errorf("decode character.updated payload: %w", err)
		}
		participantValue, hasParticipant := payload.Fields["participant_id"]
		if !hasParticipant {
			return true, nil
		}
		participantID := strings.TrimSpace(participantValue)
		if strings.TrimSpace(participantID) == "" {
			return true, nil
		}
		if len(payload.Fields) == 1 {
			return false, nil
		}
		return true, nil
	default:
		return true, nil
	}
}

func forkEventForCampaign(evt event.Event, campaignID string) event.Event {
	forked := evt
	forked.CampaignID = campaignID
	forked.Seq = 0
	forked.Hash = ""
	forked.PrevHash = ""
	forked.ChainHash = ""
	forked.Signature = ""
	forked.SignatureKeyID = ""
	if strings.EqualFold(evt.EntityType, "campaign") {
		forked.EntityID = campaignID
	}
	return forked
}

// calculateDepth calculates the fork depth by walking up the parent chain.
func calculateDepth(ctx context.Context, store storage.CampaignForkStore, campaignID string) int {
	depth := 0
	currentID := campaignID

	for i := 0; i < 100; i++ { // Limit to prevent infinite loops
		metadata, err := store.GetCampaignForkMetadata(ctx, currentID)
		if err != nil || metadata.ParentCampaignID == "" {
			break
		}
		depth++
		currentID = metadata.ParentCampaignID
	}

	return depth
}

// forkPointFromProto converts a proto ForkPoint to domain ForkPoint.
func forkPointFromProto(pb *campaignv1.ForkPoint) fork.ForkPoint {
	if pb == nil {
		return fork.ForkPoint{}
	}
	return fork.ForkPoint{
		EventSeq:  pb.GetEventSeq(),
		SessionID: pb.GetSessionId(),
	}
}

// isNotFound reports whether err is a not-found error.
func isNotFound(err error) bool {
	return err == storage.ErrNotFound
}
