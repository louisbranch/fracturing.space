package game

import (
	"context"
	"encoding/json"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func applyDaggerheartCharacterStatePatchCommand(
	ctx context.Context,
	stores Stores,
	campaignID string,
	characterID string,
	actorType event.ActorType,
	actorID string,
	payload daggerheart.CharacterStatePatchedPayload,
) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return status.Errorf(codes.Internal, "encode payload: %v", err)
	}
	_, err = executeAndApplyDomainCommand(
		ctx,
		stores,
		stores.Applier(),
		commandbuild.DaggerheartSystem(commandbuild.DaggerheartSystemInput{
			CoreInput: commandbuild.CoreInput{
				CampaignID:   campaignID,
				Type:         commandTypeDaggerheartCharacterStatePatch,
				ActorType:    commandActorTypeForEventActor(actorType),
				ActorID:      actorID,
				SessionID:    grpcmeta.SessionIDFromContext(ctx),
				RequestID:    grpcmeta.RequestIDFromContext(ctx),
				InvocationID: grpcmeta.InvocationIDFromContext(ctx),
				EntityType:   "character",
				EntityID:     characterID,
				PayloadJSON:  payloadJSON,
			},
		}),
		domainwrite.Options{
			RequireEvents:   true,
			MissingEventMsg: "character state patch did not emit an event",
			ApplyErr:        domainApplyErrorWithCodePreserve("apply event"),
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func executeDaggerheartConditionChangeCommand(
	ctx context.Context,
	stores Stores,
	campaignID string,
	characterID string,
	actorType event.ActorType,
	actorID string,
	sessionID string,
	payload daggerheart.ConditionChangedPayload,
	applyErrMessage string,
) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return status.Errorf(codes.Internal, "encode condition payload: %v", err)
	}
	_, err = executeAndApplyDomainCommand(
		ctx,
		stores,
		stores.Applier(),
		commandbuild.DaggerheartSystem(commandbuild.DaggerheartSystemInput{
			CoreInput: commandbuild.CoreInput{
				CampaignID:   campaignID,
				Type:         commandTypeDaggerheartConditionChange,
				ActorType:    commandActorTypeForEventActor(actorType),
				ActorID:      actorID,
				SessionID:    sessionID,
				RequestID:    grpcmeta.RequestIDFromContext(ctx),
				InvocationID: grpcmeta.InvocationIDFromContext(ctx),
				EntityType:   "character",
				EntityID:     characterID,
				PayloadJSON:  payloadJSON,
			},
		}),
		domainwrite.Options{
			RequireEvents:   true,
			MissingEventMsg: "condition change did not emit an event",
			ApplyErr:        domainApplyErrorWithCodePreserve(applyErrMessage),
		},
	)
	if err != nil {
		return err
	}
	return nil
}
