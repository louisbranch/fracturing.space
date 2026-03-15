package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"

	"context"
	"encoding/json"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (c campaignApplication) SetCampaignAIBinding(ctx context.Context, campaignID, aiAgentID string) (storage.CampaignRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.CampaignRecord{}, err
	}

	ownerActor, err := requireCampaignOwner(ctx, c.auth, campaignRecord)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := c.validateAIBindingAgent(ctx, campaignID, aiAgentID, ownerActor.UserID); err != nil {
		return storage.CampaignRecord{}, err
	}

	payloadJSON, err := json.Marshal(campaign.AIBindPayload{AIAgentID: strings.TrimSpace(aiAgentID)})
	if err != nil {
		return storage.CampaignRecord{}, grpcerror.Internal("encode payload", err)
	}
	actorID, actorType := handler.ResolveCommandActor(ctx)
	_, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		c.write,
		c.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         handler.CommandTypeCampaignAIBind,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "campaign",
			EntityID:     campaignID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.Options{},
	)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := rotateCampaignAIAuthEpoch(ctx, c.commandExecution(), campaignID, aiAuthRotateReasonCampaignAIBound, actorID, actorType); err != nil {
		return storage.CampaignRecord{}, err
	}

	updated, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, grpcerror.Internal("load campaign", err)
	}
	return updated, nil
}

func (c campaignApplication) ClearCampaignAIBinding(ctx context.Context, campaignID string) (storage.CampaignRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.CampaignRecord{}, err
	}
	if _, err := requireCampaignOwner(ctx, c.auth, campaignRecord); err != nil {
		return storage.CampaignRecord{}, err
	}
	if strings.TrimSpace(campaignRecord.AIAgentID) == "" {
		return campaignRecord, nil
	}

	actorID, actorType := handler.ResolveCommandActor(ctx)
	return clearCampaignAIBindingByCommand(
		ctx,
		c.commandExecution(),
		campaignID,
		actorID,
		actorType,
		grpcmeta.RequestIDFromContext(ctx),
		grpcmeta.InvocationIDFromContext(ctx),
	)
}

func (c campaignApplication) validateAIBindingAgent(ctx context.Context, campaignID, aiAgentID, ownerUserID string) error {
	if c.aiClient == nil {
		return status.Error(codes.Internal, "ai agent client is not configured")
	}

	callCtx := grpcauthctx.WithUserID(ctx, ownerUserID)
	_, err := c.aiClient.ValidateCampaignAgentBinding(callCtx, &aiv1.ValidateCampaignAgentBindingRequest{
		AgentId:    strings.TrimSpace(aiAgentID),
		CampaignId: campaignID,
	})
	if err != nil {
		return err
	}
	return nil
}

func requireCampaignOwner(ctx context.Context, deps authz.PolicyDeps, campaignRecord storage.CampaignRecord) (storage.ParticipantRecord, error) {
	actor, err := authz.RequirePolicyActor(ctx, deps, domainauthz.CapabilityManageCampaign, campaignRecord)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}
	if actor.CampaignAccess != participant.CampaignAccessOwner {
		return storage.ParticipantRecord{}, status.Error(codes.PermissionDenied, "owner permission is required")
	}
	if strings.TrimSpace(actor.UserID) == "" {
		return storage.ParticipantRecord{}, status.Error(codes.PermissionDenied, "owner user identity is required")
	}
	return actor, nil
}

func clearCampaignAIBindingByCommand(
	ctx context.Context,
	deps campaignCommandExecution,
	campaignID string,
	actorID string,
	actorType command.ActorType,
	requestID string,
	invocationID string,
) (storage.CampaignRecord, error) {
	payloadJSON, err := json.Marshal(campaign.AIUnbindPayload{})
	if err != nil {
		return storage.CampaignRecord{}, grpcerror.Internal("encode payload", err)
	}
	_, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		deps.Write,
		deps.Applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         handler.CommandTypeCampaignAIUnbind,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    requestID,
			InvocationID: invocationID,
			EntityType:   "campaign",
			EntityID:     campaignID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.Options{},
	)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := rotateCampaignAIAuthEpoch(ctx, deps, campaignID, aiAuthRotateReasonCampaignAIUnbound, actorID, actorType); err != nil {
		return storage.CampaignRecord{}, err
	}

	updated, err := deps.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, grpcerror.Internal("load campaign", err)
	}
	return updated, nil
}
