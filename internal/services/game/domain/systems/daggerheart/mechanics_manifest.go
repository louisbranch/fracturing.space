package daggerheart

import (
	"fmt"

	daggerheartdecider "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/decider"

	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
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
			Commands:    []command.Type{daggerheartdecider.CommandTypeCharacterStatePatch},
			Events:      []event.Type{daggerheartpayload.EventTypeCharacterStatePatched},
			Notes:       "HP, Hope, Stress, Armor, life state mutations.",
		},
		{
			ID:           "hope-economy",
			Name:         "Hope Economy",
			Category:     CategoryCharacterModel,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{daggerheartdecider.CommandTypeHopeSpend},
			Events:       []event.Type{daggerheartpayload.EventTypeCharacterStatePatched},
			ScenarioTags: []string{"action_roll_failure_with_hope", "spellcast_hope_cost"},
			Notes:        "Hope spend command; gains via CharacterStatePatched.",
		},
		{
			ID:           "stress-economy",
			Name:         "Stress Economy",
			Category:     CategoryCharacterModel,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{daggerheartdecider.CommandTypeStressSpend},
			Events:       []event.Type{daggerheartpayload.EventTypeCharacterStatePatched},
			ScenarioTags: []string{"companion_experience_stress_clear"},
			Notes:        "Stress spend command; gains via CharacterStatePatched.",
		},
		{
			ID:           "conditions",
			Name:         "Conditions (Hidden/Restrained/Vulnerable)",
			Category:     CategoryCharacterModel,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{daggerheartdecider.CommandTypeConditionChange},
			Events:       []event.Type{daggerheartpayload.EventTypeConditionChanged},
			ScenarioTags: []string{"condition_lifecycle", "condition_stacking_guard", "hidden_condition_scouting"},
			Notes:        "Core conditions with add/remove lifecycle.",
		},
		{
			ID:           "stat-modifiers",
			Name:         "Stat Modifiers (Evasion/Thresholds/Proficiency/Armor)",
			Category:     CategoryCharacterModel,
			Status:       MechanicImplemented,
			Requirement:  Optional,
			Commands:     []command.Type{daggerheartdecider.CommandTypeStatModifierChange},
			Events:       []event.Type{daggerheartpayload.EventTypeStatModifierChanged},
			ScenarioTags: []string{"stat_modifier_lifecycle"},
			Notes:        "Runtime stat modifier application with duration-based clearing via ClearTriggers.",
		},
		{
			ID:           "loadout-swap",
			Name:         "Loadout Swap",
			Category:     CategoryCharacterModel,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{daggerheartdecider.CommandTypeLoadoutSwap},
			Events:       []event.Type{daggerheartpayload.EventTypeLoadoutSwapped},
			ScenarioTags: []string{"loadout_swap"},
			Notes:        "Swap domain cards between vault and active loadout.",
		},
		{
			ID:           "class-features",
			Name:         "Class Feature Activations",
			Category:     CategoryCharacterModel,
			Status:       MechanicImplemented,
			Requirement:  Optional,
			Commands:     []command.Type{daggerheartdecider.CommandTypeClassFeatureApply},
			Events:       []event.Type{daggerheartpayload.EventTypeCharacterStatePatched},
			ScenarioTags: []string{"class_feature_core"},
			Notes:        "Typed class feature activations resolve through class_feature.apply into durable character state patches.",
		},
		{
			ID:           "subclass-features",
			Name:         "Subclass Feature Activations",
			Category:     CategoryCharacterModel,
			Status:       MechanicImplemented,
			Requirement:  Optional,
			Commands:     []command.Type{daggerheartdecider.CommandTypeSubclassFeatureApply},
			Events:       []event.Type{daggerheartpayload.EventTypeCharacterStatePatched, daggerheartpayload.EventTypeConditionChanged, daggerheartpayload.EventTypeAdversaryConditionChanged},
			ScenarioTags: []string{"class_feature_core"},
			Notes:        "Activated subclass abilities resolve through subclass_feature.apply into state and condition consequence events.",
		},

		// ── Combat & Damage ─────────────────────────────────────────────
		{
			ID:           "single-target-damage",
			Name:         "Single-Target Damage",
			Category:     CategoryCombatDamage,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{daggerheartdecider.CommandTypeDamageApply},
			Events:       []event.Type{daggerheartpayload.EventTypeDamageApplied},
			ScenarioTags: []string{"damage_thresholds_example", "critical_damage", "critical_damage_maximum"},
			Notes:        "Damage application with severity evaluation.",
		},
		{
			ID:           "multi-target-damage",
			Name:         "Multi-Target Damage",
			Category:     CategoryCombatDamage,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{daggerheartdecider.CommandTypeMultiTargetDamageApply},
			ScenarioTags: []string{"multi_target_attack", "sweeping_attack_all_targets"},
			Notes:        "Damage applied to multiple characters in one command.",
		},
		{
			ID:           "adversary-damage",
			Name:         "Adversary Damage",
			Category:     CategoryCombatDamage,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{daggerheartdecider.CommandTypeAdversaryDamageApply},
			Events:       []event.Type{daggerheartpayload.EventTypeAdversaryDamageApplied},
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
			Commands:     []command.Type{daggerheartdecider.CommandTypeCharacterTemporaryArmorApply},
			Events:       []event.Type{daggerheartpayload.EventTypeCharacterTemporaryArmorApplied},
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
			Commands:     []command.Type{daggerheartdecider.CommandTypeRestTake},
			Events:       []event.Type{daggerheartpayload.EventTypeRestTaken},
			ScenarioTags: []string{"rest_and_downtime"},
			Notes:        "Short and long rests with 3-short-before-long cadence enforced.",
		},
		{
			ID:           "downtime-moves",
			Name:         "Downtime Moves",
			Category:     CategoryRestDowntime,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{daggerheartdecider.CommandTypeRestTake},
			Events:       []event.Type{daggerheartpayload.EventTypeDowntimeMoveApplied},
			ScenarioTags: []string{"rest_and_downtime"},
			Notes:        "Atomic rest workflow emits downtime move events per participant selection.",
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
			Commands:     []command.Type{daggerheartdecider.CommandTypeGMFearSet, daggerheartdecider.CommandTypeGMMoveApply},
			Events:       []event.Type{daggerheartpayload.EventTypeGMFearChanged, daggerheartpayload.EventTypeGMMoveApplied},
			ScenarioTags: []string{"fear_floor", "fear_initialization", "gm_move_examples", "gm_fear_adversary_feature", "gm_fear_environment_feature", "gm_fear_adversary_experience"},
			Notes:        "Fear set/bootstrap with 0-12 range enforcement plus typed GM move spend audit events for direct moves, adversary features, environment features, and adversary experiences.",
		},
		{
			ID:           "scene-countdowns",
			Name:         "Scene Countdowns (Create/Advance/Resolve/Delete)",
			Category:     CategoryGMMechanics,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{daggerheartdecider.CommandTypeSceneCountdownCreate, daggerheartdecider.CommandTypeSceneCountdownAdvance, daggerheartdecider.CommandTypeSceneCountdownTriggerResolve, daggerheartdecider.CommandTypeSceneCountdownDelete},
			Events:       []event.Type{daggerheartpayload.EventTypeSceneCountdownCreated, daggerheartpayload.EventTypeSceneCountdownAdvanced, daggerheartpayload.EventTypeSceneCountdownTriggerResolved, daggerheartpayload.EventTypeSceneCountdownDeleted},
			ScenarioTags: []string{"countdown_lifecycle", "progress_countdown_climb"},
			Notes:        "Scene-owned countdown system for spotlight/combat board pressure, with explicit trigger-pending resolution and looping support.",
		},
		{
			ID:           "campaign-countdowns",
			Name:         "Campaign Countdowns (Create/Advance/Resolve/Delete)",
			Category:     CategoryGMMechanics,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{daggerheartdecider.CommandTypeCampaignCountdownCreate, daggerheartdecider.CommandTypeCampaignCountdownAdvance, daggerheartdecider.CommandTypeCampaignCountdownTriggerResolve, daggerheartdecider.CommandTypeCampaignCountdownDelete},
			Events:       []event.Type{daggerheartpayload.EventTypeCampaignCountdownCreated, daggerheartpayload.EventTypeCampaignCountdownAdvanced, daggerheartpayload.EventTypeCampaignCountdownTriggerResolved, daggerheartpayload.EventTypeCampaignCountdownDeleted},
			ScenarioTags: []string{"long_term_countdown"},
			Notes:        "Campaign-owned countdown system for persistent rest and project progress, with explicit trigger resolution after reaching zero.",
		},
		{
			ID:          "adversary-crud",
			Name:        "Adversary CRUD",
			Category:    CategoryGMMechanics,
			Status:      MechanicImplemented,
			Requirement: Required,
			Commands:    []command.Type{daggerheartdecider.CommandTypeAdversaryCreate, daggerheartdecider.CommandTypeAdversaryUpdate, daggerheartdecider.CommandTypeAdversaryDelete},
			Events:      []event.Type{daggerheartpayload.EventTypeAdversaryCreated, daggerheartpayload.EventTypeAdversaryUpdated, daggerheartpayload.EventTypeAdversaryDeleted},
			Notes:       "Create, update, delete adversaries in session.",
		},
		{
			ID:           "adversary-features",
			Name:         "Adversary Features",
			Category:     CategoryGMMechanics,
			Status:       MechanicImplemented,
			Requirement:  Optional,
			Commands:     []command.Type{daggerheartdecider.CommandTypeAdversaryFeatureApply, daggerheartdecider.CommandTypeGMMoveApply},
			Events:       []event.Type{daggerheartpayload.EventTypeAdversaryUpdated, daggerheartpayload.EventTypeAdversaryDamageApplied, daggerheartpayload.EventTypeAdversaryConditionChanged, daggerheartpayload.EventTypeDamageApplied, daggerheartpayload.EventTypeCharacterStatePatched, daggerheartpayload.EventTypeGMMoveApplied},
			ScenarioTags: []string{"gm_fear_adversary_feature", "gm_fear_adversary_experience", "terrifying_hope_loss", "skulk_cloaked_backstab", "ranged_warding_sphere"},
			Notes:        "Typed adversary feature staging and execution for repeatable runtime families, split between GM Fear spends and direct adversary feature application.",
		},
		{
			ID:          "adversary-conditions",
			Name:        "Adversary Conditions",
			Category:    CategoryGMMechanics,
			Status:      MechanicImplemented,
			Requirement: Required,
			Commands:    []command.Type{daggerheartdecider.CommandTypeAdversaryConditionChange},
			Events:      []event.Type{daggerheartpayload.EventTypeAdversaryConditionChanged},
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
			Commands:     []command.Type{daggerheartdecider.CommandTypeLevelUpApply},
			Events:       []event.Type{daggerheartpayload.EventTypeLevelUpApplied},
			ScenarioTags: []string{"level_up_basic", "level_up_tier_entry"},
			Notes:        "Level-up with tier achievements, advancement budget, trait marking, and damage threshold progression.",
		},
		{
			ID:           "multiclassing",
			Name:         "Multiclassing",
			Category:     CategoryProgression,
			Status:       MechanicImplemented,
			Requirement:  Required,
			Commands:     []command.Type{daggerheartdecider.CommandTypeLevelUpApply},
			Events:       []event.Type{daggerheartpayload.EventTypeLevelUpApplied},
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
			Commands:     []command.Type{daggerheartdecider.CommandTypeRestTake},
			Events:       []event.Type{daggerheartpayload.EventTypeDowntimeMoveApplied},
			ScenarioTags: []string{"downtime_move_limit"},
			Notes:        "Per-participant limit is enforced inside the atomic rest workflow; scenario coverage now uses participant-scoped downtime selections on rest.take.",
		},
		{
			ID:           "equipment-management",
			Name:         "Equipment Management",
			Category:     CategoryOptionalRules,
			Status:       MechanicImplemented,
			Requirement:  Optional,
			Commands:     []command.Type{daggerheartdecider.CommandTypeEquipmentSwap},
			Events:       []event.Type{daggerheartpayload.EventTypeEquipmentSwapped},
			ScenarioTags: []string{"equipment_swap"},
			Notes:        "In-session equip/unequip with slot tracking (active/inventory/none) and item type validation.",
		},
		{
			ID:           "domain-card-vault",
			Name:         "Domain Card Vault Management",
			Category:     CategoryOptionalRules,
			Status:       MechanicImplemented,
			Requirement:  Optional,
			Commands:     []command.Type{daggerheartdecider.CommandTypeDomainCardAcquire},
			Events:       []event.Type{daggerheartpayload.EventTypeDomainCardAcquired},
			ScenarioTags: []string{"domain_card_acquire"},
			Notes:        "Card acquisition with vault/loadout destination. Loadout swap handles vault↔loadout moves.",
		},
		{
			ID:           "environment-entities",
			Name:         "Environment Entities",
			Category:     CategoryOptionalRules,
			Status:       MechanicImplemented,
			Requirement:  Optional,
			Commands:     []command.Type{daggerheartdecider.CommandTypeEnvironmentEntityCreate, daggerheartdecider.CommandTypeEnvironmentEntityUpdate, daggerheartdecider.CommandTypeEnvironmentEntityDelete, daggerheartdecider.CommandTypeGMMoveApply},
			Events:       []event.Type{daggerheartpayload.EventTypeEnvironmentEntityCreated, daggerheartpayload.EventTypeEnvironmentEntityUpdated, daggerheartpayload.EventTypeEnvironmentEntityDeleted, daggerheartpayload.EventTypeGMMoveApplied},
			ScenarioTags: []string{"gm_fear_environment_feature"},
			Notes:        "Environment stat blocks are instantiated as runtime entities for scene/session placement and GM Fear environment-feature spends. Feature execution remains journaled rather than typed.",
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
			Commands:     []command.Type{daggerheartdecider.CommandTypeSceneCountdownCreate, daggerheartdecider.CommandTypeCampaignCountdownCreate},
			Events:       []event.Type{daggerheartpayload.EventTypeSceneCountdownCreated, daggerheartpayload.EventTypeCampaignCountdownCreated},
			ScenarioTags: []string{"countdown_variants"},
			Notes:        "Dynamic (trigger_event_type) and linked (linked_countdown_id) variants on both countdown ownership types.",
		},
		{
			ID:           "gold-currency",
			Name:         "Gold/Currency Tracking",
			Category:     CategoryOptionalRules,
			Status:       MechanicImplemented,
			Requirement:  Optional,
			Commands:     []command.Type{daggerheartdecider.CommandTypeGoldUpdate},
			Events:       []event.Type{daggerheartpayload.EventTypeGoldUpdated},
			ScenarioTags: []string{"gold_currency_tracking", "environment_prancing_pony_sing", "environment_bree_market_tip_the_scales"},
			Notes:        "Handfuls/bags/chests denomination tracking with profile persistence.",
		},
		{
			ID:           "companion-mechanics",
			Name:         "Companion Mechanics",
			Category:     CategoryOptionalRules,
			Status:       MechanicPending,
			Requirement:  Optional,
			Commands:     []command.Type{daggerheartdecider.CommandTypeCompanionExperienceBegin, daggerheartdecider.CommandTypeCompanionReturn},
			Events:       []event.Type{daggerheartpayload.EventTypeCompanionExperienceBegun, daggerheartpayload.EventTypeCompanionReturned},
			ScenarioTags: []string{"companion_experience_stress_clear"},
			Notes:        "Ranger Beastbound companion sheet plus runtime dispatch/return foundation. Broader companion combat mechanics remain pending.",
		},
		{
			ID:          "beastform-mechanics",
			Name:        "Beastform Mechanics",
			Category:    CategoryOptionalRules,
			Status:      MechanicImplemented,
			Requirement: Optional,
			Commands:    []command.Type{daggerheartdecider.CommandTypeBeastformTransform, daggerheartdecider.CommandTypeBeastformDrop},
			Events:      []event.Type{daggerheartpayload.EventTypeBeastformTransformed, daggerheartpayload.EventTypeBeastformDropped},
			Notes:       "Druid beastform transform/drop with resolved attack state, evasion bonus, and damage-triggered auto-drop.",
		},
		{
			ID:           "consumables",
			Name:         "Consumables System",
			Category:     CategoryOptionalRules,
			Status:       MechanicImplemented,
			Requirement:  Optional,
			Commands:     []command.Type{daggerheartdecider.CommandTypeConsumableUse, daggerheartdecider.CommandTypeConsumableAcquire},
			Events:       []event.Type{daggerheartpayload.EventTypeConsumableUsed, daggerheartpayload.EventTypeConsumableAcquired},
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
