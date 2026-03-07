package game

import (
	"context"
	"encoding/json"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a sceneApplication) AddCharacterToScene(ctx context.Context, campaignID string, in *campaignv1.AddCharacterToSceneRequest) error {
	sceneID := strings.TrimSpace(in.GetSceneId())
	if sceneID == "" {
		return status.Error(codes.InvalidArgument, "scene id is required")
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return status.Error(codes.InvalidArgument, "character id is required")
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return err
	}
	if err := requirePolicy(ctx, a.stores, domainauthz.CapabilityManageSessions, c); err != nil {
		return err
	}

	payload := scene.CharacterAddedPayload{SceneID: sceneID, CharacterID: characterID}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID, actorType := resolveCommandActor(ctx)

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores,
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
	sceneID := strings.TrimSpace(in.GetSceneId())
	if sceneID == "" {
		return status.Error(codes.InvalidArgument, "scene id is required")
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return status.Error(codes.InvalidArgument, "character id is required")
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return err
	}
	if err := requirePolicy(ctx, a.stores, domainauthz.CapabilityManageSessions, c); err != nil {
		return err
	}

	payload := scene.CharacterRemovedPayload{SceneID: sceneID, CharacterID: characterID}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID, actorType := resolveCommandActor(ctx)

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores,
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
	sourceSceneID := strings.TrimSpace(in.GetSourceSceneId())
	if sourceSceneID == "" {
		return status.Error(codes.InvalidArgument, "source scene id is required")
	}
	targetSceneID := strings.TrimSpace(in.GetTargetSceneId())
	if targetSceneID == "" {
		return status.Error(codes.InvalidArgument, "target scene id is required")
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return status.Error(codes.InvalidArgument, "character id is required")
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return err
	}
	if err := requirePolicy(ctx, a.stores, domainauthz.CapabilityManageSessions, c); err != nil {
		return err
	}

	payload := scene.CharacterTransferPayload{
		SourceSceneID: sourceSceneID,
		TargetSceneID: targetSceneID,
		CharacterID:   characterID,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID, actorType := resolveCommandActor(ctx)

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores,
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
