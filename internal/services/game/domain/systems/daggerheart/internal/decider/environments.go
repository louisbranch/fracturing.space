package decider

import (
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/snapstate"
)

func decideEnvironmentEntityCreate(snapshotState snapstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncWithState(cmd, snapshotState, hasSnapshot, payload.EventTypeEnvironmentEntityCreated, "environment_entity",
		func(p *payload.EnvironmentEntityCreatePayload) string {
			return strings.TrimSpace(p.EnvironmentEntityID.String())
		},
		func(s snapstate.SnapshotState, hasState bool, p *payload.EnvironmentEntityCreatePayload, _ func() time.Time) *command.Rejection {
			if hasState && isEnvironmentEntityCreateNoMutation(s, *p) {
				return &command.Rejection{
					Code:    rejectionCodeEnvironmentEntityCreateNoMutation,
					Message: "environment entity create is unchanged",
				}
			}
			p.EnvironmentEntityID = ids.EnvironmentEntityID(strings.TrimSpace(p.EnvironmentEntityID.String()))
			p.EnvironmentID = strings.TrimSpace(p.EnvironmentID)
			p.Name = strings.TrimSpace(p.Name)
			p.Type = strings.TrimSpace(p.Type)
			p.SessionID = ids.SessionID(strings.TrimSpace(p.SessionID.String()))
			p.SceneID = ids.SceneID(strings.TrimSpace(p.SceneID.String()))
			p.Notes = strings.TrimSpace(p.Notes)
			return nil
		}, now)
}

func decideEnvironmentEntityUpdate(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, payload.EventTypeEnvironmentEntityUpdated, "environment_entity",
		func(p *payload.EnvironmentEntityUpdatePayload) string {
			return strings.TrimSpace(p.EnvironmentEntityID.String())
		},
		func(p *payload.EnvironmentEntityUpdatePayload, _ func() time.Time) *command.Rejection {
			p.EnvironmentEntityID = ids.EnvironmentEntityID(strings.TrimSpace(p.EnvironmentEntityID.String()))
			p.EnvironmentID = strings.TrimSpace(p.EnvironmentID)
			p.Name = strings.TrimSpace(p.Name)
			p.Type = strings.TrimSpace(p.Type)
			p.SessionID = ids.SessionID(strings.TrimSpace(p.SessionID.String()))
			p.SceneID = ids.SceneID(strings.TrimSpace(p.SceneID.String()))
			p.Notes = strings.TrimSpace(p.Notes)
			return nil
		}, now)
}

func decideEnvironmentEntityDelete(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, payload.EventTypeEnvironmentEntityDeleted, "environment_entity",
		func(p *payload.EnvironmentEntityDeletePayload) string {
			return strings.TrimSpace(p.EnvironmentEntityID.String())
		},
		func(p *payload.EnvironmentEntityDeletePayload, _ func() time.Time) *command.Rejection {
			p.EnvironmentEntityID = ids.EnvironmentEntityID(strings.TrimSpace(p.EnvironmentEntityID.String()))
			p.Reason = strings.TrimSpace(p.Reason)
			return nil
		}, now)
}

// ── File-local helpers ─────────────────────────────────────────────────

func isEnvironmentEntityCreateNoMutation(snapshot snapstate.SnapshotState, p payload.EnvironmentEntityCreatePayload) bool {
	environmentEntity, hasEnvironmentEntity := snapshotEnvironmentEntityState(snapshot, p.EnvironmentEntityID)
	if !hasEnvironmentEntity {
		return false
	}
	return environmentEntity.EnvironmentID == strings.TrimSpace(p.EnvironmentID) &&
		environmentEntity.Name == strings.TrimSpace(p.Name) &&
		environmentEntity.Type == strings.TrimSpace(p.Type) &&
		environmentEntity.Tier == p.Tier &&
		environmentEntity.Difficulty == p.Difficulty &&
		environmentEntity.SessionID == ids.SessionID(strings.TrimSpace(p.SessionID.String())) &&
		environmentEntity.SceneID == ids.SceneID(strings.TrimSpace(p.SceneID.String())) &&
		environmentEntity.Notes == strings.TrimSpace(p.Notes)
}

func snapshotEnvironmentEntityState(snapshot snapstate.SnapshotState, environmentEntityID ids.EnvironmentEntityID) (snapstate.EnvironmentEntityState, bool) {
	trimmed := ids.EnvironmentEntityID(strings.TrimSpace(environmentEntityID.String()))
	if trimmed == "" {
		return snapstate.EnvironmentEntityState{}, false
	}
	environmentEntity, ok := snapshot.EnvironmentStates[trimmed]
	if !ok {
		return snapstate.EnvironmentEntityState{}, false
	}
	environmentEntity.EnvironmentEntityID = trimmed
	environmentEntity.CampaignID = snapshot.CampaignID
	return environmentEntity, true
}
