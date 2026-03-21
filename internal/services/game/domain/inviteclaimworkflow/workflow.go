package inviteclaimworkflow

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

// Decide executes the cross-aggregate invite claim workflow for
// `invite.claim_bind`.
func Decide(current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	var payload ClaimBindPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    command.RejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}

	now = command.NowFunc(now)
	decisionTime := now().UTC()
	fixedNow := func() time.Time { return decisionTime }

	bindDecision := participant.Decide(
		current.Participants[payload.ParticipantID],
		participantBindCommand(cmd, payload),
		fixedNow,
	)
	if len(bindDecision.Rejections) > 0 {
		return bindDecision
	}
	if len(bindDecision.Events) != 1 {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeBindEventMissing,
			Message: "invite claim workflow did not emit participant bind event",
		})
	}

	claimDecision := invite.Decide(
		current.Invites[payload.InviteID],
		inviteClaimCommand(cmd, payload),
		fixedNow,
	)
	if len(claimDecision.Rejections) > 0 {
		return claimDecision
	}
	if len(claimDecision.Events) != 1 {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeClaimEventMissing,
			Message: "invite claim workflow did not emit invite claim event",
		})
	}

	events := make([]event.Event, 0, 2)
	events = append(events, bindDecision.Events[0])
	events = append(events, claimDecision.Events[0])
	return command.Accept(events...)
}

func participantBindCommand(cmd command.Command, payload ClaimBindPayload) command.Command {
	payloadJSON, _ := json.Marshal(participant.BindPayload{
		ParticipantID: payload.ParticipantID,
		UserID:        payload.UserID,
	})
	return command.Command{
		CampaignID:    cmd.CampaignID,
		Type:          participant.CommandTypeBind,
		ActorType:     cmd.ActorType,
		ActorID:       cmd.ActorID,
		RequestID:     cmd.RequestID,
		InvocationID:  cmd.InvocationID,
		CorrelationID: cmd.CorrelationID,
		CausationID:   cmd.CausationID,
		EntityType:    "participant",
		EntityID:      string(payload.ParticipantID),
		PayloadJSON:   payloadJSON,
	}
}

func inviteClaimCommand(cmd command.Command, payload ClaimBindPayload) command.Command {
	payloadJSON, _ := json.Marshal(invite.ClaimPayload{
		InviteID:      payload.InviteID,
		ParticipantID: payload.ParticipantID,
		UserID:        payload.UserID,
		JWTID:         payload.JWTID,
	})
	return command.Command{
		CampaignID:    cmd.CampaignID,
		Type:          invite.CommandTypeClaim,
		ActorType:     cmd.ActorType,
		ActorID:       cmd.ActorID,
		RequestID:     cmd.RequestID,
		InvocationID:  cmd.InvocationID,
		CorrelationID: cmd.CorrelationID,
		CausationID:   cmd.CausationID,
		EntityType:    "invite",
		EntityID:      string(payload.InviteID),
		PayloadJSON:   payloadJSON,
	}
}
