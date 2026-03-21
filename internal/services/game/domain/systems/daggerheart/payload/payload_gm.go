package payload

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

// --- GM Fear ---

// GMFearSetPayload captures the payload for sys.daggerheart.gm_fear.set commands.
type GMFearSetPayload struct {
	After  *int   `json:"after,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// GMFearChangedPayload captures the payload for sys.daggerheart.gm_fear_changed events.
type GMFearChangedPayload struct {
	Value  int    `json:"after"`
	Reason string `json:"reason,omitempty"`
}

// --- GM Moves ---

// GMMoveTarget captures the typed Fear-spend target stored on GM move
// commands and audit events.
type GMMoveTarget struct {
	Type                rules.GMMoveTargetType  `json:"type"`
	Kind                rules.GMMoveKind        `json:"kind,omitempty"`
	Shape               rules.GMMoveShape       `json:"shape,omitempty"`
	Description         string                  `json:"description,omitempty"`
	AdversaryID         ids.AdversaryID         `json:"adversary_id,omitempty"`
	EnvironmentEntityID ids.EnvironmentEntityID `json:"environment_entity_id,omitempty"`
	EnvironmentID       string                  `json:"environment_id,omitempty"`
	FeatureID           string                  `json:"feature_id,omitempty"`
	ExperienceName      string                  `json:"experience_name,omitempty"`
}

// GMMoveApplyPayload captures the payload for sys.daggerheart.gm_move.apply
// commands.
type GMMoveApplyPayload struct {
	Target    GMMoveTarget `json:"target"`
	FearSpent int          `json:"fear_spent"`
}

// GMMoveAppliedPayload captures the payload for sys.daggerheart.gm_move_applied
// events.
type GMMoveAppliedPayload struct {
	Target    GMMoveTarget `json:"target"`
	FearSpent int          `json:"fear_spent"`
}
