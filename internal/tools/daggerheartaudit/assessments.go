package main

import (
	"fmt"
	"strings"
)

// curatedAssessment captures milestone-level audit conclusions that should be
// reproducible in generated outputs instead of being hand-edited into JSON.
type curatedAssessment struct {
	ReviewState   string
	NameStrategy  string
	SemanticMatch string
	FinalStatus   string
	GapClass      string
	EvidenceCode  []string
	EvidenceTests []string
	EvidenceDocs  []string
	Notes         []string
	FollowUpEpic  string
}

var curatedAssessments = map[string]curatedAssessment{
	"character-creation": {
		ReviewState:   "reviewed",
		NameStrategy:  "canonical",
		SemanticMatch: "matched",
		FinalStatus:   "covered",
		EvidenceCode: []string{
			"internal/services/game/domain/systems/daggerheart/creation_workflow.go",
			"internal/services/game/api/grpc/systems/daggerheart/creationworkflow/provider.go",
			"internal/services/web/modules/campaigns/workflow/daggerheart/form.go",
			"internal/services/mcp/domain/character_handlers.go",
			"api/proto/systems/daggerheart/v1/state.proto",
		},
		EvidenceTests: []string{
			"internal/services/web/modules/campaigns/workflow/daggerheart/form_test.go",
		},
		EvidenceDocs: []string{
			"docs/reference/daggerheart-creation-workflow.md",
			"docs/product/daggerheart-PRD.md",
		},
		Notes: []string{
			"The implementation intentionally keeps free-form steps at the end of the workflow. The ordering difference is accepted so long as it does not change rule enforcement or persisted semantics.",
			"Character creation now persists structured heritage selections, subclass creation requirements, and companion sheets where required.",
		},
	},
	"core-materials": {
		ReviewState:   "reviewed",
		NameStrategy:  "canonical",
		SemanticMatch: "matched",
		FinalStatus:   "covered",
		EvidenceCode: []string{
			"internal/services/game/domain/systems/daggerheart/contentstore/contracts.go",
			"internal/services/game/api/grpc/systems/daggerheart/contenttransport/catalog_orchestrator.go",
			"internal/services/game/domain/systems/daggerheart/mechanics_manifest.go",
			"api/proto/systems/daggerheart/v1/state.proto",
		},
		EvidenceTests: []string{
			"internal/services/game/api/grpc/systems/daggerheart/contenttransport/service_support_test.go",
			"internal/services/game/api/grpc/systems/daggerheart/actions_swap_loadout_test.go",
		},
		EvidenceDocs: []string{
			"docs/product/daggerheart-PRD.md",
		},
		Notes: []string{
			"Catalog coverage exists for classes, subclasses, domains, beastforms, and companion experiences.",
			"The previously open class-feature, subclass-feature, beastform, and companion-runtime slices are now implemented and reflected through runtime state, transport, and mechanics-manifest coverage.",
			"Remaining open content-driven mechanics are tracked by their narrower epics rather than this umbrella reference row.",
		},
	},
	"core-mechanics": {
		ReviewState:   "reviewed",
		NameStrategy:  "canonical",
		SemanticMatch: "matched",
		FinalStatus:   "covered",
		EvidenceCode: []string{
			"internal/services/game/domain/systems/daggerheart/internal/mechanics/rest.go",
			"internal/services/game/domain/systems/daggerheart/internal/mechanics/downtime.go",
			"internal/services/game/domain/systems/daggerheart/compat_conditions.go",
			"internal/services/game/domain/systems/daggerheart/internal/mechanics/death.go",
		},
		EvidenceTests: []string{
			"internal/services/game/domain/systems/daggerheart/internal/mechanics/death_rest_downtime_branches_test.go",
			"internal/services/game/domain/systems/daggerheart/compat_conditions_branches_test.go",
		},
		EvidenceDocs: []string{
			"docs/product/daggerheart-PRD.md",
		},
		Notes: []string{
			"Core death, downtime, fear, and condition rules are now covered by their dedicated audit rows and aligned runtime paths.",
			"The umbrella core-mechanics row no longer carries independent runtime drift after the condition lifecycle cutover.",
		},
	},
	"glossary-conditions": {
		ReviewState:   "reviewed",
		NameStrategy:  "canonical",
		SemanticMatch: "matched",
		FinalStatus:   "covered",
		EvidenceCode: []string{
			"internal/services/game/domain/systems/daggerheart/compat_conditions.go",
			"internal/services/game/api/grpc/systems/daggerheart/workfloweffects/handler.go",
			"internal/services/game/api/grpc/systems/daggerheart/conditiontransport/handler.go",
			"internal/services/game/api/grpc/systems/daggerheart/charactermutationtransport/subclass.go",
		},
		EvidenceTests: []string{
			"internal/services/game/domain/systems/daggerheart/compat_conditions_branches_test.go",
			"internal/services/game/domain/systems/daggerheart/condition_logic_branches_test.go",
			"internal/services/game/api/grpc/systems/daggerheart/workfloweffects/handler_test.go",
		},
		EvidenceDocs: []string{
			"docs/product/daggerheart-PRD.md",
		},
		Notes: []string{
			"Structured condition entries now carry class, source, stable IDs, and lifecycle triggers across transport, projection, and replay paths.",
			"Stress-threshold vulnerable and Nightwalker cloaked both flow through the shared condition model instead of subclass-specific duplicate state.",
		},
	},
	"glossary-death": {
		ReviewState:   "reviewed",
		NameStrategy:  "canonical",
		SemanticMatch: "matched",
		FinalStatus:   "covered",
		EvidenceCode: []string{
			"internal/services/game/domain/systems/daggerheart/internal/mechanics/death.go",
			"internal/services/game/api/grpc/systems/daggerheart/recoverytransport/handler.go",
		},
		EvidenceTests: []string{
			"internal/services/game/domain/systems/daggerheart/internal/mechanics/death_rest_downtime_branches_test.go",
			"internal/services/game/api/grpc/systems/daggerheart/actions_death_blaze_test.go",
			"internal/services/game/api/grpc/systems/daggerheart/recoverytransport/handler_test.go",
			"internal/test/game/scenarios/systems/daggerheart/death_move_last_hope_slot.lua",
		},
		EvidenceDocs: []string{
			"docs/product/daggerheart-PRD.md",
		},
		Notes: []string{
			"Blaze of Glory, Avoid Death, Risk It All, unconscious recovery, and scar-driven hope-slot reduction are implemented.",
			"Avoid Death now becomes terminal when the gained scar crosses out the final Hope slot, matching the reference journey-end rule.",
		},
	},
	"glossary-downtime": {
		ReviewState:   "reviewed",
		NameStrategy:  "canonical",
		SemanticMatch: "matched",
		FinalStatus:   "covered",
		EvidenceCode: []string{
			"internal/services/game/domain/systems/daggerheart/internal/mechanics/rest.go",
			"internal/services/game/domain/systems/daggerheart/internal/mechanics/downtime.go",
			"internal/services/game/domain/systems/daggerheart/rest_package.go",
			"internal/services/game/api/grpc/systems/daggerheart/recoverytransport/handler.go",
			"api/proto/systems/daggerheart/v1/state.proto",
		},
		EvidenceTests: []string{
			"internal/services/game/domain/systems/daggerheart/internal/mechanics/death_rest_downtime_branches_test.go",
			"internal/services/game/api/grpc/systems/daggerheart/actions_rest_test.go",
			"internal/services/game/api/grpc/systems/daggerheart/recoverytransport/handler_test.go",
			"internal/test/game/scenarios/systems/daggerheart/rest_and_downtime.lua",
			"internal/test/game/scenarios/systems/daggerheart/rest_long_project.lua",
		},
		EvidenceDocs: []string{
			"docs/reference/daggerheart-event-timeline-contract.md",
		},
		Notes: []string{
			"Rests now carry participant-scoped downtime selections atomically, including short-rest recovery moves, long-rest full recovery moves, grouped prepare, and project advancement.",
			"Interrupted short rests remain durable no-op rest records, while interrupted long rests downgrade to short-rest move legality and refresh semantics.",
		},
	},
	"glossary-fear": {
		ReviewState:   "reviewed",
		NameStrategy:  "canonical",
		SemanticMatch: "matched",
		FinalStatus:   "covered",
		EvidenceCode: []string{
			"internal/services/game/domain/readiness/session_start_workflow.go",
			"internal/services/game/domain/systems/daggerheart/decider_state_conditions.go",
			"internal/services/game/domain/systems/daggerheart/module_validation_state.go",
			"internal/services/game/api/grpc/systems/daggerheart/gmmovetransport/handler.go",
			"internal/services/game/api/grpc/systems/daggerheart/outcometransport/handler.go",
		},
		EvidenceTests: []string{
			"internal/services/game/domain/readiness/session_start_workflow_test.go",
			"internal/services/game/domain/systems/daggerheart/gm_move_test.go",
			"internal/services/game/api/grpc/systems/daggerheart/gmmovetransport/handler_test.go",
			"internal/services/game/api/grpc/systems/daggerheart/actions_gm_move_test.go",
			"internal/test/game/scenarios/systems/daggerheart/fear_initialization.lua",
			"internal/test/game/scenarios/systems/daggerheart/gm_fear_adversary_feature.lua",
			"internal/test/game/scenarios/systems/daggerheart/gm_fear_environment_feature.lua",
			"internal/test/game/scenarios/systems/daggerheart/gm_fear_adversary_experience.lua",
		},
		EvidenceDocs: []string{
			"docs/reference/daggerheart-event-timeline-contract.md",
			"docs/product/daggerheart-PRD.md",
		},
		Notes: []string{
			"First-session fear seeding, gains, cap enforcement, rest gains, and typed GM move spends are implemented.",
			"Fear spends are replay-safe across direct moves, adversary fear features, environment fear features, and adversary experience spends, with catalog-backed validation where the content model supports it.",
		},
	},
	"glossary-hope": {
		ReviewState:   "reviewed",
		NameStrategy:  "canonical",
		SemanticMatch: "matched",
		FinalStatus:   "covered",
		EvidenceCode: []string{
			"internal/services/game/domain/systems/daggerheart/state_factory.go",
			"internal/services/game/api/grpc/systems/daggerheart/outcometransport/handler.go",
			"internal/services/game/api/grpc/systems/daggerheart/sessionrolltransport/helpers.go",
		},
		EvidenceTests: []string{
			"internal/services/game/domain/systems/daggerheart/state_factory_test.go",
			"internal/services/game/api/grpc/systems/daggerheart/actions_session_rolls_test.go",
			"internal/services/game/api/grpc/systems/daggerheart/actions_apply_roll_outcome_domain_test.go",
		},
		EvidenceDocs: []string{
			"docs/product/daggerheart-PRD.md",
		},
		Notes: []string{
			"The extracted corpus text is truncated, but repo behavior matches the expected hope defaults, caps, spend flows, prepare gains, and roll-outcome gains described elsewhere in the reference set.",
		},
	},
	"glossary-session-zero": {
		ReviewState:   "reviewed",
		NameStrategy:  "not_applicable",
		SemanticMatch: "not_applicable",
		FinalStatus:   "not_applicable",
		Notes: []string{
			"This glossary row is a pointer to campaign-frame appendix material, not a durable runtime invariant for the Daggerheart system implementation.",
		},
	},
	"glossary-stress": {
		ReviewState:   "reviewed",
		NameStrategy:  "canonical",
		SemanticMatch: "matched",
		FinalStatus:   "covered",
		EvidenceCode: []string{
			"internal/services/game/domain/systems/daggerheart/internal/mechanics/character_state.go",
			"internal/services/game/api/grpc/systems/daggerheart/workfloweffects/handler.go",
			"internal/services/game/api/grpc/systems/daggerheart/actions_swap_loadout_test.go",
		},
		EvidenceTests: []string{
			"internal/services/game/domain/systems/daggerheart/internal/mechanics/character_state_branches_test.go",
			"internal/services/game/api/grpc/systems/daggerheart/workfloweffects/handler_test.go",
			"internal/services/game/api/grpc/systems/daggerheart/actions_swap_loadout_test.go",
		},
		EvidenceDocs: []string{
			"docs/product/daggerheart-PRD.md",
		},
		Notes: []string{
			"Stress overflow into HP, the vulnerable threshold transition, and stress spending for modeled mechanics are implemented.",
		},
	},
	"introduction": {
		ReviewState:   "reviewed",
		NameStrategy:  "not_applicable",
		SemanticMatch: "not_applicable",
		FinalStatus:   "not_applicable",
		Notes: []string{
			"The introduction is editorial framing and table-principles guidance, not an executable contract the repo should fully enforce.",
		},
	},
	"playbook-gm-fear-and-moves": {
		ReviewState:   "reviewed",
		NameStrategy:  "canonical",
		SemanticMatch: "matched",
		FinalStatus:   "covered",
		EvidenceCode: []string{
			"internal/services/game/api/grpc/systems/daggerheart/gmmovetransport/handler.go",
			"internal/services/game/api/grpc/systems/daggerheart/outcometransport/handler.go",
			"internal/services/game/domain/systems/daggerheart/decider_state_conditions.go",
			"internal/services/game/domain/systems/daggerheart/mechanics_manifest.go",
		},
		EvidenceTests: []string{
			"internal/services/game/api/grpc/systems/daggerheart/gmmovetransport/handler_test.go",
			"internal/services/game/api/grpc/systems/daggerheart/actions_gm_move_test.go",
			"internal/test/game/scenarios/systems/daggerheart/gm_move_examples.lua",
			"internal/test/game/scenarios/systems/daggerheart/gm_fear_adversary_feature.lua",
			"internal/test/game/scenarios/systems/daggerheart/gm_fear_environment_feature.lua",
			"internal/test/game/scenarios/systems/daggerheart/gm_fear_adversary_experience.lua",
		},
		EvidenceDocs: []string{
			"docs/reference/daggerheart-event-timeline-contract.md",
		},
		Notes: []string{
			"The repo supports typed GM move semantics, fear accounting, the session spotlight consequence path, and explicit spend targets for adversary features, environment features, and adversary experiences.",
			"Content-driven feature text is still adjudicated by the GM rather than auto-executed, which is acceptable because the playbook requires the move taxonomy and spend semantics, not deterministic natural-language effect execution.",
		},
	},
	"playbook-rests-and-downtime": {
		ReviewState:   "reviewed",
		NameStrategy:  "canonical",
		SemanticMatch: "matched",
		FinalStatus:   "covered",
		EvidenceCode: []string{
			"internal/services/game/domain/systems/daggerheart/internal/mechanics/rest.go",
			"internal/services/game/domain/systems/daggerheart/internal/mechanics/downtime.go",
			"internal/services/game/domain/systems/daggerheart/rest_package.go",
			"internal/services/game/api/grpc/systems/daggerheart/recoverytransport/handler.go",
			"api/proto/systems/daggerheart/v1/state.proto",
		},
		EvidenceTests: []string{
			"internal/services/game/domain/systems/daggerheart/internal/mechanics/death_rest_downtime_branches_test.go",
			"internal/services/game/api/grpc/systems/daggerheart/recoverytransport/handler_test.go",
			"internal/services/game/api/grpc/systems/daggerheart/actions_rest_test.go",
			"internal/test/game/scenarios/systems/daggerheart/rest_and_downtime.lua",
			"internal/test/game/scenarios/systems/daggerheart/rest_long_project.lua",
		},
		EvidenceDocs: []string{
			"docs/reference/daggerheart-event-timeline-contract.md",
		},
		Notes: []string{
			"The transport and domain now expose the full rest/downtime menu through one atomic rest request rather than a split rest-plus-move write path.",
			"Project advancement is no longer state-neutral; long rests can advance countdowns directly and work_on_project selections can auto-advance or carry GM-set deltas.",
		},
	},
	"playbook-session-flow": {
		ReviewState:   "reviewed",
		NameStrategy:  "canonical",
		SemanticMatch: "matched",
		FinalStatus:   "covered",
		EvidenceCode: []string{
			"internal/services/game/api/grpc/systems/daggerheart/sessionflowtransport/handler.go",
			"internal/services/game/api/grpc/systems/daggerheart/outcometransport/handler.go",
			"internal/services/game/api/grpc/systems/daggerheart/workflow_session_service.go",
		},
		EvidenceTests: []string{
			"internal/services/game/api/grpc/systems/daggerheart/actions_session_reaction_flow_test.go",
			"internal/services/game/api/grpc/systems/daggerheart/actions_apply_roll_outcome_domain_effects_test.go",
		},
		EvidenceDocs: []string{
			"docs/reference/daggerheart-event-timeline-contract.md",
		},
		Notes: []string{
			"Reaction, group-action, tag-team, outcome resolution, and spotlight-transfer behavior are modeled.",
			"Advice such as auto-succeeding uninteresting actions remains GM guidance rather than a runtime-enforced rule, which is acceptable for this playbook row.",
		},
	},
	"playbook-witherwild-session-zero": {
		ReviewState:   "reviewed",
		NameStrategy:  "not_applicable",
		SemanticMatch: "not_applicable",
		FinalStatus:   "not_applicable",
		Notes: []string{
			"This is campaign-frame prompt material and does not define a repo-owned Daggerheart system invariant.",
		},
	},
	"running-an-adventure": {
		ReviewState:   "reviewed",
		NameStrategy:  "canonical",
		SemanticMatch: "matched",
		FinalStatus:   "covered",
		EvidenceCode: []string{
			"internal/services/game/api/grpc/systems/daggerheart/outcometransport/handler.go",
			"internal/services/game/api/grpc/systems/daggerheart/gmmovetransport/handler.go",
			"internal/services/game/api/grpc/systems/daggerheart/countdowntransport/handler.go",
			"internal/services/game/domain/systems/daggerheart/mechanics_manifest.go",
		},
		EvidenceTests: []string{
			"internal/services/game/api/grpc/systems/daggerheart/actions_gm_move_test.go",
			"internal/services/game/api/grpc/systems/daggerheart/actions_apply_roll_outcome_domain_effects_test.go",
		},
		EvidenceDocs: []string{
			"docs/reference/daggerheart-event-timeline-contract.md",
			"docs/reference/daggerheart-ai-gm-guidance.md",
			"docs/product/daggerheart-PRD.md",
		},
		Notes: []string{
			"Scene consequence handling, fear accounting, countdown primitives, spotlight transfer, and typed GM move semantics are implemented where the system owns durable mechanics.",
			"The remaining reference material in this section is GM-facing table guidance rather than runtime-owned state. That guidance is covered by the AI GM summary doc instead of being tracked as a mechanics gap.",
		},
	},
}

