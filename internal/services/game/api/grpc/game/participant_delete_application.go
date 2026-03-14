package game

import (
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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (c participantApplication) DeleteParticipant(ctx context.Context, campaignID string, in *campaignv1.DeleteParticipantRequest) (storage.ParticipantRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.ParticipantRecord{}, err
	}
	policyActor, err := requirePolicyActorWithDependencies(ctx, c.auth, domainauthz.CapabilityManageParticipants, campaignRecord)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}

	participantID, err := validate.RequiredID(in.GetParticipantId(), "participant id")
	if err != nil {
		return storage.ParticipantRecord{}, err
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
		emitAuthzDecisionTelemetry(ctx, authzDecisionEvent{
			Store:      c.auth.Audit,
			CampaignID: campaignID,
			Capability: domainauthz.CapabilityManageParticipants,
			Decision:   authzDecisionDeny,
			ReasonCode: decision.ReasonCode,
			Actor:      policyActor,
			Err:        authErr,
			ExtraAttributes: map[string]any{
				"target_participant_id":         participantID,
				"target_campaign_access":        strings.TrimSpace(string(current.CampaignAccess)),
				"target_owns_active_characters": targetOwnsActiveCharacters,
			},
		})
		return storage.ParticipantRecord{}, authErr
	}

	reason := strings.TrimSpace(in.GetReason())
	payload := participant.LeavePayload{
		ParticipantID: ids.ParticipantID(participantID),
		Reason:        reason,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.ParticipantRecord{}, grpcerror.Internal("encode payload", err)
	}

	actorID, actorType := resolveCommandActor(ctx)
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.write,
		c.applier,
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
				campaignCommandExecution{
					Campaign: c.stores.Campaign,
					Write:    c.write,
					Applier:  c.applier,
				},
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
