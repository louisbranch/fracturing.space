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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (c participantApplication) DeleteParticipant(ctx context.Context, campaignID string, in *campaignv1.DeleteParticipantRequest) (storage.ParticipantRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.ParticipantRecord{}, err
	}
	policyActor, err := requirePolicyActor(ctx, c.stores, domainauthz.CapabilityManageParticipants, campaignRecord)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}

	participantID := strings.TrimSpace(in.GetParticipantId())
	if participantID == "" {
		return storage.ParticipantRecord{}, status.Error(codes.InvalidArgument, "participant id is required")
	}

	current, err := c.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}
	ownerCount, err := countCampaignOwners(ctx, c.stores.Participant, campaignID)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}
	targetOwnsActiveCharacters, err := participantOwnsActiveCharacters(ctx, c.stores.Character, campaignID, participantID)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}
	decision := domainauthz.CanParticipantRemovalWithOwnedResources(
		policyActor.CampaignAccess,
		current.CampaignAccess,
		ownerCount,
		targetOwnsActiveCharacters,
	)
	if !decision.Allowed {
		authErr := participantPolicyDecisionError(decision.ReasonCode)
		emitAuthzDecisionTelemetry(
			ctx,
			c.stores.Audit,
			campaignID,
			domainauthz.CapabilityManageParticipants,
			authzDecisionDeny,
			decision.ReasonCode,
			policyActor,
			authErr,
			map[string]any{
				"target_participant_id":         participantID,
				"target_campaign_access":        strings.TrimSpace(string(current.CampaignAccess)),
				"target_owns_active_characters": targetOwnsActiveCharacters,
			},
		)
		return storage.ParticipantRecord{}, authErr
	}

	reason := strings.TrimSpace(in.GetReason())
	applier := c.stores.Applier()
	payload := participant.LeavePayload{
		ParticipantID: participantID,
		Reason:        reason,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.ParticipantRecord{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID, actorType := resolveCommandActor(ctx)
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores,
		applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeParticipantLeave,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "participant",
			EntityID:     participantID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.Options{
			ApplyErr: domainApplyErrorWithCodePreserve("apply event"),
		},
	)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}

	if current.CampaignAccess == participant.CampaignAccessOwner {
		campaignRecord, campaignErr := c.stores.Campaign.Get(ctx, campaignID)
		if campaignErr != nil {
			return storage.ParticipantRecord{}, campaignErr
		}
		if strings.TrimSpace(campaignRecord.AIAgentID) != "" {
			if _, clearErr := clearCampaignAIBindingByCommand(
				ctx,
				c.stores,
				campaignID,
				actorID,
				actorType,
				grpcmeta.RequestIDFromContext(ctx),
				grpcmeta.InvocationIDFromContext(ctx),
			); clearErr != nil {
				return storage.ParticipantRecord{}, clearErr
			}
		}
	}

	return current, nil
}