func baselineAssessmentForRow(row auditMatrixRow, advClasses map[string]adversaryEntryClass, abilityMatches map[string]abilityDomainCardMatch, itemMatches map[string]itemEffectMatch) (curatedAssessment, bool) {
	switch row.Kind {
	case "ability":
		return buildAbilityAssessment(row, abilityMatches), true
	case "adversary":
		return buildAdversaryAssessment(row, advClasses), true
	case "ancestry", "community":
		return curatedAssessment{
			ReviewState:   "reviewed",
			NameStrategy:  "canonical",
			SemanticMatch: "matched",
			FinalStatus:   "covered",
			EvidenceCode: []string{
				"api/proto/systems/daggerheart/v1/content.proto",
				"internal/services/game/domain/systems/daggerheart/character_profile_contract.go",
				"internal/services/game/api/grpc/systems/daggerheart/creationworkflow/provider.go",
			},
			EvidenceTests: []string{
				"internal/services/game/api/grpc/systems/daggerheart/contenttransport/service_support_test.go",
				"internal/services/web/modules/campaigns/workflow/daggerheart/view_test.go",
			},
			EvidenceDocs: []string{
				"docs/product/daggerheart-PRD.md",
			},
			Notes: []string{
				"Heritage catalog entries are imported, selectable during creation, and persisted through the structured heritage profile contract.",
				"Single-heritage and mixed-heritage slot selection is now enforced and exposed without requiring canonical naming on the character sheet.",
			},
		}, true
	case "armor":
		if row.ReferenceID == "armor-tyris-soft-armor" {
			return curatedAssessment{
				ReviewState:   "reviewed",
				NameStrategy:  "canonical",
				SemanticMatch: "matched",
				FinalStatus:   "covered",
				EvidenceCode: []string{
					"api/proto/systems/daggerheart/v1/content.proto",
					"api/proto/systems/daggerheart/v1/service.proto",
					"internal/tools/importer/content/daggerheart/v1/armor_rules.go",
					"internal/services/game/api/grpc/systems/daggerheart/sessionrolltransport/handler.go",
				},
				EvidenceTests: []string{
					"internal/tools/importer/content/daggerheart/v1/importer_test.go",
					"internal/services/game/api/grpc/systems/daggerheart/sessionrolltransport/handler_test.go",
					"internal/test/game/scenarios/systems/daggerheart/armor_quiet.lua",
				},
				EvidenceDocs: []string{
					"docs/product/daggerheart-PRD.md",
				},
				Notes: []string{
					"Tyris Soft Armor now derives a typed quiet bonus and applies it through the action-roll transport when the declared roll context is move silently.",
				},
			}, true
		}
		if row.ReferenceID == "armor-veritas-opal-armor" {
			return curatedAssessment{
				ReviewState:   "reviewed",
				NameStrategy:  "canonical",
				SemanticMatch: "matched",
				FinalStatus:   "covered",
				EvidenceCode: []string{
					"api/proto/systems/daggerheart/v1/content.proto",
				},
				EvidenceTests: []string{},
				EvidenceDocs: []string{
					"docs/product/daggerheart-PRD.md",
					"docs/reference/daggerheart-ai-gm-guidance.md",
				},
				Notes: []string{
					"Veritas Opal Armor is treated as GM-facing narrative guidance rather than runtime automation because lie detection and conversational proximity are not first-class system mechanics.",
				},
			}, true
		}
		return curatedAssessment{
			ReviewState:   "reviewed",
			NameStrategy:  "canonical",
			SemanticMatch: "matched",
			FinalStatus:   "covered",
			EvidenceCode: []string{
				"api/proto/systems/daggerheart/v1/content.proto",
				"internal/tools/importer/content/daggerheart/v1/armor_rules.go",
				"internal/services/game/domain/systems/daggerheart/armor_profile.go",
				"internal/services/game/api/grpc/systems/daggerheart/damagetransport/handler.go",
				"internal/services/game/api/grpc/systems/daggerheart/sessionflowtransport/handler.go",
				"internal/services/game/api/grpc/systems/daggerheart/sessionrolltransport/handler.go",
			},
			EvidenceTests: []string{
				"internal/services/game/api/grpc/systems/daggerheart/contenttransport/service_support_test.go",
				"internal/services/game/api/grpc/systems/daggerheart/damagetransport/handler_test.go",
				"internal/services/game/api/grpc/systems/daggerheart/damagetransport/armor_helpers_test.go",
				"internal/services/game/api/grpc/systems/daggerheart/sessionflowtransport/handler_test.go",
				"internal/services/game/api/grpc/systems/daggerheart/sessionflowtransport/armor_helpers_test.go",
				"internal/services/game/api/grpc/systems/daggerheart/sessionrolltransport/handler_test.go",
				"internal/services/game/domain/systems/daggerheart/armor_profile_test.go",
				"internal/test/game/scenarios/systems/daggerheart/armor_hopeful_and_sharp.lua",
				"internal/test/game/scenarios/systems/daggerheart/armor_incoming_reactions.lua",
				"internal/test/game/scenarios/systems/daggerheart/armor_last_chance_reactions.lua",
			},
			EvidenceDocs: []string{
				"docs/product/daggerheart-PRD.md",
			},
			Notes: []string{
				"Equipped armor is now the runtime authority for derived evasion, trait modifiers, thresholds, mitigation, and reactive armor behavior.",
				"Deterministic and reactive armor features are modeled through typed content rules, transport validation, and scenario coverage.",
			},
		}, true
	case "beastform":
		return curatedAssessment{
			ReviewState:   "reviewed",
			NameStrategy:  "canonical",
			SemanticMatch: "matched",
			FinalStatus:   "covered",
			EvidenceCode: []string{
				"api/proto/systems/daggerheart/v1/service.proto",
				"api/proto/systems/daggerheart/v1/state.proto",
				"internal/services/game/domain/systems/daggerheart/decider_beastform.go",
				"internal/services/game/api/grpc/systems/daggerheart/charactermutationtransport/beastform.go",
				"internal/services/game/api/grpc/systems/daggerheart/sessionflowtransport/handler.go",
				"internal/services/game/api/grpc/systems/daggerheart/damagetransport/handler.go",
				"internal/services/game/domain/systems/daggerheart/mechanics_manifest.go",
			},
			EvidenceTests: []string{
				"internal/services/game/api/grpc/systems/daggerheart/charactermutationtransport/handler_test.go",
				"internal/services/game/api/grpc/systems/daggerheart/sessionflowtransport/handler_test.go",
				"internal/services/game/domain/systems/daggerheart/module_test.go",
			},
			EvidenceDocs: []string{
				"docs/reference/daggerheart-event-timeline-contract.md",
			},
			Notes: []string{
				"Beastform catalog rows are imported and resolved into first-class active beastform runtime state.",
				"Transform, drop, attack resolution, evasion, and damage-triggered auto-drop are implemented through dedicated commands, events, and transport flows.",
			},
		}, true
	case "class":
		return curatedAssessment{
			ReviewState:   "reviewed",
			NameStrategy:  "canonical",
			SemanticMatch: "matched",
			FinalStatus:   "covered",
			EvidenceCode: []string{
				"api/proto/systems/daggerheart/v1/service.proto",
				"internal/services/game/api/grpc/systems/daggerheart/charactermutationtransport/handler.go",
				"internal/services/game/domain/systems/daggerheart/decider_class_features.go",
				"internal/services/game/domain/systems/daggerheart/class_state.go",
				"internal/services/game/domain/systems/daggerheart/mechanics_manifest.go",
			},
			EvidenceTests: []string{
				"internal/services/game/api/grpc/systems/daggerheart/charactermutationtransport/handler_test.go",
				"internal/services/game/domain/systems/daggerheart/module_test.go",
				"internal/test/game/scenarios/systems/daggerheart/class_feature_core.lua",
			},
			EvidenceDocs: []string{
				"docs/reference/daggerheart-event-timeline-contract.md",
			},
			Notes: []string{
				"Class rows still drive starting stats, domain access, and creation-time selection.",
				"Activated class features and hope-feature consequences now run through typed class-feature commands, consequence batches, and persistent class state where needed.",
			},
		}, true
	case "consumable":
		return buildItemAssessment(row, itemMatches), true
	case "domain":
		return curatedAssessment{
			ReviewState:   "reviewed",
			NameStrategy:  "canonical",
			SemanticMatch: "matched",
			FinalStatus:   "covered",
			EvidenceCode: []string{
				"api/proto/systems/daggerheart/v1/content.proto",
				"internal/services/game/domain/systems/daggerheart/contentstore/contracts.go",
				"internal/services/game/api/grpc/systems/daggerheart/creationworkflow/provider.go",
			},
			EvidenceTests: []string{
				"internal/services/game/api/grpc/systems/daggerheart/contenttransport/service_support_test.go",
			},
			EvidenceDocs: []string{
				"docs/product/daggerheart-PRD.md",
			},
			Notes: []string{
				"Domain rows are catalog descriptors used for class access and card listing, and that surface is implemented in content and creation flows.",
			},
		}, true
	case "environment":
		return curatedAssessment{
			ReviewState:   "reviewed",
			NameStrategy:  "canonical",
			SemanticMatch: "matched",
			FinalStatus:   "covered",
			EvidenceCode: []string{
				"api/proto/systems/daggerheart/v1/service.proto",
				"internal/services/game/api/grpc/systems/daggerheart/environmenttransport/handler.go",
				"internal/services/game/domain/systems/daggerheart/decider_environments.go",
				"internal/services/game/domain/systems/daggerheart/mechanics_manifest.go",
			},
			EvidenceTests: []string{
				"internal/services/game/api/grpc/systems/daggerheart/gmmovetransport/handler_test.go",
			},
			EvidenceDocs: []string{
				"docs/product/daggerheart-PRD.md",
			},
			Notes: []string{
				"Environment rows are instantiated as first-class runtime entities with create/update/delete/get/list transport.",
				"GM Fear environment-feature spends now target runtime environment entities and validate feature IDs against the referenced catalog environment row.",
			},
		}, true
	case "item":
		return buildItemAssessment(row, itemMatches), true
	case "subclass":
		return curatedAssessment{
			ReviewState:   "reviewed",
			NameStrategy:  "canonical",
			SemanticMatch: "matched",
			FinalStatus:   "covered",
			EvidenceCode: []string{
				"api/proto/systems/daggerheart/v1/service.proto",
				"api/proto/systems/daggerheart/v1/state.proto",
				"internal/services/game/api/grpc/systems/daggerheart/creationworkflow/provider.go",
				"internal/services/game/domain/systems/daggerheart/subclass_tracks.go",
				"internal/services/game/api/grpc/systems/daggerheart/charactermutationtransport/subclass.go",
				"internal/services/game/api/grpc/systems/daggerheart/sessionflowtransport/handler.go",
				"internal/services/game/api/grpc/systems/daggerheart/outcometransport/handler.go",
				"internal/services/game/domain/systems/daggerheart/mechanics_manifest.go",
			},
			EvidenceTests: []string{
				"internal/services/game/api/grpc/systems/daggerheart/charactermutationtransport/handler_test.go",
				"internal/services/game/api/grpc/systems/daggerheart/sessionflowtransport/subclass_state_test.go",
				"internal/services/game/api/grpc/systems/daggerheart/sessionflowtransport/nemesis_test.go",
				"internal/test/game/scenarios/systems/daggerheart/subclass_progression_tracks.lua",
				"internal/test/game/scenarios/systems/daggerheart/subclass_multiclass_tracks.lua",
			},
			EvidenceDocs: []string{
				"docs/reference/daggerheart-event-timeline-contract.md",
			},
			Notes: []string{
				"Subclass rows are cataloged, creation-selectable, and projected through authoritative primary and multiclass progression tracks with derived active feature reads.",
				"Subclass runtime semantics now include typed activation paths, persistent subclass state, and combat or outcome consumers for the implemented feature families, including Beastbound companion-backed and beastform-dependent slices.",
			},
		}, true
	case "weapon":
		return curatedAssessment{
			ReviewState:   "reviewed",
			NameStrategy:  "canonical",
			SemanticMatch: "matched",
			FinalStatus:   "covered",
			EvidenceCode: []string{
				"api/proto/systems/daggerheart/v1/content.proto",
				"internal/services/game/api/grpc/systems/daggerheart/creationworkflow/provider.go",
				"internal/services/game/api/grpc/systems/daggerheart/damagetransport/helpers.go",
			},
			EvidenceTests: []string{
				"internal/services/game/api/grpc/systems/daggerheart/contenttransport/service_support_test.go",
				"internal/services/web/modules/campaigns/workflow/daggerheart/form_test.go",
			},
			EvidenceDocs: []string{
				"docs/reference/daggerheart-creation-workflow.md",
			},
			Notes: []string{
				"Weapon rows are structured stat entries used in catalog, creation, and damage resolution flows.",
				"Cross-weapon loadout invariants are audited separately in the character-creation rule row.",
			},
		}, true
	default:
		return curatedAssessment{}, false
	}
}

