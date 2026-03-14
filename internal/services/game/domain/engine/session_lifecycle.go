package engine

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/readiness"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

// SessionLifecycle owns the cross-aggregate lifecycle exception for
// session.start so CoreDecider can stay focused on routing.
type SessionLifecycle interface {
	Start(current aggregate.State, cmd command.Command, now func() time.Time) command.Decision
}

type sessionLifecycle struct {
	systems *module.Registry
}

// NewSessionLifecycle builds the lifecycle policy used for session.start.
func NewSessionLifecycle(systems *module.Registry) SessionLifecycle {
	return sessionLifecycle{systems: systems}
}

func (l sessionLifecycle) Start(current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	if now == nil {
		now = time.Now
	}
	decisionTime := now().UTC()
	fixedNow := func() time.Time { return decisionTime }

	report := readiness.EvaluateSessionStartReport(current, readiness.ReportOptions{
		SystemReadiness:        l.systemReadiness(current),
		IncludeSessionBoundary: true,
		HasActiveSession:       current.Session.Started,
	})
	if !report.Ready() {
		first := report.Blockers[0]
		return command.Reject(command.Rejection{Code: first.Code, Message: first.Message})
	}

	startDecision := session.Decide(current.Session, cmd, fixedNow)
	if len(startDecision.Rejections) > 0 {
		return startDecision
	}

	if current.Campaign.Status != campaign.StatusDraft {
		return startDecision
	}

	campaignPayloadJSON, _ := json.Marshal(campaign.UpdatePayload{Fields: map[string]string{"status": string(campaign.StatusActive)}})
	campaignActivated := command.NewEvent(
		cmd,
		campaign.EventTypeUpdated,
		"campaign",
		string(cmd.CampaignID),
		campaignPayloadJSON,
		decisionTime,
	)

	events := make([]event.Event, 0, len(startDecision.Events)+1)
	events = append(events, campaignActivated)
	events = append(events, startDecision.Events...)
	return command.Accept(events...)
}

func (l sessionLifecycle) systemReadiness(current aggregate.State) readiness.CharacterSystemReadiness {
	if l.systems == nil {
		return nil
	}
	systemID := strings.TrimSpace(string(current.Campaign.GameSystem))
	if systemID == "" {
		return nil
	}
	mod := l.systems.Get(systemID, "")
	if mod == nil {
		return nil
	}
	checker, ok := mod.(module.CharacterReadinessChecker)
	if !ok {
		return nil
	}
	systemState := current.Systems[module.Key{ID: mod.ID(), Version: mod.Version()}]
	return func(characterID string) (bool, string) {
		ch, ok := current.Characters[ids.CharacterID(characterID)]
		if !ok {
			return false, "character is missing"
		}
		return checker.CharacterReady(systemState, ch)
	}
}
