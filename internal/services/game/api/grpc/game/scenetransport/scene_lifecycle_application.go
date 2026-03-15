package scenetransport

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
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

func (a sceneApplication) CreateScene(ctx context.Context, campaignID string, in *campaignv1.CreateSceneRequest) (string, error) {
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return "", err
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return "", err
	}
	if err := authz.RequirePolicy(ctx, a.auth, domainauthz.CapabilityManageSessions, c); err != nil {
		return "", err
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return "", err
	}

	if err := validate.MaxLength(in.GetName(), "name", validate.MaxNameLen); err != nil {
		return "", err
	}
	if err := validate.MaxLength(in.GetDescription(), "description", validate.MaxDescriptionLen); err != nil {
		return "", err
	}

	sceneID, err := a.idGenerator()
	if err != nil {
		return "", grpcerror.Internal("generate scene id", err)
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
		return "", grpcerror.Internal("encode payload", err)
	}

	actorID, actorType := handler.ResolveCommandActor(ctx)

	_, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		a.write,
		a.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         handler.CommandTypeSceneCreate,
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
	if err := authz.RequirePolicy(ctx, a.auth, domainauthz.CapabilityManageSessions, c); err != nil {
		return err
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return err
	}
	if err := validate.MaxLength(in.GetName(), "name", validate.MaxNameLen); err != nil {
		return err
	}
	if err := validate.MaxLength(in.GetDescription(), "description", validate.MaxDescriptionLen); err != nil {
		return err
	}

	payload := scene.UpdatePayload{
		SceneID:     ids.SceneID(sceneID),
		Name:        strings.TrimSpace(in.GetName()),
		Description: strings.TrimSpace(in.GetDescription()),
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
			Type:         handler.CommandTypeSceneUpdate,
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
	if err := authz.RequirePolicy(ctx, a.auth, domainauthz.CapabilityManageSessions, c); err != nil {
		return err
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return err
	}

	payload := scene.EndPayload{
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
			Type:         handler.CommandTypeSceneEnd,
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
	if a.stores.Scene == nil {
		return "", grpcerror.Internal("transition scene source lookup", errors.New("scene store is not configured"))
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return "", err
	}
	if err := authz.RequirePolicy(ctx, a.auth, domainauthz.CapabilityManageSessions, c); err != nil {
		return "", err
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return "", err
	}
	if err := validate.MaxLength(in.GetName(), "name", validate.MaxNameLen); err != nil {
		return "", err
	}
	if err := validate.MaxLength(in.GetDescription(), "description", validate.MaxDescriptionLen); err != nil {
		return "", err
	}
	sourceScene, err := a.stores.Scene.GetScene(ctx, campaignID, sourceSceneID)
	if err != nil {
		return "", err
	}
	sessionID, err := validate.RequiredID(sourceScene.SessionID, "session id")
	if err != nil {
		return "", err
	}

	newSceneID, err := a.idGenerator()
	if err != nil {
		return "", grpcerror.Internal("generate scene id", err)
	}

	payload := scene.TransitionPayload{
		SourceSceneID: ids.SceneID(sourceSceneID),
		Name:          strings.TrimSpace(in.GetName()),
		Description:   strings.TrimSpace(in.GetDescription()),
		NewSceneID:    ids.SceneID(newSceneID),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", grpcerror.Internal("encode payload", err)
	}

	actorID, actorType := handler.ResolveCommandActor(ctx)

	_, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		a.write,
		a.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         handler.CommandTypeSceneTransition,
			ActorType:    actorType,
			ActorID:      actorID,
			SessionID:    sessionID,
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