// buildAdversaryAssessment returns a per-entry assessment for an adversary
// row based on whether its features resolve to supported runtime rule kinds,
// recurring rules, or narrative-only GM guidance.
func buildAdversaryAssessment(row auditMatrixRow, advClasses map[string]adversaryEntryClass) curatedAssessment {
	evidenceCode := []string{
		"api/proto/systems/daggerheart/v1/content.proto",
		"internal/services/game/domain/systems/daggerheart/state.go",
		"internal/services/game/api/grpc/systems/daggerheart/damagetransport/handler.go",
		"internal/services/game/api/grpc/systems/daggerheart/sessionflowtransport/handler.go",
		"internal/services/game/domain/systems/daggerheart/mechanics_manifest.go",
	}
	evidenceTests := []string{
		"internal/services/game/api/grpc/systems/daggerheart/adversaries_test.go",
		"internal/services/game/api/grpc/systems/daggerheart/damagetransport/handler_test.go",
		"internal/services/game/api/grpc/systems/daggerheart/sessionflowtransport/handler_adversary_attack_flow_test.go",
	}
	evidenceDocs := []string{
		"docs/reference/daggerheart-event-timeline-contract.md",
		"docs/product/daggerheart-PRD.md",
	}

	cls, ok := advClasses[row.ReferenceID]
	if !ok || len(cls.Features) == 0 {
		// No parsed features available — adversary has no Feature section
		// or classification data was not provided. Base stats, damage, and
		// conditions are already modeled.
		return curatedAssessment{
			ReviewState:   "reviewed",
			NameStrategy:  "canonical",
			SemanticMatch: "matched",
			FinalStatus:   "covered",
			EvidenceCode:  evidenceCode,
			EvidenceTests: evidenceTests,
			EvidenceDocs:  evidenceDocs,
			Notes: []string{
				"Adversary catalog, runtime state, damage application, and condition changes exist.",
				"Entry has no typed features requiring additional runtime automation.",
			},
		}
	}

	if cls.allCovered() {
		notes := []string{
			"Adversary catalog, runtime state, damage application, and condition changes exist.",
		}
		if cls.SupportedCount > 0 && cls.NarrativeCount == 0 && cls.RecurringCount == 0 {
			notes = append(notes, fmt.Sprintf("All %d features resolve to supported runtime rule kinds.", cls.SupportedCount))
		} else {
			parts := []string{}
			if cls.SupportedCount > 0 {
				parts = append(parts, fmt.Sprintf("%d runtime-supported", cls.SupportedCount))
			}
			if cls.RecurringCount > 0 {
				parts = append(parts, fmt.Sprintf("%d recurring-rule", cls.RecurringCount))
			}
			if cls.NarrativeCount > 0 {
				parts = append(parts, fmt.Sprintf("%d narrative-only GM guidance", cls.NarrativeCount))
			}
			notes = append(notes, fmt.Sprintf("Features: %s.", strings.Join(parts, ", ")))
		}
		return curatedAssessment{
			ReviewState:   "reviewed",
			NameStrategy:  "canonical",
			SemanticMatch: "matched",
			FinalStatus:   "covered",
			EvidenceCode:  evidenceCode,
			EvidenceTests: evidenceTests,
			EvidenceDocs:  evidenceDocs,
			Notes:         notes,
		}
	}

	// Should not reach here given the current classification logic (all
	// features resolve to supported, recurring, or narrative), but keep
	// a fallback gap path for safety.
	return curatedAssessment{
		ReviewState:   "reviewed",
		NameStrategy:  "canonical",
		SemanticMatch: "partial",
		FinalStatus:   "gap",
		GapClass:      "behavior",
		EvidenceCode:  evidenceCode,
		EvidenceTests: evidenceTests,
		EvidenceDocs:  evidenceDocs,
		Notes: []string{
			"Adversary catalog, runtime state, damage application, and condition changes exist.",
			"Some features may require additional runtime support beyond the current 10 typed rule kinds.",
		},
		FollowUpEpic: "adversary-feature-parity",
	}
}

