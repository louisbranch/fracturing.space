package game

import (
	"context"
	"encoding/json"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
)

func (a sceneApplication) AddCharacterToScene(ctx context.Context, campaignID string, in *campaignv1.AddCharacterToSceneRequest) error {
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return err
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return err
	}
	if err := requirePolicy(ctx, a.stores, domainauthz.CapabilityManageSessions, c); err != nil {
		return err
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return err
	}

	payload := scene.CharacterAddedPayload{SceneID: ids.SceneID(sceneID), CharacterID: ids.CharacterID(characterID)}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return grpcerror.Internal("encode payload", err)
	}

	actorID, actorType := resolveCommandActor(ctx)

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores.Write,
		a.stores.Applier(),
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeSceneCharacterAdd,
			ActorType:    actorType,
			ActorID:      actorID,
			SceneID:      sceneID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "scene",
			EntityID:     sceneID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.RequireEvents("scene.character.add did not emit an event"),
	)
	return err
}

func (a sceneApplication) RemoveCharacterFromScene(ctx context.Context, campaignID string, in *campaignv1.RemoveCharacterFromSceneRequest) error {
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return err
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return err
	}
	if err := requirePolicy(ctx, a.stores, domainauthz.CapabilityManageSessions, c); err != nil {
		return err
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return err
	}

	payload := scene.CharacterRemovedPayload{SceneID: ids.SceneID(sceneID), CharacterID: ids.CharacterID(characterID)}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return grpcerror.Internal("encode payload", err)
	}

	actorID, actorType := resolveCommandActor(ctx)

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores.Write,
		a.stores.Applier(),
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeSceneCharacterRemove,
			ActorType:    actorType,
			ActorID:      actorID,
			SceneID:      sceneID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "scene",
			EntityID:     sceneID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.RequireEvents("scene.character.remove did not emit an event"),
	)
	return err
}

func (a sceneApplication) TransferCharacter(ctx context.Context, campaignID string, in *campaignv1.TransferCharacterRequest) error {
	sourceSceneID, err := validate.RequiredID(in.GetSourceSceneId(), "source scene id")
	if err != nil {
		return err
	}
	targetSceneID, err := validate.RequiredID(in.GetTargetSceneId(), "target scene id")
	if err != nil {
		return err
	}
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return err
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return err
	}
	if err := requirePolicy(ctx, a.stores, domainauthz.CapabilityManageSessions, c); err != nil {
		return err
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return err
	}

	payload := scene.CharacterTransferPayload{
		SourceSceneID: ids.SceneID(sourceSceneID),
		TargetSceneID: ids.SceneID(targetSceneID),
		CharacterID:   ids.CharacterID(characterID),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return grpcerror.Internal("encode payload", err)
	}

	actorID, actorType := resolveCommandActor(ctx)

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores.Write,
		a.stores.Applier(),
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeSceneCharacterTransfer,
			ActorType:    actorType,
			ActorID:      actorID,
			SceneID:      sourceSceneID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "scene",
			EntityID:     sourceSceneID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.RequireEvents("scene.character.transfer did not emit an event"),
	)
	return err
}
