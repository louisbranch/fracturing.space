package scenetransport

import (
	"context"
	"encoding/json"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a sceneApplication) SetSceneSpotlight(ctx context.Context, campaignID string, in *campaignv1.SetSceneSpotlightRequest) error {
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return err
	}
	spotlightType, err := scene.NormalizeSpotlightType(in.GetType())
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return err
	}
	if err := authz.RequirePolicy(ctx, a.auth, domainauthz.CapabilityManageSessions(), c); err != nil {
		return err
	}

	payload := scene.SpotlightSetPayload{
		SceneID:       ids.SceneID(sceneID),
		SpotlightType: spotlightType,
		CharacterID:   ids.CharacterID(strings.TrimSpace(in.GetCharacterId())),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return grpcerror.Internal("encode payload", err)
	}

	actorID, actorType := handler.ResolveCommandActor(ctx)

	_, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		a.write,
		a.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         handler.CommandTypeSceneSpotlightSet,
			ActorType:    actorType,
			ActorID:      actorID,
			SceneID:      sceneID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "scene",
			EntityID:     sceneID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.RequireEvents("scene.spotlight_set did not emit an event"),
	)
	return err
}

func (a sceneApplication) ClearSceneSpotlight(ctx context.Context, campaignID string, in *campaignv1.ClearSceneSpotlightRequest) error {
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return err
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return err
	}
	if err := authz.RequirePolicy(ctx, a.auth, domainauthz.CapabilityManageSessions(), c); err != nil {
		return err
	}

	payload := scene.SpotlightClearedPayload{
		SceneID: ids.SceneID(sceneID),
		Reason:  strings.TrimSpace(in.GetReason()),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return grpcerror.Internal("encode payload", err)
	}

	actorID, actorType := handler.ResolveCommandActor(ctx)

	_, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		a.write,
		a.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         handler.CommandTypeSceneSpotlightClear,
			ActorType:    actorType,
			ActorID:      actorID,
			SceneID:      sceneID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "scene",
			EntityID:     sceneID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.RequireEvents("scene.spotlight_clear did not emit an event"),
	)
	return err
}
