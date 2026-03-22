package payload

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
)

// --- Roll ---

// RollRngInfo captures RNG metadata for roll events.
type RollRngInfo struct {
	SeedUsed   uint64 `json:"seed_used"`
	RngAlgo    string `json:"rng_algo"`
	SeedSource string `json:"seed_source"`
	RollMode   string `json:"roll_mode"`
}

// --- Damage ---

// DamageApplyPayload captures the payload for sys.daggerheart.damage.apply commands.
type DamageApplyPayload struct {
	CharacterID        ids.CharacterID   `json:"character_id"`
	HpBefore           *int              `json:"hp_before,omitempty"`
	HpAfter            *int              `json:"hp_after,omitempty"`
	StressAfter        *int              `json:"stress_after,omitempty"`
	ArmorBefore        *int              `json:"armor_before,omitempty"`
	ArmorAfter         *int              `json:"armor_after,omitempty"`
	ArmorSpent         int               `json:"armor_spent,omitempty"`
	Severity           string            `json:"severity,omitempty"`
	Marks              int               `json:"marks,omitempty"`
	DamageType         string            `json:"damage_type,omitempty"`
	RollSeq            *uint64           `json:"roll_seq,omitempty"`
	ResistPhysical     bool              `json:"resist_physical,omitempty"`
	ResistMagic        bool              `json:"resist_magic,omitempty"`
	ImmunePhysical     bool              `json:"immune_physical,omitempty"`
	ImmuneMagic        bool              `json:"immune_magic,omitempty"`
	Direct             bool              `json:"direct,omitempty"`
	MassiveDamage      bool              `json:"massive_damage,omitempty"`
	Mitigated          bool              `json:"mitigated,omitempty"`
	Source             string            `json:"source,omitempty"`
	SourceCharacterIDs []ids.CharacterID `json:"source_character_ids,omitempty"`
}

// DamageAppliedPayload captures the payload for sys.daggerheart.damage_applied events.
type DamageAppliedPayload struct {
	CharacterID        ids.CharacterID   `json:"character_id"`
	Hp                 *int              `json:"hp_after,omitempty"`
	Stress             *int              `json:"stress_after,omitempty"`
	Armor              *int              `json:"armor_after,omitempty"`
	ArmorSpent         int               `json:"armor_spent,omitempty"`
	Severity           string            `json:"severity,omitempty"`
	Marks              int               `json:"marks,omitempty"`
	DamageType         string            `json:"damage_type,omitempty"`
	RollSeq            *uint64           `json:"roll_seq,omitempty"`
	ResistPhysical     bool              `json:"resist_physical,omitempty"`
	ResistMagic        bool              `json:"resist_magic,omitempty"`
	ImmunePhysical     bool              `json:"immune_physical,omitempty"`
	ImmuneMagic        bool              `json:"immune_magic,omitempty"`
	Direct             bool              `json:"direct,omitempty"`
	MassiveDamage      bool              `json:"massive_damage,omitempty"`
	Mitigated          bool              `json:"mitigated,omitempty"`
	Source             string            `json:"source,omitempty"`
	SourceCharacterIDs []ids.CharacterID `json:"source_character_ids,omitempty"`
}

// MultiTargetDamageApplyPayload captures the payload for
// sys.daggerheart.multi_target_damage.apply commands.
type MultiTargetDamageApplyPayload struct {
	Targets []DamageApplyPayload `json:"targets"`
}

// --- Adversary damage ---

// AdversaryDamageApplyPayload captures the payload for sys.daggerheart.adversary_damage.apply commands.
type AdversaryDamageApplyPayload struct {
	AdversaryID        dhids.AdversaryID `json:"adversary_id"`
	HpBefore           *int              `json:"hp_before,omitempty"`
	HpAfter            *int              `json:"hp_after,omitempty"`
	ArmorBefore        *int              `json:"armor_before,omitempty"`
	ArmorAfter         *int              `json:"armor_after,omitempty"`
	ArmorSpent         int               `json:"armor_spent,omitempty"`
	Severity           string            `json:"severity,omitempty"`
	Marks              int               `json:"marks,omitempty"`
	DamageType         string            `json:"damage_type,omitempty"`
	RollSeq            *uint64           `json:"roll_seq,omitempty"`
	ResistPhysical     bool              `json:"resist_physical,omitempty"`
	ResistMagic        bool              `json:"resist_magic,omitempty"`
	ImmunePhysical     bool              `json:"immune_physical,omitempty"`
	ImmuneMagic        bool              `json:"immune_magic,omitempty"`
	Direct             bool              `json:"direct,omitempty"`
	MassiveDamage      bool              `json:"massive_damage,omitempty"`
	Mitigated          bool              `json:"mitigated,omitempty"`
	Source             string            `json:"source,omitempty"`
	SourceCharacterIDs []ids.CharacterID `json:"source_character_ids,omitempty"`
}

// AdversaryDamageAppliedPayload captures the payload for sys.daggerheart.adversary_damage_applied events.
type AdversaryDamageAppliedPayload struct {
	AdversaryID        dhids.AdversaryID `json:"adversary_id"`
	Hp                 *int              `json:"hp_after,omitempty"`
	Armor              *int              `json:"armor_after,omitempty"`
	ArmorSpent         int               `json:"armor_spent,omitempty"`
	Severity           string            `json:"severity,omitempty"`
	Marks              int               `json:"marks,omitempty"`
	DamageType         string            `json:"damage_type,omitempty"`
	RollSeq            *uint64           `json:"roll_seq,omitempty"`
	ResistPhysical     bool              `json:"resist_physical,omitempty"`
	ResistMagic        bool              `json:"resist_magic,omitempty"`
	ImmunePhysical     bool              `json:"immune_physical,omitempty"`
	ImmuneMagic        bool              `json:"immune_magic,omitempty"`
	Direct             bool              `json:"direct,omitempty"`
	MassiveDamage      bool              `json:"massive_damage,omitempty"`
	Mitigated          bool              `json:"mitigated,omitempty"`
	Source             string            `json:"source,omitempty"`
	SourceCharacterIDs []ids.CharacterID `json:"source_character_ids,omitempty"`
}

// --- Downtime ---

// DowntimeMoveAppliedPayload captures the payload for sys.daggerheart.downtime_move_applied events.
type DowntimeMoveAppliedPayload struct {
	ActorCharacterID  ids.CharacterID   `json:"actor_character_id"`
	TargetCharacterID ids.CharacterID   `json:"target_character_id,omitempty"`
	Move              string            `json:"move"`
	RestType          string            `json:"rest_type,omitempty"`
	GroupID           string            `json:"group_id,omitempty"`
	CountdownID       dhids.CountdownID `json:"countdown_id,omitempty"`
	HP                *int              `json:"hp_after,omitempty"`
	Hope              *int              `json:"hope_after,omitempty"`
	Stress            *int              `json:"stress_after,omitempty"`
	Armor             *int              `json:"armor_after,omitempty"`
}
