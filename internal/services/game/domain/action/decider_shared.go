package action

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// acceptActionEvent creates the standard action event envelope for accepted commands.
//
// Centralizing this constructor keeps action event metadata consistent even when
// specific systems add new action shapes.
func acceptActionEvent(cmd command.Command, now func() time.Time, eventType event.Type, entityType, entityID string, payload any) command.Decision {
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, eventType, entityType, entityID, payloadJSON, now().UTC())

	return command.Accept(evt)
}

func buildOutcomeEffectEvent(cmd command.Command, now func() time.Time, effect OutcomeAppliedEffect) event.Event {
	payloadJSON := effect.PayloadJSON
	if len(payloadJSON) == 0 {
		payloadJSON = []byte("{}")
	}
	evt := command.NewEvent(
		cmd,
		event.Type(strings.TrimSpace(effect.Type)),
		strings.TrimSpace(effect.EntityType),
		strings.TrimSpace(effect.EntityID),
		payloadJSON,
		now().UTC(),
	)
	evt.SystemID = strings.TrimSpace(effect.SystemID)
	evt.SystemVersion = strings.TrimSpace(effect.SystemVersion)
	return evt
}

type outcomeEffectPolicy struct {
	allowed map[string]struct{}
}

func newOutcomeEffectPolicy(allowed ...string) outcomeEffectPolicy {
	policy := outcomeEffectPolicy{allowed: make(map[string]struct{}, len(allowed))}
	for _, effectType := range allowed {
		policy.allowed[strings.TrimSpace(effectType)] = struct{}{}
	}
	return policy
}

func (p outcomeEffectPolicy) hasSystemOwned(effects []OutcomeAppliedEffect) bool {
	for _, effect := range effects {
		if strings.HasPrefix(strings.TrimSpace(effect.Type), "sys.") {
			return true
		}
		if strings.TrimSpace(effect.SystemID) != "" || strings.TrimSpace(effect.SystemVersion) != "" {
			return true
		}
	}
	return false
}

func (p outcomeEffectPolicy) hasDisallowed(effects []OutcomeAppliedEffect) bool {
	for _, effect := range effects {
		if !p.isAllowed(strings.TrimSpace(effect.Type)) {
			return true
		}
	}
	return false
}

func (p outcomeEffectPolicy) isAllowed(effectType string) bool {
	_, ok := p.allowed[strings.TrimSpace(effectType)]
	return ok
}

func hasSystemOwnedOutcomeEffect(effects []OutcomeAppliedEffect) bool {
	return coreOutcomeEffectPolicy.hasSystemOwned(effects)
}

func hasDisallowedCoreOutcomeEffect(effects []OutcomeAppliedEffect) bool {
	return coreOutcomeEffectPolicy.hasDisallowed(effects)
}
