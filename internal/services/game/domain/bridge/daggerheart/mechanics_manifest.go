package daggerheart

import (
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// MechanicStatus describes whether a mechanic is implemented.
type MechanicStatus int

const (
	// MechanicPending indicates the mechanic is not yet implemented.
	MechanicPending MechanicStatus = iota
	// MechanicImplemented indicates the mechanic is implemented and tested.
	MechanicImplemented
	// MechanicNotApplicable indicates the mechanic is a content/data concern, not a server mechanic.
	MechanicNotApplicable
)

// MechanicRequirement declares whether a mechanic blocks the COMPLETE stage.
type MechanicRequirement int

const (
	// Required means the mechanic must be implemented for the system to be COMPLETE.
	Required MechanicRequirement = iota
	// Optional means the mechanic is not required for the COMPLETE stage.
	Optional
)

// MechanicCategory groups mechanics by PRD/SRD domain area.
type MechanicCategory int

const (
	CategoryResolution        MechanicCategory = iota // Dice, rolls, outcomes
	CategoryCharacterModel                            // Character creation, state, conditions
	CategoryCombatDamage                              // Damage application, armor, resistance
	CategoryRestDowntime                              // Rests, downtime moves
	CategoryDeathScars                                // Death moves, scars
	CategoryGMMechanics                               // Fear economy, adversaries, countdowns
	CategoryProgression                               // Leveling, multiclassing
	CategoryContentArchetypes                         // Domain cards, classes (data, not mechanics)
	CategoryOptionalRules                             // Optional variant rules
)

// Mechanic declares a single PRD/SRD mechanic with its implementation evidence.
type Mechanic struct {
	ID           string              // Unique identifier for this mechanic.
	Name         string              // Human-readable name.
	Category     MechanicCategory    // Grouping category.
	Status       MechanicStatus      // Implementation status.
	Requirement  MechanicRequirement // Whether this blocks COMPLETE stage.
	Commands     []command.Type      // System commands implementing this mechanic.
	Events       []event.Type        // Events emitted by this mechanic.
	ScenarioTags []string            // Scenario file basenames (without .lua) covering this mechanic.
	Notes        string              // Context about status or scope.
}

// MechanicsManifest returns the authoritative mechanic list for Daggerheart.
// Each entry maps a PRD/SRD mechanic to its implementation evidence (commands,
// events, scenario files). This manifest drives DeriveImplementationStage and
// is enforced by tests that break when code and manifest drift apart.
func MechanicsManifest() []Mechanic {
	return []Mechanic{
		// ── Resolution ──────────────────────────────────────────────────
		{
			ID:           "duality-dice",
			Name:         "Duality Dice (Action Rolls)",
			Category:     CategoryResolution,
			Status:       MechanicImplemented,
			Requirement:  Required,
			ScenarioTags: []string{"action_roll_outcomes", "action_roll_critical_success", "action_roll_failure_with_hope"},
			Notes:        "Core roll resolution: Hope/Fear duality, critical success. Driven by session roll infrastructure, no system command needed.",
		},
		{
			ID:           "reaction-rolls",
			Name:         "Reaction Rolls",
			Category:     CategoryResolution,
			Status:       MechanicImplemented,
			Requirement:  Required,
			ScenarioTags: []string{"reaction_flow", "direct_damage_reaction"},
			Notes:        "Reaction rolls triggered by adversary actions.",
		},
		{
			ID:           "advantage-disadvantage",
			Name:         "Advantage/Disadvantage",
			Category:     CategoryResolution,
			Status:       MechanicImplemented,
			Requirement:  Required,
			ScenarioTags: []string{"advantage_disguise_roll", "advantage_cancellation", "help_advantage_roll"},
			Notes:        "Extra d6 added/removed from roll pools.",
		},
		{
			ID:           "group-actions",
			Name:         "Group Actions",
			Category:     CategoryResolution,
			Status:       MechanicImplemented,
			Requirement:  Required,
			ScenarioTags: []string{"group_action", "group_action_escape", "group_finesse_sneak"},
			Notes:        "Multiple characters rolling together via gRPC SessionGroupAction.",
		},
		{
			ID:           "tag-team",
			Name:         "Tag Team Rolls",
			Category:     CategoryResolution,
			Status:       MechanicImplemented,
			Requirement:  Required,
			ScenarioTags: []string{"tag_team"},
			Notes:        "Two characters sharing a roll via gRPC SessionTagTeam.",
		},

		// ── Character Model ─────────────────────────────────────────────
		{
			ID:          "character-creation",
			Name:        "Character Creation (9-Step Workflow)",
			Category:    CategoryCharacterModel,
			Status:      MechanicImplemented,
			Requirement: Required,
			Notes:       "Full creation workflow with readiness gates. Validated by CharacterReady().",
		},
		{
			ID:          "character-state-patch",
			Name:        "Character State Patching",
			Category:    CategoryCharacterModel,
			Status:      MechanicImplemented,
			Requirement: Required,
			Commands:    []command.Type{commandTypeCharacterStatePatch},
			Events:      []event.Type{EventTypeCharacterStatePatched},
			Notes:       "HP, Hope, Stress, Armor, life state mutations.",
		},
		{
			ID:           "hope-economy",
			Name:         "Hope Economy",
			Category:     CategoryCharacterModel,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{commandTypeHopeSpend},
			Events:       []event.Type{EventTypeCharacterStatePatched},
			ScenarioTags: []string{"action_roll_failure_with_hope", "spellcast_hope_cost"},
			Notes:        "Hope spend command; gains via CharacterStatePatched.",
		},
		{
			ID:           "stress-economy",
			Name:         "Stress Economy",
			Category:     CategoryCharacterModel,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{commandTypeStressSpend},
			Events:       []event.Type{EventTypeCharacterStatePatched},
			ScenarioTags: []string{"companion_experience_stress_clear"},
			Notes:        "Stress spend command; gains via CharacterStatePatched.",
		},
		{
			ID:           "conditions",
			Name:         "Conditions (Hidden/Restrained/Vulnerable)",
			Category:     CategoryCharacterModel,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{commandTypeConditionChange},
			Events:       []event.Type{EventTypeConditionChanged},
			ScenarioTags: []string{"condition_lifecycle", "condition_stacking_guard", "hidden_condition_scouting"},
			Notes:        "Core conditions with add/remove lifecycle.",
		},
		{
			ID:           "loadout-swap",
			Name:         "Loadout Swap",
			Category:     CategoryCharacterModel,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{commandTypeLoadoutSwap},
			Events:       []event.Type{EventTypeLoadoutSwapped},
			ScenarioTags: []string{"loadout_swap"},
			Notes:        "Swap domain cards between vault and active loadout.",
		},

		// ── Combat & Damage ─────────────────────────────────────────────
		{
			ID:           "single-target-damage",
			Name:         "Single-Target Damage",
			Category:     CategoryCombatDamage,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{commandTypeDamageApply},
			Events:       []event.Type{EventTypeDamageApplied},
			ScenarioTags: []string{"damage_thresholds_example", "critical_damage", "critical_damage_maximum"},
			Notes:        "Damage application with severity evaluation.",
		},
		{
			ID:           "multi-target-damage",
			Name:         "Multi-Target Damage",
			Category:     CategoryCombatDamage,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{commandTypeMultiTargetDamageApply},
			ScenarioTags: []string{"multi_target_attack", "sweeping_attack_all_targets"},
			Notes:        "Damage applied to multiple characters in one command.",
		},
		{
			ID:           "adversary-damage",
			Name:         "Adversary Damage",
			Category:     CategoryCombatDamage,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{commandTypeAdversaryDamageApply},
			Events:       []event.Type{EventTypeAdversaryDamageApplied},
			ScenarioTags: []string{"adversary_spotlight", "adversary_spotlight_chain"},
			Notes:        "Damage dealt to adversaries.",
		},
		{
			ID:           "armor-mitigation",
			Name:         "Armor Slot Mitigation",
			Category:     CategoryCombatDamage,
			Status:       MechanicImplemented,
			Requirement:  Required,
			ScenarioTags: []string{"armor_mitigation", "armor_depletion", "fear_spotlight_armor_mitigation"},
			Notes:        "Armor spend wired into damage payloads.",
		},
		{
			ID:           "temporary-armor",
			Name:         "Temporary Armor",
			Category:     CategoryCombatDamage,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{commandTypeCharacterTemporaryArmorApply},
			Events:       []event.Type{EventTypeCharacterTemporaryArmorApplied},
			ScenarioTags: []string{"temporary_armor_bonus"},
			Notes:        "Temporary armor with duration-based expiry.",
		},
		{
			ID:           "damage-severity",
			Name:         "Damage Severity Thresholds",
			Category:     CategoryCombatDamage,
			Status:       MechanicImplemented,
			Requirement:  Required,
			ScenarioTags: []string{"damage_thresholds_example", "gm_move_severity"},
			Notes:        "EvaluateDamage() with Minor/Major/Severe/Massive thresholds.",
		},
		{
			ID:           "damage-rolls",
			Name:         "Damage Rolls (Dice Pool)",
			Category:     CategoryCombatDamage,
			Status:       MechanicImplemented,
			Requirement:  Required,
			ScenarioTags: []string{"damage_roll_modifier", "damage_roll_proficiency", "sam_critical_broadsword"},
			Notes:        "Dice pool + proficiency + crit bonus for damage resolution.",
		},
		{
			ID:           "resistance-immunity",
			Name:         "Resistance/Immunity",
			Category:     CategoryCombatDamage,
			Status:       MechanicImplemented,
			Requirement:  Required,
			ScenarioTags: []string{"help_and_resistance"},
			Notes:        "ApplyResistance() in damage domain.",
		},

		// ── Rest & Downtime ─────────────────────────────────────────────
		{
			ID:           "rest",
			Name:         "Rest (Short + Long)",
			Category:     CategoryRestDowntime,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{commandTypeRestTake},
			Events:       []event.Type{EventTypeRestTaken},
			ScenarioTags: []string{"rest_and_downtime"},
			Notes:        "Short and long rests with 3-short-before-long cadence enforced.",
		},
		{
			ID:           "downtime-moves",
			Name:         "Downtime Moves",
			Category:     CategoryRestDowntime,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{commandTypeDowntimeMoveApply},
			Events:       []event.Type{EventTypeDowntimeMoveApplied},
			ScenarioTags: []string{"rest_and_downtime"},
			Notes:        "Downtime move application with state changes.",
		},

		// ── Death & Scars ───────────────────────────────────────────────
		{
			ID:           "death-blaze-of-glory",
			Name:         "Death: Blaze of Glory",
			Category:     CategoryDeathScars,
			Status:       MechanicImplemented,
			Requirement:  Required,
			ScenarioTags: []string{"blaze_of_glory"},
			Notes:        "ResolveDeathMove with DeathMoveBlazeOfGlory.",
		},
		{
			ID:           "death-avoid-death",
			Name:         "Death: Avoid Death (Scars)",
			Category:     CategoryDeathScars,
			Status:       MechanicImplemented,
			Requirement:  Required,
			ScenarioTags: []string{"death_move"},
			Notes:        "ResolveDeathMove with DeathMoveAvoidDeath; scar reduces hope_max.",
		},
		{
			ID:           "death-risk-it-all",
			Name:         "Death: Risk It All",
			Category:     CategoryDeathScars,
			Status:       MechanicImplemented,
			Requirement:  Required,
			ScenarioTags: []string{"death_move"},
			Notes:        "ResolveDeathMove with DeathMoveRiskItAll; three outcome branches.",
		},

		// ── GM Mechanics ────────────────────────────────────────────────
		{
			ID:           "gm-fear",
			Name:         "GM Fear Economy",
			Category:     CategoryGMMechanics,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{commandTypeGMFearSet},
			Events:       []event.Type{EventTypeGMFearChanged},
			ScenarioTags: []string{"fear_floor", "gm_fear_spend_chain"},
			Notes:        "Fear set with 0-12 range enforcement.",
		},
		{
			ID:           "countdowns",
			Name:         "Countdowns (Create/Update/Delete)",
			Category:     CategoryGMMechanics,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{commandTypeCountdownCreate, commandTypeCountdownUpdate, commandTypeCountdownDelete},
			Events:       []event.Type{EventTypeCountdownCreated, EventTypeCountdownUpdated, EventTypeCountdownDeleted},
			ScenarioTags: []string{"countdown_lifecycle", "long_term_countdown", "progress_countdown_climb"},
			Notes:        "Base countdown system with looping support.",
		},
		{
			ID:          "adversary-crud",
			Name:        "Adversary CRUD",
			Category:    CategoryGMMechanics,
			Status:      MechanicImplemented,
			Requirement: Required,
			Commands:    []command.Type{commandTypeAdversaryCreate, commandTypeAdversaryUpdate, commandTypeAdversaryDelete},
			Events:      []event.Type{EventTypeAdversaryCreated, EventTypeAdversaryUpdated, EventTypeAdversaryDeleted},
			Notes:       "Create, update, delete adversaries in session.",
		},
		{
			ID:          "adversary-conditions",
			Name:        "Adversary Conditions",
			Category:    CategoryGMMechanics,
			Status:      MechanicImplemented,
			Requirement: Required,
			Commands:    []command.Type{commandTypeAdversaryConditionChange},
			Events:      []event.Type{EventTypeAdversaryConditionChanged},
			Notes:       "Add/remove conditions on adversaries.",
		},
		{
			ID:           "spotlight",
			Name:         "Spotlight System",
			Category:     CategoryGMMechanics,
			Status:       MechanicImplemented,
			Requirement:  Required,
			ScenarioTags: []string{"full_example_spotlight_sequence", "adversary_spotlight"},
			Notes:        "Core session spotlight infrastructure.",
		},

		// ── Progression (Pending) ───────────────────────────────────────
		{
			ID:           "leveling",
			Name:         "Leveling/Progression",
			Category:     CategoryProgression,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{commandTypeLevelUpApply},
			Events:       []event.Type{EventTypeLevelUpApplied},
			ScenarioTags: []string{"level_up_basic", "level_up_tier_entry"},
			Notes:        "Level-up with tier achievements, advancement budget, trait marking, and damage threshold progression.",
		},
		{
			ID:           "multiclassing",
			Name:         "Multiclassing",
			Category:     CategoryProgression,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{commandTypeLevelUpApply},
			Events:       []event.Type{EventTypeLevelUpApplied},
			ScenarioTags: []string{"multiclass_unlock"},
			Notes:        "Multiclass at level 5+ via level-up advancement. Secondary class/subclass/domain selection with foundation card.",
		},

		// ── Content/Archetypes (Not Applicable) ─────────────────────────
		{
			ID:          "content-catalogs",
			Name:        "Content Catalogs (Classes, Items, Domain Cards, Adversary Templates)",
			Category:    CategoryContentArchetypes,
			Status:      MechanicNotApplicable,
			Requirement: Optional,
			Notes:       "Data/content concerns, not server mechanics. SRD confirms domain card abilities compose existing mechanics (rolls, Hope/Stress, conditions, damage).",
		},

		// ── Optional Rules ──────────────────────────────────────────────
		{
			ID:           "downtime-move-limits",
			Name:         "Downtime Move Limits",
			Category:     CategoryOptionalRules,
			Status:       MechanicImplemented,
			Requirement:  Optional,
			Commands:     []command.Type{commandTypeDowntimeMoveApply},
			Events:       []event.Type{EventTypeDowntimeMoveApplied},
			ScenarioTags: []string{"downtime_move_limit"},
			Notes:        "Server-side 2-per-rest limit enforced via snapshot state counter. Reset on rest.",
		},
		{
			ID:           "equipment-management",
			Name:         "Equipment Management",
			Category:     CategoryOptionalRules,
			Status:       MechanicImplemented,
			Requirement:  Optional,
			Commands:     []command.Type{commandTypeEquipmentSwap},
			Events:       []event.Type{EventTypeEquipmentSwapped},
			ScenarioTags: []string{"equipment_swap"},
			Notes:        "In-session equip/unequip with slot tracking (active/inventory/none) and item type validation.",
		},
		{
			ID:           "domain-card-vault",
			Name:         "Domain Card Vault Management",
			Category:     CategoryOptionalRules,
			Status:       MechanicImplemented,
			Requirement:  Optional,
			Commands:     []command.Type{commandTypeDomainCardAcquire},
			Events:       []event.Type{EventTypeDomainCardAcquired},
			ScenarioTags: []string{"domain_card_acquire"},
			Notes:        "Card acquisition with vault/loadout destination. Loadout swap handles vault↔loadout moves.",
		},
		{
			ID:          "environment-entities",
			Name:        "Environment Entities",
			Category:    CategoryOptionalRules,
			Status:      MechanicPending,
			Requirement: Optional,
			Notes:       "PRD defines environment stat blocks with impulses/difficulty/features. Currently narrative-only in scenarios.",
		},
		{
			ID:          "spotlight-tracker-tokens",
			Name:        "Spotlight Tracker Tokens",
			Category:    CategoryOptionalRules,
			Status:      MechanicPending,
			Requirement: Optional,
			Notes:       "Optional rule from PRD.",
		},
		{
			ID:          "fate-rolls",
			Name:        "Fate Rolls",
			Category:    CategoryOptionalRules,
			Status:      MechanicPending,
			Requirement: Optional,
			Notes:       "Optional rule from PRD.",
		},
		{
			ID:           "countdown-variants",
			Name:         "Countdown Variants",
			Category:     CategoryOptionalRules,
			Status:       MechanicImplemented,
			Requirement:  Optional,
			Commands:     []command.Type{commandTypeCountdownCreate},
			Events:       []event.Type{EventTypeCountdownCreated},
			ScenarioTags: []string{"countdown_variants"},
			Notes:        "Dynamic (trigger_event_type) and Linked (linked_countdown_id) variant types on countdown creation.",
		},
		{
			ID:           "gold-currency",
			Name:         "Gold/Currency Tracking",
			Category:     CategoryOptionalRules,
			Status:       MechanicImplemented,
			Requirement:  Optional,
			Commands:     []command.Type{commandTypeGoldUpdate},
			Events:       []event.Type{EventTypeGoldUpdated},
			ScenarioTags: []string{"gold_currency_tracking", "environment_prancing_pony_sing", "environment_bree_market_tip_the_scales"},
			Notes:        "Handfuls/bags/chests denomination tracking with profile persistence.",
		},
		{
			ID:          "companion-mechanics",
			Name:        "Companion Mechanics",
			Category:    CategoryOptionalRules,
			Status:      MechanicPending,
			Requirement: Optional,
			Notes:       "Ranger Beastbound subclass companion sheet. Class-specific content extension.",
		},
		{
			ID:          "beastform-mechanics",
			Name:        "Beastform Mechanics",
			Category:    CategoryOptionalRules,
			Status:      MechanicPending,
			Requirement: Optional,
			Notes:       "Druid transformation rules with tiers. Class-specific content extension.",
		},
		{
			ID:           "consumables",
			Name:         "Consumables System",
			Category:     CategoryOptionalRules,
			Status:       MechanicImplemented,
			Requirement:  Optional,
			Commands:     []command.Type{commandTypeConsumableUse, commandTypeConsumableAcquire},
			Events:       []event.Type{EventTypeConsumableUsed, EventTypeConsumableAcquired},
			ScenarioTags: []string{"consumable_lifecycle"},
			Notes:        "Use/acquire consumables with stack max 5 and quantity tracking.",
		},
		{
			ID:          "underwater-combat",
			Name:        "Underwater Combat",
			Category:    CategoryOptionalRules,
			Status:      MechanicNotApplicable,
			Requirement: Optional,
			Notes:       "Composes existing mechanics: disadvantage + countdown. No new server mechanic needed.",
		},
		{
			ID:          "pc-vs-pc",
			Name:        "PC vs PC Conflict",
			Category:    CategoryOptionalRules,
			Status:      MechanicPending,
			Requirement: Optional,
			Notes:       "Optional rule from PRD.",
		},
	}
}

// DeriveImplementationStage computes the implementation stage from the manifest.
// Only Required mechanics affect the stage: all implemented → COMPLETE,
// some implemented → PARTIAL, none implemented → PLANNED.
func DeriveImplementationStage() bridge.ImplementationStage {
	manifest := MechanicsManifest()

	var requiredTotal, requiredImplemented int
	for _, m := range manifest {
		if m.Requirement != Required {
			continue
		}
		requiredTotal++
		if m.Status == MechanicImplemented {
			requiredImplemented++
		}
	}

	switch {
	case requiredTotal == 0:
		return bridge.ImplementationStagePlanned
	case requiredImplemented == requiredTotal:
		return bridge.ImplementationStageComplete
	case requiredImplemented > 0:
		return bridge.ImplementationStagePartial
	default:
		return bridge.ImplementationStagePlanned
	}
}

// deriveImplementationNotes returns a human-readable coverage summary
// describing the current mechanic implementation state.
func deriveImplementationNotes() string {
	manifest := MechanicsManifest()

	var total, implemented, pending int
	for _, m := range manifest {
		if m.Status == MechanicNotApplicable {
			continue
		}
		total++
		switch m.Status {
		case MechanicImplemented:
			implemented++
		case MechanicPending:
			pending++
		}
	}

	return fmt.Sprintf("%d/%d mechanics implemented, %d pending", implemented, total, pending)
}
