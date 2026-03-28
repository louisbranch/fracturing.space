package participanttransport

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"

	"context"
	"encoding/json"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
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
	policyActor, err := authz.RequirePolicyActor(ctx, c.auth, domainauthz.CapabilityManageParticipants(), campaignRecord)
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
	ownerCount, err := authz.CountCampaignOwners(ctx, c.stores.Participant, campaignID)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}
	targetOwnsActiveCharacters, err := authz.ParticipantOwnsActiveCharacters(ctx, c.stores.Character, campaignID, participantID)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}
	targetControlsActiveCharacters, err := authz.ParticipantControlsActiveCharacters(ctx, c.stores.Character, campaignID, participantID)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}
	decision := domainauthz.CanParticipantRemovalEligibility(
		policyActor.CampaignAccess,
		current.CampaignAccess,
		ownerCount,
		current.Controller,
		targetOwnsActiveCharacters,
		targetControlsActiveCharacters,
	)
	if !decision.Allowed {
		authErr := participantPolicyDecisionError(decision.ReasonCode)
		authz.EmitDecisionTelemetry(ctx, authz.DecisionEvent{
			Store:      c.auth.Audit,
			CampaignID: campaignID,
			Capability: domainauthz.CapabilityManageParticipants(),
			Decision:   authz.DecisionDeny,
			ReasonCode: decision.ReasonCode,
			Actor:      policyActor,
			Err:        authErr,
			ExtraAttributes: map[string]any{
				"target_participant_id":             participantID,
				"target_campaign_access":            strings.TrimSpace(string(current.CampaignAccess)),
				"target_owns_active_characters":     targetOwnsActiveCharacters,
				"target_controls_active_characters": targetControlsActiveCharacters,
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

	actorID, actorType := handler.ResolveCommandActor(ctx)
	_, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		c.write,
		c.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandids.ParticipantLeave,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "participant",
			EntityID:     participantID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.Options{
			ApplyErr: handler.ApplyErrorWithCodePreserve("apply event"),
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
		if strings.TrimSpace(campaignRecord.AIAgentID) != "" && c.clearCampaignAIBinding != nil {
			if _, clearErr := c.clearCampaignAIBinding(
				ctx,
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
