package daggerheart

import (
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

func decideEnvironmentEntityCreate(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncWithState(cmd, snapshotState, hasSnapshot, EventTypeEnvironmentEntityCreated, "environment_entity",
		func(p *EnvironmentEntityCreatePayload) string {
			return strings.TrimSpace(p.EnvironmentEntityID.String())
		},
		func(s SnapshotState, hasState bool, p *EnvironmentEntityCreatePayload, _ func() time.Time) *command.Rejection {
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
	return module.DecideFunc(cmd, EventTypeEnvironmentEntityUpdated, "environment_entity",
		func(p *EnvironmentEntityUpdatePayload) string {
			return strings.TrimSpace(p.EnvironmentEntityID.String())
		},
		func(p *EnvironmentEntityUpdatePayload, _ func() time.Time) *command.Rejection {
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
	return module.DecideFunc(cmd, EventTypeEnvironmentEntityDeleted, "environment_entity",
		func(p *EnvironmentEntityDeletePayload) string {
			return strings.TrimSpace(p.EnvironmentEntityID.String())
		},
		func(p *EnvironmentEntityDeletePayload, _ func() time.Time) *command.Rejection {
			p.EnvironmentEntityID = ids.EnvironmentEntityID(strings.TrimSpace(p.EnvironmentEntityID.String()))
			p.Reason = strings.TrimSpace(p.Reason)
			return nil
		}, now)
}

func isEnvironmentEntityCreateNoMutation(snapshot SnapshotState, payload EnvironmentEntityCreatePayload) bool {
	environmentEntity, hasEnvironmentEntity := snapshotEnvironmentEntityState(snapshot, payload.EnvironmentEntityID)
	if !hasEnvironmentEntity {
		return false
	}
	return environmentEntity.EnvironmentID == strings.TrimSpace(payload.EnvironmentID) &&
		environmentEntity.Name == strings.TrimSpace(payload.Name) &&
		environmentEntity.Type == strings.TrimSpace(payload.Type) &&
		environmentEntity.Tier == payload.Tier &&
		environmentEntity.Difficulty == payload.Difficulty &&
		environmentEntity.SessionID == ids.SessionID(strings.TrimSpace(payload.SessionID.String())) &&
		environmentEntity.SceneID == ids.SceneID(strings.TrimSpace(payload.SceneID.String())) &&
		environmentEntity.Notes == strings.TrimSpace(payload.Notes)
}

func snapshotEnvironmentEntityState(snapshot SnapshotState, environmentEntityID ids.EnvironmentEntityID) (EnvironmentEntityState, bool) {
	trimmed := ids.EnvironmentEntityID(strings.TrimSpace(environmentEntityID.String()))
	if trimmed == "" {
		return EnvironmentEntityState{}, false
	}
	environmentEntity, ok := snapshot.EnvironmentStates[trimmed]
	if !ok {
		return EnvironmentEntityState{}, false
	}
	environmentEntity.EnvironmentEntityID = trimmed
	environmentEntity.CampaignID = snapshot.CampaignID
	return environmentEntity, true
}
