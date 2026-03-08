package game

import (
	"context"
	"encoding/json"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a sceneApplication) CreateScene(ctx context.Context, campaignID string, in *campaignv1.CreateSceneRequest) (string, error) {
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return "", err
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return "", err
	}
	if err := requirePolicy(ctx, a.stores, domainauthz.CapabilityManageSessions, c); err != nil {
		return "", err
	}

	sceneID, err := a.idGenerator()
	if err != nil {
		return "", status.Errorf(codes.Internal, "generate scene id: %v", err)
	}

	charIDs := make([]ids.CharacterID, 0, len(in.GetCharacterIds()))
	for _, id := range in.GetCharacterIds() {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			charIDs = append(charIDs, ids.CharacterID(trimmed))
		}
	}

	payload := scene.CreatePayload{
		SceneID:      ids.SceneID(sceneID),
		Name:         strings.TrimSpace(in.GetName()),
		Description:  strings.TrimSpace(in.GetDescription()),
		CharacterIDs: charIDs,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID, actorType := resolveCommandActor(ctx)

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores.Write,
		a.stores.Applier(),
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeSceneCreate,
			ActorType:    actorType,
			ActorID:      actorID,
			SessionID:    sessionID,
			SceneID:      sceneID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "scene",
			EntityID:     sceneID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.RequireEvents("scene.create did not emit an event"),
	)
	if err != nil {
		return "", err
	}
	return sceneID, nil
}

func (a sceneApplication) UpdateScene(ctx context.Context, campaignID string, in *campaignv1.UpdateSceneRequest) error {
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
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

	payload := scene.UpdatePayload{
		SceneID:     ids.SceneID(sceneID),
		Name:        strings.TrimSpace(in.GetName()),
		Description: strings.TrimSpace(in.GetDescription()),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID, actorType := resolveCommandActor(ctx)

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores.Write,
		a.stores.Applier(),
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeSceneUpdate,
			ActorType:    actorType,
			ActorID:      actorID,
			SceneID:      sceneID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "scene",
			EntityID:     sceneID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.RequireEvents("scene.update did not emit an event"),
	)
	return err
}

func (a sceneApplication) EndScene(ctx context.Context, campaignID string, in *campaignv1.EndSceneRequest) error {
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
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

	payload := scene.EndPayload{
		SceneID: ids.SceneID(sceneID),
		Reason:  strings.TrimSpace(in.GetReason()),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID, actorType := resolveCommandActor(ctx)

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores.Write,
		a.stores.Applier(),
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeSceneEnd,
			ActorType:    actorType,
			ActorID:      actorID,
			SceneID:      sceneID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "scene",
			EntityID:     sceneID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.RequireEvents("scene.end did not emit an event"),
	)
	return err
}

func (a sceneApplication) TransitionScene(ctx context.Context, campaignID string, in *campaignv1.TransitionSceneRequest) (string, error) {
	sourceSceneID, err := validate.RequiredID(in.GetSourceSceneId(), "source scene id")
	if err != nil {
		return "", err
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return "", err
	}
	if err := requirePolicy(ctx, a.stores, domainauthz.CapabilityManageSessions, c); err != nil {
		return "", err
	}

	newSceneID, err := a.idGenerator()
	if err != nil {
		return "", status.Errorf(codes.Internal, "generate scene id: %v", err)
	}

	payload := scene.TransitionPayload{
		SourceSceneID: ids.SceneID(sourceSceneID),
		Name:          strings.TrimSpace(in.GetName()),
		Description:   strings.TrimSpace(in.GetDescription()),
		NewSceneID:    ids.SceneID(newSceneID),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID, actorType := resolveCommandActor(ctx)

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores.Write,
		a.stores.Applier(),
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeSceneTransition,
			ActorType:    actorType,
			ActorID:      actorID,
			SceneID:      sourceSceneID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "scene",
			EntityID:     sourceSceneID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.RequireEvents("scene.transition did not emit an event"),
	)
	if err != nil {
		return "", err
	}
	return newSceneID, nil
}
