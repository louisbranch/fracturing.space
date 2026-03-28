package decider

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/normalize"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func snapshotSceneCountdownState(snapshot daggerheartstate.SnapshotState, countdownID dhids.CountdownID) (daggerheartstate.SceneCountdownState, bool) {
	value, ok := snapshot.SceneCountdownStates[normalize.ID(countdownID)]
	return value, ok
}

func snapshotCampaignCountdownState(snapshot daggerheartstate.SnapshotState, countdownID dhids.CountdownID) (daggerheartstate.CampaignCountdownState, bool) {
	value, ok := snapshot.CampaignCountdownStates[normalize.ID(countdownID)]
	return value, ok
}

func decideRestTake(snapshotState daggerheartstate.SnapshotState, cmd command.Command, now func() time.Time) command.Decision {
	var p payload.RestTakePayload
	if err := json.Unmarshal(cmd.PayloadJSON, &p); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	now = command.RequireNowFunc(now)
	p.RestType = normalize.String(p.RestType)
	for i := range p.CampaignCountdownAdvances {
		if rejection := campaignCountdownAdvanceSnapshotRejection(snapshotState, p.CampaignCountdownAdvances[i]); rejection != nil {
			return command.Reject(*rejection)
		}
		p.CampaignCountdownAdvances[i].CountdownID = normalize.ID(p.CampaignCountdownAdvances[i].CountdownID)
		p.CampaignCountdownAdvances[i].Reason = normalize.String(p.CampaignCountdownAdvances[i].Reason)
	}

	eventPayload := payload.RestTakenPayload{
		RestType:        p.RestType,
		Interrupted:     p.Interrupted,
		GMFear:          p.GMFearAfter,
		ShortRests:      p.ShortRestsAfter,
		RefreshRest:     p.RefreshRest,
		RefreshLongRest: p.RefreshLongRest,
	}
	eventPayload.Participants = append(eventPayload.Participants, p.Participants...)
	eventPayloadJSON, _ := json.Marshal(eventPayload)
	entityID := normalize.String(cmd.EntityID)
	if entityID == "" {
		entityID = string(cmd.CampaignID)
	}
	restEvent := command.NewEvent(cmd, payload.EventTypeRestTaken, "session", entityID, eventPayloadJSON, now().UTC())
	events := []event.Event{restEvent}

	for _, move := range p.DowntimeMoves {
		move.ActorCharacterID = normalize.ID(move.ActorCharacterID)
		move.TargetCharacterID = normalize.ID(move.TargetCharacterID)
		move.CampaignCountdownID = normalize.ID(move.CampaignCountdownID)
		move.GroupID = normalize.String(move.GroupID)
		move.RestType = normalize.String(move.RestType)
		move.Move = normalize.String(move.Move)
		movePayloadJSON, _ := json.Marshal(move)
		entityID := move.ActorCharacterID.String()
		if entityID == "" {
			entityID = string(cmd.CampaignID)
		}
		events = append(events, command.NewEvent(cmd, payload.EventTypeDowntimeMoveApplied, "character", entityID, movePayloadJSON, now().UTC()))
	}

	for _, src := range p.CampaignCountdownAdvances {
		countdownPayloadJSON, _ := json.Marshal(payload.CampaignCountdownAdvancedPayload(src))
		events = append(events, command.NewEvent(cmd, payload.EventTypeCampaignCountdownAdvanced, "campaign_countdown", src.CountdownID.String(), countdownPayloadJSON, now().UTC()))
	}

	return command.Accept(events...)
}

func decideSceneCountdownCreate(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, payload.EventTypeSceneCountdownCreated, "scene_countdown",
		func(p *payload.SceneCountdownCreatePayload) string { return normalize.ID(p.CountdownID).String() },
		func(p *payload.SceneCountdownCreatePayload, _ func() time.Time) *command.Rejection {
			return normalizeCountdownCreatePayload(p)
		},
		now,
	)
}

func decideCampaignCountdownCreate(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, payload.EventTypeCampaignCountdownCreated, "campaign_countdown",
		func(p *payload.CampaignCountdownCreatePayload) string { return normalize.ID(p.CountdownID).String() },
		func(p *payload.CampaignCountdownCreatePayload, _ func() time.Time) *command.Rejection {
			value := payload.SceneCountdownCreatePayload(*p)
			rejection := normalizeCountdownCreatePayload(&value)
			*p = payload.CampaignCountdownCreatePayload(value)
			return rejection
		},
		now,
	)
}

