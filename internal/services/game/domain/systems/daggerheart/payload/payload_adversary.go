package payload

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

// --- Adversary CRUD ---

// AdversaryCreatePayload captures the payload for sys.daggerheart.adversary.create commands.
type AdversaryCreatePayload struct {
	AdversaryID       dhids.AdversaryID                 `json:"adversary_id"`
	AdversaryEntryID  string                            `json:"adversary_entry_id"`
	Name              string                            `json:"name"`
	Kind              string                            `json:"kind,omitempty"`
	SessionID         ids.SessionID                     `json:"session_id"`
	SceneID           ids.SceneID                       `json:"scene_id"`
	Notes             string                            `json:"notes,omitempty"`
	HP                int                               `json:"hp"`
	HPMax             int                               `json:"hp_max"`
	Stress            int                               `json:"stress"`
	StressMax         int                               `json:"stress_max"`
	Evasion           int                               `json:"evasion"`
	Major             int                               `json:"major_threshold"`
	Severe            int                               `json:"severe_threshold"`
	Armor             int                               `json:"armor"`
	FeatureStates     []rules.AdversaryFeatureState     `json:"feature_states,omitempty"`
	PendingExperience *rules.AdversaryPendingExperience `json:"pending_experience,omitempty"`
	SpotlightGateID   ids.GateID                        `json:"spotlight_gate_id,omitempty"`
	SpotlightCount    int                               `json:"spotlight_count,omitempty"`
}

// AdversaryCreatedPayload captures the payload for sys.daggerheart.adversary_created events.
type AdversaryCreatedPayload = AdversaryCreatePayload

// AdversaryUpdatePayload captures the payload for sys.daggerheart.adversary.update commands.
type AdversaryUpdatePayload struct {
	AdversaryID       dhids.AdversaryID                 `json:"adversary_id"`
	AdversaryEntryID  string                            `json:"adversary_entry_id"`
	Name              string                            `json:"name"`
	Kind              string                            `json:"kind,omitempty"`
	SessionID         ids.SessionID                     `json:"session_id"`
	SceneID           ids.SceneID                       `json:"scene_id"`
	Notes             string                            `json:"notes,omitempty"`
	HP                int                               `json:"hp"`
	HPMax             int                               `json:"hp_max"`
	Stress            int                               `json:"stress"`
	StressMax         int                               `json:"stress_max"`
	Evasion           int                               `json:"evasion"`
	Major             int                               `json:"major_threshold"`
	Severe            int                               `json:"severe_threshold"`
	Armor             int                               `json:"armor"`
	FeatureStates     []rules.AdversaryFeatureState     `json:"feature_states,omitempty"`
	PendingExperience *rules.AdversaryPendingExperience `json:"pending_experience,omitempty"`
	SpotlightGateID   ids.GateID                        `json:"spotlight_gate_id,omitempty"`
	SpotlightCount    int                               `json:"spotlight_count,omitempty"`
}

// AdversaryFeatureApplyPayload captures one supported adversary feature state
// mutation and the resulting adversary projection update.
type AdversaryFeatureApplyPayload struct {
	ActorAdversaryID        dhids.AdversaryID                 `json:"actor_adversary_id"`
	AdversaryID             dhids.AdversaryID                 `json:"adversary_id"`
	FeatureID               string                            `json:"feature_id"`
	TargetCharacterID       ids.CharacterID                   `json:"target_character_id,omitempty"`
	TargetAdversaryID       dhids.AdversaryID                 `json:"target_adversary_id,omitempty"`
	StressBefore            *int                              `json:"stress_before,omitempty"`
	StressAfter             *int                              `json:"stress_after,omitempty"`
	FeatureStatesBefore     []rules.AdversaryFeatureState     `json:"feature_states_before,omitempty"`
	FeatureStatesAfter      []rules.AdversaryFeatureState     `json:"feature_states_after,omitempty"`
	PendingExperienceBefore *rules.AdversaryPendingExperience `json:"pending_experience_before,omitempty"`
	PendingExperienceAfter  *rules.AdversaryPendingExperience `json:"pending_experience_after,omitempty"`
}

// AdversaryUpdatedPayload captures the payload for sys.daggerheart.adversary_updated events.
type AdversaryUpdatedPayload = AdversaryUpdatePayload

// AdversaryDeletePayload captures the payload for sys.daggerheart.adversary.delete commands.
type AdversaryDeletePayload struct {
	AdversaryID dhids.AdversaryID `json:"adversary_id"`
	Reason      string            `json:"reason,omitempty"`
}

// AdversaryDeletedPayload captures the payload for sys.daggerheart.adversary_deleted events.
type AdversaryDeletedPayload = AdversaryDeletePayload
