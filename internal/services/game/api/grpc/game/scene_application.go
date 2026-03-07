package game

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type sceneApplication struct {
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

func newSceneApplication(service *SceneService) sceneApplication {
	app := sceneApplication{stores: service.stores, clock: service.clock, idGenerator: service.idGenerator}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}

func (a sceneApplication) CreateScene(ctx context.Context, campaignID string, in *campaignv1.CreateSceneRequest) (string, error) {
	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return "", status.Error(codes.InvalidArgument, "session id is required")
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

	charIDs := make([]string, 0, len(in.GetCharacterIds()))
	for _, id := range in.GetCharacterIds() {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			charIDs = append(charIDs, trimmed)
		}
	}

	payload := scene.CreatePayload{
		SceneID:      sceneID,
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
		a.stores,
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
	sceneID := strings.TrimSpace(in.GetSceneId())
	if sceneID == "" {
		return status.Error(codes.InvalidArgument, "scene id is required")
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return err
	}
	if err := requirePolicy(ctx, a.stores, domainauthz.CapabilityManageSessions, c); err != nil {
		return err
	}

	payload := scene.UpdatePayload{
		SceneID:     sceneID,
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
		a.stores,
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
	sceneID := strings.TrimSpace(in.GetSceneId())
	if sceneID == "" {
		return status.Error(codes.InvalidArgument, "scene id is required")
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return err
	}
	if err := requirePolicy(ctx, a.stores, domainauthz.CapabilityManageSessions, c); err != nil {
		return err
	}

	payload := scene.EndPayload{
		SceneID: sceneID,
		Reason:  strings.TrimSpace(in.GetReason()),
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

func (a sceneApplication) TransitionScene(ctx context.Context, campaignID string, in *campaignv1.TransitionSceneRequest) (string, error) {
	sourceSceneID := strings.TrimSpace(in.GetSourceSceneId())
	if sourceSceneID == "" {
		return "", status.Error(codes.InvalidArgument, "source scene id is required")
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
		SourceSceneID: sourceSceneID,
		Name:          strings.TrimSpace(in.GetName()),
		Description:   strings.TrimSpace(in.GetDescription()),
		NewSceneID:    newSceneID,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID, actorType := resolveCommandActor(ctx)

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores,
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

func (a sceneApplication) OpenSceneGate(ctx context.Context, campaignID string, in *campaignv1.OpenSceneGateRequest) error {
	sceneID := strings.TrimSpace(in.GetSceneId())
	if sceneID == "" {
		return status.Error(codes.InvalidArgument, "scene id is required")
	}
	gateType, err := scene.NormalizeGateType(in.GetGateType())
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return err
	}
	if err := requirePolicy(ctx, a.stores, domainauthz.CapabilityManageSessions, c); err != nil {
		return err
	}

	gateID := strings.TrimSpace(in.GetGateId())
	if gateID == "" {
		gateID, err = a.idGenerator()
		if err != nil {
			return status.Errorf(codes.Internal, "generate gate id: %v", err)
		}
	}

	payload := scene.GateOpenedPayload{
		SceneID:  sceneID,
		GateID:   gateID,
		GateType: gateType,
		Reason:   strings.TrimSpace(in.GetReason()),
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
			Type:         commandTypeSceneGateOpen,
			ActorType:    actorType,
			ActorID:      actorID,
			SceneID:      sceneID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "scene_gate",
			EntityID:     gateID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.RequireEvents("scene.gate_open did not emit an event"),
	)
	return err
}

func (a sceneApplication) ResolveSceneGate(ctx context.Context, campaignID string, in *campaignv1.ResolveSceneGateRequest) error {
	sceneID := strings.TrimSpace(in.GetSceneId())
	if sceneID == "" {
		return status.Error(codes.InvalidArgument, "scene id is required")
	}
	gateID := strings.TrimSpace(in.GetGateId())
	if gateID == "" {
		return status.Error(codes.InvalidArgument, "gate id is required")
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return err
	}
	if err := requirePolicy(ctx, a.stores, domainauthz.CapabilityManageSessions, c); err != nil {
		return err
	}

	payload := scene.GateResolvedPayload{
		SceneID:  sceneID,
		GateID:   gateID,
		Decision: strings.TrimSpace(in.GetDecision()),
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
			Type:         commandTypeSceneGateResolve,
			ActorType:    actorType,
			ActorID:      actorID,
			SceneID:      sceneID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "scene_gate",
			EntityID:     gateID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.RequireEvents("scene.gate_resolve did not emit an event"),
	)
	return err
}

func (a sceneApplication) AbandonSceneGate(ctx context.Context, campaignID string, in *campaignv1.AbandonSceneGateRequest) error {
	sceneID := strings.TrimSpace(in.GetSceneId())
	if sceneID == "" {
		return status.Error(codes.InvalidArgument, "scene id is required")
	}
	gateID := strings.TrimSpace(in.GetGateId())
	if gateID == "" {
		return status.Error(codes.InvalidArgument, "gate id is required")
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return err
	}
	if err := requirePolicy(ctx, a.stores, domainauthz.CapabilityManageSessions, c); err != nil {
		return err
	}

	payload := scene.GateAbandonedPayload{
		SceneID: sceneID,
		GateID:  gateID,
		Reason:  strings.TrimSpace(in.GetReason()),
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
			Type:         commandTypeSceneGateAbandon,
			ActorType:    actorType,
			ActorID:      actorID,
			SceneID:      sceneID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "scene_gate",
			EntityID:     gateID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.RequireEvents("scene.gate_abandon did not emit an event"),
	)
	return err
}

func (a sceneApplication) SetSceneSpotlight(ctx context.Context, campaignID string, in *campaignv1.SetSceneSpotlightRequest) error {
	sceneID := strings.TrimSpace(in.GetSceneId())
	if sceneID == "" {
		return status.Error(codes.InvalidArgument, "scene id is required")
	}
	spotlightType, err := scene.NormalizeSpotlightType(in.GetType())
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return err
	}
	if err := requirePolicy(ctx, a.stores, domainauthz.CapabilityManageSessions, c); err != nil {
		return err
	}

	payload := scene.SpotlightSetPayload{
		SceneID:       sceneID,
		SpotlightType: spotlightType,
		CharacterID:   strings.TrimSpace(in.GetCharacterId()),
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
			Type:         commandTypeSceneSpotlightSet,
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
	sceneID := strings.TrimSpace(in.GetSceneId())
	if sceneID == "" {
		return status.Error(codes.InvalidArgument, "scene id is required")
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return err
	}
	if err := requirePolicy(ctx, a.stores, domainauthz.CapabilityManageSessions, c); err != nil {
		return err
	}

	payload := scene.SpotlightClearedPayload{
		SceneID: sceneID,
		Reason:  strings.TrimSpace(in.GetReason()),
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
			Type:         commandTypeSceneSpotlightClear,
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
