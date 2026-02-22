package daggerheart

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

func decideRestTake(snapshotState SnapshotState, cmd command.Command, now func() time.Time) command.Decision {
	var payload RestTakePayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	if now == nil {
		now = time.Now
	}
	payload.RestType = strings.TrimSpace(payload.RestType)
	if payload.LongTermCountdown != nil {
		if rejection := countdownUpdateSnapshotRejection(snapshotState, *payload.LongTermCountdown); rejection != nil {
			return command.Reject(*rejection)
		}
		payload.LongTermCountdown.CountdownID = strings.TrimSpace(payload.LongTermCountdown.CountdownID)
		payload.LongTermCountdown.Reason = strings.TrimSpace(payload.LongTermCountdown.Reason)
	}
	payloadJSON, _ := json.Marshal(payload)
	entityID := strings.TrimSpace(cmd.EntityID)
	if entityID == "" {
		entityID = cmd.CampaignID
	}
	restEvent := command.NewEvent(cmd, EventTypeRestTaken, "session", entityID, payloadJSON, now().UTC())

	if payload.LongTermCountdown == nil {
		return command.Accept(restEvent)
	}
	countdownPayload := *payload.LongTermCountdown
	countdownPayloadJSON, _ := json.Marshal(countdownPayload)
	countdownEvent := command.NewEvent(cmd, EventTypeCountdownUpdated, "countdown", countdownPayload.CountdownID, countdownPayloadJSON, now().UTC())
	return command.Accept(restEvent, countdownEvent)
}

func decideCountdownCreate(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, EventTypeCountdownCreated, "countdown",
		func(p *CountdownCreatePayload) string { return strings.TrimSpace(p.CountdownID) },
		func(p *CountdownCreatePayload, _ func() time.Time) *command.Rejection {
			p.CountdownID = strings.TrimSpace(p.CountdownID)
			p.Name = strings.TrimSpace(p.Name)
			p.Kind = strings.TrimSpace(p.Kind)
			p.Direction = strings.TrimSpace(p.Direction)
			return nil
		}, now)
}

func decideCountdownUpdate(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncWithState(cmd, snapshotState, hasSnapshot, EventTypeCountdownUpdated, "countdown",
		func(p *CountdownUpdatePayload) string { return strings.TrimSpace(p.CountdownID) },
		func(s SnapshotState, hasState bool, p *CountdownUpdatePayload, _ func() time.Time) *command.Rejection {
			if hasState {
				if rejection := countdownUpdateSnapshotRejection(s, *p); rejection != nil {
					return rejection
				}
			}
			p.CountdownID = strings.TrimSpace(p.CountdownID)
			p.Reason = strings.TrimSpace(p.Reason)
			return nil
		}, now)
}

func decideCountdownDelete(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, EventTypeCountdownDeleted, "countdown",
		func(p *CountdownDeletePayload) string { return strings.TrimSpace(p.CountdownID) },
		func(p *CountdownDeletePayload, _ func() time.Time) *command.Rejection {
			p.CountdownID = strings.TrimSpace(p.CountdownID)
			p.Reason = strings.TrimSpace(p.Reason)
			return nil
		}, now)
}

func isCountdownUpdateNoMutation(snapshot SnapshotState, payload CountdownUpdatePayload) bool {
	countdown, hasCountdown := snapshotCountdownState(snapshot, payload.CountdownID)
	if !hasCountdown {
		return false
	}
	if countdown.Current != payload.After {
		return false
	}
	if payload.Looped && !countdown.Looping {
		return false
	}
	return true
}

func countdownUpdateSnapshotRejection(snapshot SnapshotState, payload CountdownUpdatePayload) *command.Rejection {
	if countdown, hasCountdown := snapshotCountdownState(snapshot, payload.CountdownID); hasCountdown && payload.Before != countdown.Current {
		return &command.Rejection{
			Code:    rejectionCodeCountdownBeforeMismatch,
			Message: "countdown before does not match current state",
		}
	}
	if isCountdownUpdateNoMutation(snapshot, payload) {
		// FIXME(telemetry): metric for idempotent countdown updates.
		return &command.Rejection{
			Code:    rejectionCodeCountdownUpdateNoMutation,
			Message: "countdown update is unchanged",
		}
	}
	return nil
}

func snapshotCountdownState(snapshot SnapshotState, countdownID string) (CountdownState, bool) {
	countdownID = strings.TrimSpace(countdownID)
	if countdownID == "" {
		return CountdownState{}, false
	}
	countdown, ok := snapshot.CountdownStates[countdownID]
	if !ok {
		return CountdownState{}, false
	}
	countdown.CountdownID = countdownID
	countdown.CampaignID = snapshot.CampaignID
	return countdown, true
}
