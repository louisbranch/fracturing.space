package game

import (
	"context"
	"encoding/json"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func applyDaggerheartCharacterStatePatchCommand(
	ctx context.Context,
	stores Stores,
	campaignID string,
	characterID string,
	actorType event.ActorType,
	actorID string,
	payload daggerheart.CharacterStatePatchPayload,
) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return grpcerror.Internal("encode payload", err)
	}
	_, err = executeAndApplyDomainCommand(
		ctx,
		stores.Write,
		stores.Applier(),
		commandbuild.System(commandbuild.SystemInput{
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
			SystemID:      daggerheart.SystemID,
			SystemVersion: daggerheart.SystemVersion,
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
	payload daggerheart.ConditionChangePayload,
	applyErrMessage string,
) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return grpcerror.Internal("encode condition payload", err)
	}
	_, err = executeAndApplyDomainCommand(
		ctx,
		stores.Write,
		stores.Applier(),
		commandbuild.System(commandbuild.SystemInput{
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
			SystemID:      daggerheart.SystemID,
			SystemVersion: daggerheart.SystemVersion,
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
