package readiness

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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

// SessionStartWorkflow owns the one intentional cross-aggregate exception in
// the core write path: readiness-gated first-session activation.
type SessionStartWorkflow interface {
	Start(current aggregate.State, cmd command.Command, now func() time.Time) command.Decision
}

type sessionStartWorkflow struct {
	systems *module.Registry
}

// NewSessionStartWorkflow builds the readiness-owned workflow used for
// session.start orchestration.
func NewSessionStartWorkflow(systems *module.Registry) SessionStartWorkflow {
	return sessionStartWorkflow{systems: systems}
}

func (w sessionStartWorkflow) Start(current aggregate.State, cmd command.Command, now func() time.Time) command.Decision {
	if now == nil {
		now = time.Now
	}
	decisionTime := now().UTC()
	fixedNow := func() time.Time { return decisionTime }

	report := EvaluateSessionStartReport(current, ReportOptions{
		SystemReadiness:        w.systemReadiness(current),
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

func (w sessionStartWorkflow) systemReadiness(current aggregate.State) CharacterSystemReadiness {
	if w.systems == nil {
		return nil
	}
	systemID := strings.TrimSpace(string(current.Campaign.GameSystem))
	if systemID == "" {
		return nil
	}
	mod := w.systems.Get(systemID, "")
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
