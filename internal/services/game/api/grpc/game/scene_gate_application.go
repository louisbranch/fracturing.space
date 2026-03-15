package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"

	"context"
	"encoding/json"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a sceneApplication) OpenSceneGate(ctx context.Context, campaignID string, in *campaignv1.OpenSceneGateRequest) error {
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return err
	}
	gateType, err := scene.NormalizeGateType(in.GetGateType())
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return err
	}
	if err := authz.RequirePolicy(ctx, a.auth, domainauthz.CapabilityManageSessions, c); err != nil {
		return err
	}

	gateID := strings.TrimSpace(in.GetGateId())
	if gateID == "" {
		gateID, err = a.idGenerator()
		if err != nil {
			return grpcerror.Internal("generate gate id", err)
		}
	}

	payload := scene.GateOpenedPayload{
		SceneID:  ids.SceneID(sceneID),
		GateID:   ids.GateID(gateID),
		GateType: gateType,
		Reason:   strings.TrimSpace(in.GetReason()),
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
			Type:         handler.CommandTypeSceneGateOpen,
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
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return err
	}
	gateID, err := validate.RequiredID(in.GetGateId(), "gate id")
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

	payload := scene.GateResolvedPayload{
		SceneID:  ids.SceneID(sceneID),
		GateID:   ids.GateID(gateID),
		Decision: strings.TrimSpace(in.GetDecision()),
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
			Type:         handler.CommandTypeSceneGateResolve,
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
	sceneID, err := validate.RequiredID(in.GetSceneId(), "scene id")
	if err != nil {
		return err
	}
	gateID, err := validate.RequiredID(in.GetGateId(), "gate id")
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

	payload := scene.GateAbandonedPayload{
		SceneID: ids.SceneID(sceneID),
		GateID:  ids.GateID(gateID),
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
			Type:         handler.CommandTypeSceneGateAbandon,
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
