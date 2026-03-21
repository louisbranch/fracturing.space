package snapshottransport

import (
	"context"
	"encoding/json"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
)

func applyDaggerheartCharacterStatePatchCommand(
	ctx context.Context,
	write domainwriteexec.WritePath,
	applier projection.Applier,
	campaignID string,
	characterID string,
	actorType event.ActorType,
	actorID string,
	payload daggerheartpayload.CharacterStatePatchPayload,
) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return grpcerror.Internal("encode payload", err)
	}
	_, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		write,
		applier,
		commandbuild.System(commandbuild.SystemInput{
			CoreInput: commandbuild.CoreInput{
				CampaignID:   campaignID,
				Type:         handler.CommandTypeDaggerheartCharacterStatePatch,
				ActorType:    handler.CommandActorTypeForEventActor(actorType),
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
			ApplyErr:        handler.ApplyErrorWithCodePreserve("apply event"),
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func executeDaggerheartConditionChangeCommand(
	ctx context.Context,
	write domainwriteexec.WritePath,
	applier projection.Applier,
	campaignID string,
	characterID string,
	actorType event.ActorType,
	actorID string,
	sessionID string,
	payload daggerheartpayload.ConditionChangePayload,
	applyErrMessage string,
) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return grpcerror.Internal("encode condition payload", err)
	}
	_, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		write,
		applier,
		commandbuild.System(commandbuild.SystemInput{
			CoreInput: commandbuild.CoreInput{
				CampaignID:   campaignID,
				Type:         handler.CommandTypeDaggerheartConditionChange,
				ActorType:    handler.CommandActorTypeForEventActor(actorType),
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
			ApplyErr:        handler.ApplyErrorWithCodePreserve(applyErrMessage),
		},
	)
	if err != nil {
		return err
	}
	return nil
}