// buildAbilityAssessment returns a per-entry assessment for an ability row
// based on whether it resolves to a domain card in the content catalog and
// whether its mechanical effects can be expressed through mutation primitives.
func buildAbilityAssessment(row auditMatrixRow, abilityMatches map[string]abilityDomainCardMatch) curatedAssessment {
	evidenceCode := []string{
		"api/proto/systems/daggerheart/v1/content.proto",
		"internal/services/game/domain/systems/daggerheart/contentstore/contracts.go",
		"internal/tools/importer/content/daggerheart/v1/",
	}
	evidenceTests := []string{
		"internal/services/game/api/grpc/systems/daggerheart/contenttransport/service_support_test.go",
	}
	evidenceDocs := []string{
		"docs/product/daggerheart-PRD.md",
	}

	match, ok := abilityMatches[row.ReferenceID]
	if !ok {
		return curatedAssessment{
			ReviewState:   "reviewed",
			NameStrategy:  "canonical",
			SemanticMatch: "ambiguous",
			FinalStatus:   "gap",
			GapClass:      "ambiguous_mapping",
			EvidenceCode:  evidenceCode,
			EvidenceTests: evidenceTests,
			EvidenceDocs:  evidenceDocs,
			Notes: []string{
				"Reference ability does not resolve to a domain card by name.",
				"May map to a class feature, subclass feature, or other runtime mechanic.",
			},
			FollowUpEpic: "ability-mapping-and-semantic-audit",
		}
	}

	classification := classifyAbilityEffects(match.FeatureText)

	baseNotes := []string{
		fmt.Sprintf("Ability maps to domain card %s in domain %s.", match.DomainCardID, match.DomainID),
		"Domain cards are imported, cataloged, and served through content surfaces with acquisition, loadout, and recall-cost mechanics.",
	}
	effectNote := fmt.Sprintf("Detected effects: %s (expressibility: %s).",
		joinEffectCategories(classification.Effects), string(classification.Expressibility))
	baseNotes = append(baseNotes, effectNote)

	// Merge scenario evidence into test evidence.
	for _, s := range classification.Scenarios {
		evidenceTests = appendUnique(evidenceTests, s)
	}

	switch classification.Expressibility {
	case missingPrimitive:
		return curatedAssessment{
			ReviewState:   "reviewed",
			NameStrategy:  "canonical",
			SemanticMatch: "matched",
			FinalStatus:   "gap",
			GapClass:      "missing_primitive",
			EvidenceCode:  evidenceCode,
			EvidenceTests: evidenceTests,
			EvidenceDocs:  evidenceDocs,
			Notes:         baseNotes,
			FollowUpEpic:  "domain-card-primitive-gaps",
		}
	default:
		return curatedAssessment{
			ReviewState:   "reviewed",
			NameStrategy:  "canonical",
			SemanticMatch: "matched",
			FinalStatus:   "covered",
			EvidenceCode:  evidenceCode,
			EvidenceTests: evidenceTests,
			EvidenceDocs:  evidenceDocs,
			Notes:         baseNotes,
		}
	}
}