func decideSceneCountdownAdvance(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot, payload.EventTypeSceneCountdownAdvanced, "scene_countdown",
		func(p *payload.SceneCountdownAdvancePayload) string { return normalize.ID(p.CountdownID).String() },
		func(s daggerheartstate.SnapshotState, hasState bool, p *payload.SceneCountdownAdvancePayload, _ func() time.Time) *command.Rejection {
			if hasState {
				if rejection := sceneCountdownAdvanceSnapshotRejection(s, *p); rejection != nil {
					return rejection
				}
			}
			normalizeCountdownAdvancePayload(p)
			return nil
		},
		func(_ daggerheartstate.SnapshotState, _ bool, p payload.SceneCountdownAdvancePayload) payload.SceneCountdownAdvancedPayload {
			return payload.SceneCountdownAdvancedPayload(p)
		},
		now,
	)
}

func decideCampaignCountdownAdvance(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot, payload.EventTypeCampaignCountdownAdvanced, "campaign_countdown",
		func(p *payload.CampaignCountdownAdvancePayload) string { return normalize.ID(p.CountdownID).String() },
		func(s daggerheartstate.SnapshotState, hasState bool, p *payload.CampaignCountdownAdvancePayload, _ func() time.Time) *command.Rejection {
			if hasState {
				if rejection := campaignCountdownAdvanceSnapshotRejection(s, *p); rejection != nil {
					return rejection
				}
			}
			value := payload.SceneCountdownAdvancePayload(*p)
			normalizeCountdownAdvancePayload(&value)
			*p = payload.CampaignCountdownAdvancePayload(value)
			return nil
		},
		func(_ daggerheartstate.SnapshotState, _ bool, p payload.CampaignCountdownAdvancePayload) payload.CampaignCountdownAdvancedPayload {
			return payload.CampaignCountdownAdvancedPayload(p)
		},
		now,
	)
}

func decideSceneCountdownTriggerResolve(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot, payload.EventTypeSceneCountdownTriggerResolved, "scene_countdown",
		func(p *payload.SceneCountdownTriggerResolvePayload) string {
			return normalize.ID(p.CountdownID).String()
		},
		func(s daggerheartstate.SnapshotState, hasState bool, p *payload.SceneCountdownTriggerResolvePayload, _ func() time.Time) *command.Rejection {
			if hasState {
				if rejection := sceneCountdownTriggerResolveSnapshotRejection(s, *p); rejection != nil {
					return rejection
				}
			}
			normalizeCountdownTriggerResolvePayload(p)
			return nil
		},
		func(_ daggerheartstate.SnapshotState, _ bool, p payload.SceneCountdownTriggerResolvePayload) payload.SceneCountdownTriggerResolvedPayload {
			return payload.SceneCountdownTriggerResolvedPayload(p)
		},
		now,
	)
}

func decideCampaignCountdownTriggerResolve(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot, payload.EventTypeCampaignCountdownTriggerResolved, "campaign_countdown",
		func(p *payload.CampaignCountdownTriggerResolvePayload) string {
			return normalize.ID(p.CountdownID).String()
		},
		func(s daggerheartstate.SnapshotState, hasState bool, p *payload.CampaignCountdownTriggerResolvePayload, _ func() time.Time) *command.Rejection {
			if hasState {
				if rejection := campaignCountdownTriggerResolveSnapshotRejection(s, *p); rejection != nil {
					return rejection
				}
			}
			value := payload.SceneCountdownTriggerResolvePayload(*p)
			normalizeCountdownTriggerResolvePayload(&value)
			*p = payload.CampaignCountdownTriggerResolvePayload(value)
			return nil
		},
		func(_ daggerheartstate.SnapshotState, _ bool, p payload.CampaignCountdownTriggerResolvePayload) payload.CampaignCountdownTriggerResolvedPayload {
			return payload.CampaignCountdownTriggerResolvedPayload(p)
		},
		now,
	)
}

func decideSceneCountdownDelete(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, payload.EventTypeSceneCountdownDeleted, "scene_countdown",
		func(p *payload.SceneCountdownDeletePayload) string { return normalize.ID(p.CountdownID).String() },
		func(p *payload.SceneCountdownDeletePayload, _ func() time.Time) *command.Rejection {
			normalizeCountdownDeletePayload(p)
			return nil
		},
		now,
	)
}

func decideCampaignCountdownDelete(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, payload.EventTypeCampaignCountdownDeleted, "campaign_countdown",
		func(p *payload.CampaignCountdownDeletePayload) string { return normalize.ID(p.CountdownID).String() },
		func(p *payload.CampaignCountdownDeletePayload, _ func() time.Time) *command.Rejection {
			value := payload.SceneCountdownDeletePayload(*p)
			normalizeCountdownDeletePayload(&value)
			*p = payload.CampaignCountdownDeletePayload(value)
			return nil
		},
		now,
	)
}

