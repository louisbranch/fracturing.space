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