func joinEffectCategories(effects []abilityEffectCategory) string {
	parts := make([]string, len(effects))
	for i, e := range effects {
		parts[i] = string(e)
	}
	return strings.Join(parts, ", ")
}

// applyCuratedAssessment overlays durable milestone conclusions onto the
// generated default row so regeneration remains deterministic.
func applyCuratedAssessment(row *auditMatrixRow, advClasses map[string]adversaryEntryClass, abilityMatches map[string]abilityDomainCardMatch, itemMatches map[string]itemEffectMatch) {
	if row == nil {
		return
	}
	assessment, ok := curatedAssessments[row.ReferenceID]
	if !ok {
		assessment, ok = baselineAssessmentForRow(*row, advClasses, abilityMatches, itemMatches)
		if !ok {
			return
		}
	}
	if assessment.ReviewState != "" {
		row.ReviewState = assessment.ReviewState
	}
	if assessment.NameStrategy != "" {
		row.NameStrategy = assessment.NameStrategy
	}
	if assessment.SemanticMatch != "" {
		row.SemanticMatch = assessment.SemanticMatch
	}
	if assessment.FinalStatus != "" {
		row.FinalStatus = assessment.FinalStatus
	}
	if assessment.GapClass != "" {
		row.GapClass = assessment.GapClass
	}
	row.EvidenceCode = append([]string(nil), assessment.EvidenceCode...)
	row.EvidenceTests = append([]string(nil), assessment.EvidenceTests...)
	row.EvidenceDocs = append([]string(nil), assessment.EvidenceDocs...)
	row.Notes = append([]string(nil), assessment.Notes...)
	row.FollowUpEpic = assessment.FollowUpEpic
}