func normalizeCountdownCreatePayload(p *payload.SceneCountdownCreatePayload) *command.Rejection {
	p.SessionID = normalize.ID(p.SessionID)
	p.SceneID = normalize.ID(p.SceneID)
	p.CountdownID = normalize.ID(p.CountdownID)
	p.Name = normalize.String(p.Name)
	p.Tone = normalize.String(p.Tone)
	p.AdvancementPolicy = normalize.String(p.AdvancementPolicy)
	p.LoopBehavior = normalize.String(p.LoopBehavior)
	p.Status = normalize.String(p.Status)
	p.LinkedCountdownID = normalize.ID(p.LinkedCountdownID)
	if p.RemainingValue <= 0 {
		p.RemainingValue = p.StartingValue
	}
	if p.LoopBehavior == "" {
		p.LoopBehavior = "none"
	}
	if p.RemainingValue <= 0 {
		p.RemainingValue = p.StartingValue
	}
	return nil
}

func normalizeCountdownAdvancePayload(p *payload.SceneCountdownAdvancePayload) {
	p.CountdownID = normalize.ID(p.CountdownID)
	p.StatusBefore = normalize.String(p.StatusBefore)
	p.StatusAfter = normalize.String(p.StatusAfter)
	p.Reason = normalize.String(p.Reason)
}

func normalizeCountdownTriggerResolvePayload(p *payload.SceneCountdownTriggerResolvePayload) {
	p.CountdownID = normalize.ID(p.CountdownID)
	p.StatusBefore = normalize.String(p.StatusBefore)
	p.StatusAfter = normalize.String(p.StatusAfter)
	p.Reason = normalize.String(p.Reason)
}

func normalizeCountdownDeletePayload(p *payload.SceneCountdownDeletePayload) {
	p.CountdownID = normalize.ID(p.CountdownID)
	p.Reason = normalize.String(p.Reason)
}

func isSceneCountdownAdvanceNoMutation(snapshot daggerheartstate.SnapshotState, p payload.SceneCountdownAdvancePayload) bool {
	countdown, hasCountdown := snapshotSceneCountdownState(snapshot, p.CountdownID)
	if !hasCountdown {
		return false
	}
	return countdown.RemainingValue == p.AfterRemaining && countdown.Status == p.StatusAfter
}

func isCampaignCountdownAdvanceNoMutation(snapshot daggerheartstate.SnapshotState, p payload.CampaignCountdownAdvancePayload) bool {
	countdown, hasCountdown := snapshotCampaignCountdownState(snapshot, p.CountdownID)
	if !hasCountdown {
		return false
	}
	return countdown.RemainingValue == p.AfterRemaining && countdown.Status == p.StatusAfter
}

func sceneCountdownAdvanceSnapshotRejection(snapshot daggerheartstate.SnapshotState, p payload.SceneCountdownAdvancePayload) *command.Rejection {
	countdown, ok := snapshotSceneCountdownState(snapshot, p.CountdownID)
	if !ok {
		return nil
	}
	if countdown.RemainingValue != p.BeforeRemaining {
		return &command.Rejection{Code: rejectionCodeCountdownBeforeMismatch, Message: "scene countdown before_remaining does not match snapshot"}
	}
	if isSceneCountdownAdvanceNoMutation(snapshot, p) {
		return &command.Rejection{Code: rejectionCodeCountdownAdvanceNoMutation, Message: "scene countdown advance does not change state"}
	}
	return nil
}

func campaignCountdownAdvanceSnapshotRejection(snapshot daggerheartstate.SnapshotState, p payload.CampaignCountdownAdvancePayload) *command.Rejection {
	countdown, ok := snapshotCampaignCountdownState(snapshot, p.CountdownID)
	if !ok {
		return nil
	}
	if countdown.RemainingValue != p.BeforeRemaining {
		return &command.Rejection{Code: rejectionCodeCountdownBeforeMismatch, Message: "campaign countdown before_remaining does not match snapshot"}
	}
	if isCampaignCountdownAdvanceNoMutation(snapshot, p) {
		return &command.Rejection{Code: rejectionCodeCountdownAdvanceNoMutation, Message: "campaign countdown advance does not change state"}
	}
	return nil
}

func sceneCountdownTriggerResolveSnapshotRejection(snapshot daggerheartstate.SnapshotState, p payload.SceneCountdownTriggerResolvePayload) *command.Rejection {
	countdown, ok := snapshotSceneCountdownState(snapshot, p.CountdownID)
	if !ok {
		return nil
	}
	if countdown.Status != p.StatusBefore {
		return &command.Rejection{Code: rejectionCodeCountdownBeforeMismatch, Message: "scene countdown status_before does not match snapshot"}
	}
	return nil
}

func campaignCountdownTriggerResolveSnapshotRejection(snapshot daggerheartstate.SnapshotState, p payload.CampaignCountdownTriggerResolvePayload) *command.Rejection {
	countdown, ok := snapshotCampaignCountdownState(snapshot, p.CountdownID)
	if !ok {
		return nil
	}
	if countdown.Status != p.StatusBefore {
		return &command.Rejection{Code: rejectionCodeCountdownBeforeMismatch, Message: "campaign countdown status_before does not match snapshot"}
	}
	return nil
}
