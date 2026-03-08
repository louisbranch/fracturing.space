package game

import (
	"context"
	"encoding/json"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (c characterApplication) DeleteCharacter(ctx context.Context, campaignID string, in *campaignv1.DeleteCharacterRequest) (storage.CharacterRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CharacterRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.CharacterRecord{}, err
	}

	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return storage.CharacterRecord{}, status.Error(codes.InvalidArgument, "character id is required")
	}

	ch, err := c.stores.Character.GetCharacter(ctx, campaignID, characterID)
	if err != nil {
		return storage.CharacterRecord{}, err
	}
	policyActor, err := requireCharacterMutationPolicy(
		ctx,
		c.stores,
		campaignRecord,
		characterID,
	)
	if err != nil {
		return storage.CharacterRecord{}, err
	}

	actorID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
	if actorID == "" {
		actorID = strings.TrimSpace(policyActor.ID)
	}
	reason := strings.TrimSpace(in.GetReason())
	applier := c.stores.Applier()
	payload := character.DeletePayload{
		CharacterID: ids.CharacterID(characterID),
		Reason:      reason,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.CharacterRecord{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores,
		applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeCharacterDelete,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "character",
			EntityID:     characterID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.Options{
			ApplyErrMessage: "apply event",
		},
	)
	if err != nil {
		return storage.CharacterRecord{}, err
	}

	return ch, nil
}