// validateAuditRow enforces the matrix contract so reviewed rows cannot drift
// into undocumented or partially-classified states.
func validateAuditRow(row auditMatrixRow, entry corpusIndexEntry, requireFinalStatus bool) error {
	if strings.TrimSpace(row.Kind) == "" || row.Kind != entry.Kind {
		return fmt.Errorf("audit row %q kind = %q, want %q", row.ReferenceID, row.Kind, entry.Kind)
	}
	if strings.TrimSpace(row.Path) == "" || row.Path != entry.Path {
		return fmt.Errorf("audit row %q path = %q, want %q", row.ReferenceID, row.Path, entry.Path)
	}
	if strings.TrimSpace(row.AuditArea) == "" {
		return fmt.Errorf("audit row %q missing audit_area", row.ReferenceID)
	}
	if strings.TrimSpace(row.Normativity) == "" {
		return fmt.Errorf("audit row %q missing normativity", row.ReferenceID)
	}
	if !isAllowedReviewState(row.ReviewState) {
		return fmt.Errorf("audit row %q has unsupported review_state %q", row.ReferenceID, row.ReviewState)
	}
	if !isAllowedSemanticMatch(row.SemanticMatch) {
		return fmt.Errorf("audit row %q has unsupported semantic_match %q", row.ReferenceID, row.SemanticMatch)
	}
	if len(row.RepoMappings) == 0 {
		return fmt.Errorf("audit row %q missing repo mappings", row.ReferenceID)
	}
	if len(row.SurfaceApplicability) == 0 {
		return fmt.Errorf("audit row %q missing surface applicability", row.ReferenceID)
	}
	if requireFinalStatus && strings.TrimSpace(row.FinalStatus) == "" {
		return fmt.Errorf("audit row %q missing final_status", row.ReferenceID)
	}
	if row.NameStrategy != "" && !isAllowedNameStrategy(row.NameStrategy) {
		return fmt.Errorf("audit row %q has unsupported name_strategy %q", row.ReferenceID, row.NameStrategy)
	}
	if row.FinalStatus != "" && !isAllowedFinalStatus(row.FinalStatus) {
		return fmt.Errorf("audit row %q has unsupported final_status %q", row.ReferenceID, row.FinalStatus)
	}
	if row.GapClass != "" && !isAllowedGapClass(row.GapClass) {
		return fmt.Errorf("audit row %q has unsupported gap_class %q", row.ReferenceID, row.GapClass)
	}
	if row.ReviewState != "reviewed" {
		return nil
	}
	if row.FinalStatus == "" {
		return fmt.Errorf("audit row %q is reviewed but missing final_status", row.ReferenceID)
	}
	if row.NameStrategy == "" {
		return fmt.Errorf("audit row %q is reviewed but missing name_strategy", row.ReferenceID)
	}
	if row.SemanticMatch == "unknown" {
		return fmt.Errorf("audit row %q is reviewed but semantic_match is still unknown", row.ReferenceID)
	}
	if len(row.EvidenceCode) == 0 && len(row.EvidenceTests) == 0 && len(row.EvidenceDocs) == 0 && len(row.Notes) == 0 {
		return fmt.Errorf("audit row %q is reviewed but has no evidence or rationale", row.ReferenceID)
	}
	switch row.FinalStatus {
	case "covered":
		if row.GapClass != "" || row.FollowUpEpic != "" {
			return fmt.Errorf("audit row %q is covered but still records gap metadata", row.ReferenceID)
		}
	case "gap":
		if row.GapClass == "" {
			return fmt.Errorf("audit row %q is a gap but missing gap_class", row.ReferenceID)
		}
		if strings.TrimSpace(row.FollowUpEpic) == "" {
			return fmt.Errorf("audit row %q is a gap but missing follow_up_epic", row.ReferenceID)
		}
	case "not_applicable":
		if row.GapClass != "" || row.FollowUpEpic != "" {
			return fmt.Errorf("audit row %q is not_applicable but still records gap metadata", row.ReferenceID)
		}
	}
	return nil
}

func isAllowedReviewState(value string) bool {
	switch value {
	case "pending", "reviewed":
		return true
	default:
		return false
	}
}

func isAllowedSemanticMatch(value string) bool {
	switch value {
	case "unknown", "matched", "partial", "ambiguous", "not_applicable":
		return true
	default:
		return false
	}
}

func isAllowedFinalStatus(value string) bool {
	switch value {
	case "covered", "gap", "not_applicable":
		return true
	default:
		return false
	}
}

func isAllowedGapClass(value string) bool {
	switch value {
	case "behavior", "missing_model", "missing_primitive", "content_schema", "surface_parity", "test_gap", "repo_doc_drift", "ambiguous_mapping":
		return true
	default:
		return false
	}
}

func summaryBucket(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unreviewed"
	}
	return value
}
