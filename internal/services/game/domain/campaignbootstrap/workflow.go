package campaignbootstrap

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

// Decide executes the cross-aggregate campaign bootstrap workflow for
// `campaign.create_with_participants`.
func Decide(current campaign.State, cmd command.Command, now func() time.Time) command.Decision {
	if current.Created {
		return command.Reject(command.Rejection{
			Code:    "CAMPAIGN_ALREADY_EXISTS",
			Message: "campaign already exists",
		})
	}

	var payload campaign.CreateWithParticipantsPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    command.RejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	if len(payload.Participants) == 0 {
		return command.Reject(command.Rejection{
			Code:    "CAMPAIGN_PARTICIPANTS_REQUIRED",
			Message: "at least one participant is required",
		})
	}

	now = command.RequireNowFunc(now)
	decisionTime := now().UTC()
	fixedNow := func() time.Time { return decisionTime }

	campaignDecision := campaign.Decide(current, bootstrapCampaignCreateCommand(cmd, payload.Campaign), fixedNow)
	if len(campaignDecision.Rejections) > 0 {
		return campaignDecision
	}
	if len(campaignDecision.Events) != 1 {
		return command.Reject(command.Rejection{
			Code:    command.RejectionCodeCommandTypeUnsupported,
			Message: "campaign bootstrap did not emit create event",
		})
	}

	participantEvents := make([]event.Event, 0, len(payload.Participants))
	seenParticipantIDs := make(map[string]struct{}, len(payload.Participants))
	for _, joinPayload := range payload.Participants {
		joinDecision := participantBootstrapDecision(cmd, joinPayload, fixedNow, seenParticipantIDs)
		if len(joinDecision.Rejections) > 0 {
			return joinDecision
		}
		participantEvents = append(participantEvents, joinDecision.Events[0])
	}

	events := make([]event.Event, 0, 1+len(participantEvents))
	events = append(events, campaignDecision.Events[0])
	events = append(events, participantEvents...)
	return command.Accept(events...)
}

func bootstrapCampaignCreateCommand(cmd command.Command, payload campaign.CreatePayload) command.Command {
	payloadJSON, _ := json.Marshal(payload)
	return command.Command{
		CampaignID:    cmd.CampaignID,
		Type:          campaign.CommandTypeCreate,
		ActorType:     cmd.ActorType,
		ActorID:       cmd.ActorID,
		RequestID:     cmd.RequestID,
		InvocationID:  cmd.InvocationID,
		CorrelationID: cmd.CorrelationID,
		CausationID:   cmd.CausationID,
		EntityType:    "campaign",
		EntityID:      string(cmd.CampaignID),
		PayloadJSON:   payloadJSON,
	}
}

func participantBootstrapDecision(cmd command.Command, bp campaign.BootstrapParticipant, now func() time.Time, seenParticipantIDs map[string]struct{}) command.Decision {
	participantID := strings.TrimSpace(bp.ParticipantID.String())
	if _, exists := seenParticipantIDs[participantID]; exists {
		return command.Reject(command.Rejection{
			Code:    "CAMPAIGN_PARTICIPANT_DUPLICATE",
			Message: "participant ids must be unique",
		})
	}
	seenParticipantIDs[participantID] = struct{}{}

	joinPayload := participant.JoinPayload{
		ParticipantID:  bp.ParticipantID,
		UserID:         bp.UserID,
		Name:           bp.Name,
		Role:           bp.Role,
		Controller:     bp.Controller,
		CampaignAccess: bp.CampaignAccess,
		AvatarSetID:    bp.AvatarSetID,
		AvatarAssetID:  bp.AvatarAssetID,
		Pronouns:       bp.Pronouns,
	}
	joinPayloadJSON, _ := json.Marshal(joinPayload)
	joinDecision := participant.Decide(
		participant.State{},
		command.Command{
			CampaignID:    cmd.CampaignID,
			Type:          participant.CommandTypeJoin,
			ActorType:     command.ActorTypeSystem,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			EntityType:    "participant",
			EntityID:      participantID,
			PayloadJSON:   joinPayloadJSON,
		},
		now,
	)
	if len(joinDecision.Rejections) > 0 {
		rejection := joinDecision.Rejections[0]
		return command.Reject(command.Rejection{Code: rejection.Code, Message: rejection.Message})
	}
	if len(joinDecision.Events) != 1 {
		return command.Reject(command.Rejection{
			Code:    command.RejectionCodeCommandTypeUnsupported,
			Message: "participant bootstrap did not emit join event",
		})
	}
	return joinDecision
}
