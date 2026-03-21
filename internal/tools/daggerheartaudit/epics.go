package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type epicDefinition struct {
	Title            string
	Priority         string
	Summary          string
	Boundary         string
	DependsOn        []string
	ContractsToTouch []string
	TestsRequired    []string
	RemovalCriteria  []string
}

var epicDefinitions = map[string]epicDefinition{
	"ability-mapping-and-semantic-audit": {
		Title:    "Ability Mapping And Semantic Audit",
		Priority: "p3",
		Summary:  "Introduce an authoritative mapping layer from extracted ability rows to domain cards, class features, subclass features, or explicit runtime mechanics so ability coverage is proven instead of inferred.",
		Boundary: "Content-to-runtime semantic mapping for ability-shaped reference entries.",
		ContractsToTouch: []string{
			"internal/services/game/domain/systems/daggerheart/contentstore/contracts.go",
			"internal/services/game/domain/systems/daggerheart/mechanics_manifest.go",
			"api/proto/systems/daggerheart/v1/content.proto",
			"internal/tools/importer/content/daggerheart/v1/",
		},
		TestsRequired: []string{
			"Tooling tests that every ability row resolves to a stable runtime mapping or an explicit unsupported rationale.",
			"Content transport tests for any new ability-derived catalog or metadata fields.",
			"Targeted domain or transport tests for newly mapped ability mechanics.",
		},
		RemovalCriteria: []string{
			"Remove temporary alias tables once every ability row points at an authoritative runtime or catalog contract.",
			"Drop provisional ambiguous-mapping status once the audit can prove semantic equivalence automatically.",
		},
	},
	"adversary-feature-parity": {
		Title:    "Adversary Feature Parity",
		Priority: "p2",
		Summary:  "Close the gap between adversary catalog entries and runtime adversary behavior, especially fear features, move semantics, and entry-specific rules.",
		Boundary: "Adversary runtime semantics beyond base stats, conditions, and damage application.",
		ContractsToTouch: []string{
			"internal/services/game/domain/systems/daggerheart/state.go",
			"internal/services/game/api/grpc/systems/daggerheart/damagetransport/",
			"internal/services/game/api/grpc/systems/daggerheart/gmmovetransport/",
			"api/proto/systems/daggerheart/v1/content.proto",
		},
		TestsRequired: []string{
			"Domain tests for adversary feature activation and state transitions.",
			"gRPC integration tests for adversary feature execution paths.",
			"Scenario coverage for representative adversary fear-feature interactions.",
		},
		RemovalCriteria: []string{
			"Remove entry-specific special cases once adversary feature execution is data-driven or otherwise uniformly modeled.",
		},
	},
	"beastform-mechanics": {
		Title:    "Beastform Mechanics",
		Priority: "p2",
		Summary:  "Model beastform transformation, attack semantics, and state transitions as first-class runtime behavior instead of content-only catalog rows.",
		Boundary: "Character transformation and beastform-specific combat/profile behavior.",
		ContractsToTouch: []string{
			"internal/services/game/domain/systems/daggerheart/",
			"internal/services/game/api/grpc/systems/daggerheart/",
			"api/proto/systems/daggerheart/v1/state.proto",
			"api/proto/systems/daggerheart/v1/content.proto",
		},
		TestsRequired: []string{
			"Domain tests for transform/drop-form invariants and armor/HP interactions.",
			"Transport tests for beastform command handling and read-side shaping.",
			"Scenario coverage for representative beastform combat turns.",
		},
		RemovalCriteria: []string{
			"Remove manifest TODO markers and any temporary content-only fallbacks once beastform state is runtime-owned.",
		},
	},
	"class-feature-modeling": {
		Title:    "Class Feature Modeling",
		Priority: "p2",
		Summary:  "Promote class hope features and starting class features from descriptive content into explicit runtime contracts and execution paths.",
		Boundary: "Class-owned mechanics and character state hooks.",
		ContractsToTouch: []string{
			"internal/services/game/domain/systems/daggerheart/character_profile_contract.go",
			"internal/services/game/domain/systems/daggerheart/",
			"api/proto/systems/daggerheart/v1/state.proto",
			"api/proto/systems/daggerheart/v1/content.proto",
		},
		TestsRequired: []string{
			"Domain tests for representative class feature activation.",
			"Creation workflow tests proving class-selected mechanics persist correctly.",
			"Transport tests for class-feature request/response shaping where exposed.",
		},
		RemovalCriteria: []string{
			"Remove doc-only reliance for class mechanics once each class feature has an explicit runtime path or declared non-goal.",
		},
	},
	"condition-model-expansion": {
		Title:    "Condition Model Expansion",
		Priority: "p2",
		Summary:  "Extend condition handling beyond the standard named set so temporary tags, special conditions, and clearing rules are first-class and auditable.",
		Boundary: "Condition lifecycle, storage shape, and clearing semantics.",
		ContractsToTouch: []string{
			"internal/services/game/domain/systems/daggerheart/compat_conditions.go",
			"internal/services/game/api/grpc/systems/daggerheart/workfloweffects/",
			"api/proto/systems/daggerheart/v1/state.proto",
		},
		TestsRequired: []string{
			"Domain tests for temporary and special-condition clearing invariants.",
			"Workflow effect tests for replay-safe condition application/removal.",
			"Transport tests for serialized condition state.",
		},
		RemovalCriteria: []string{
			"Remove ad hoc per-feature condition logic once condition lifecycle rules live in a shared model.",
		},
	},
	"core-mechanics-alignment": {
		Title:    "Core Mechanics Alignment",
		Priority: "p2",
		Summary:  "Reconcile remaining umbrella-level mechanics drift so resource, recovery, fear, and condition rules match the extracted reference at the system boundary.",
		Boundary: "Cross-cutting Daggerheart mechanics that do not fit a narrower epic cleanly.",
		ContractsToTouch: []string{
			"internal/services/game/domain/systems/daggerheart/",
			"internal/services/game/api/grpc/systems/daggerheart/",
			"docs/reference/daggerheart-event-timeline-contract.md",
		},
		TestsRequired: []string{
			"Cross-package domain and transport coverage for the aligned mechanic slices.",
			"Audit-tooling regression checks for the previously drifted clauses.",
		},
		RemovalCriteria: []string{
			"Delete temporary umbrella exceptions from the audit once narrower epics close the underlying drift.",
		},
	},
	"creation-workflow-alignment": {
		Title:    "Creation Workflow Alignment",
		Priority: "p1",
		Summary:  "Align the creation workflow ordering and readiness contract across domain, gRPC, web, MCP, and docs with the extracted reference.",
		Boundary: "Character creation sequencing and step-owned fields.",
		ContractsToTouch: []string{
			"internal/services/game/domain/systems/daggerheart/creation_workflow.go",
			"internal/services/game/api/grpc/systems/daggerheart/creationworkflow/",
			"internal/services/web/modules/campaigns/workflow/daggerheart/",
			"internal/services/ai/orchestration/gametools/tools_daggerheart.go",
			"api/proto/systems/daggerheart/v1/state.proto",
			"docs/reference/daggerheart-creation-workflow.md",
		},
		TestsRequired: []string{
			"Domain tests for step ordering and readiness gating.",
			"gRPC/web integration tests proving the same canonical sequence.",
			"Doc-alignment tests or fixtures where ordering is surfaced in generated content.",
		},
		RemovalCriteria: []string{
			"Remove any compatibility handling for the old equipment-before-details order once all callers use the canonical sequence.",
		},
	},
	"domain-card-primitive-gaps": {
		Title:    "Domain Card Primitive Gaps",
		Priority: "p2",
		Summary:  "Add missing mutation primitives (HP healing, Hope granting, direct stress reduction, temporary non-armor buffs) so domain card ability effects can be expressed through the command surface.",
		Boundary: "Mutation command coverage for mechanical effects described in domain card feature text.",
		ContractsToTouch: []string{
			"api/proto/systems/daggerheart/v1/service.proto",
			"internal/services/game/api/grpc/systems/daggerheart/charactermutationtransport/",
			"internal/services/game/domain/systems/daggerheart/",
		},
		TestsRequired: []string{
			"Domain tests for HP healing, Hope granting, and stress reduction commands.",
			"Transport tests for new mutation RPCs or extended request payloads.",
			"Scenario coverage reusing existing damage, condition, and rest flows to verify primitive integration.",
		},
		RemovalCriteria: []string{
			"Remove missing_primitive gap classification once each effect category has a corresponding mutation command or an explicit non-goal declaration.",
		},
	},
	"death-scar-terminal-hope": {
		Title:    "Death Scar Terminal Hope",
		Priority: "p1",
		Summary:  "Enforce the terminal outcome when scars remove the final hope slot so death-move semantics fully match the reference.",
		Boundary: "Death-move resolution and post-scar character lifecycle.",
		ContractsToTouch: []string{
			"internal/services/game/domain/systems/daggerheart/internal/mechanics/death.go",
			"internal/services/game/api/grpc/systems/daggerheart/recoverytransport/handler.go",
			"api/proto/systems/daggerheart/v1/state.proto",
		},
		TestsRequired: []string{
			"Domain tests for final-hope-slot scar outcomes.",
			"Recovery transport tests proving terminal state projection.",
		},
		RemovalCriteria: []string{
			"Remove any implicit 'alive with zero hope slots' handling once terminal-state semantics are explicit and enforced.",
		},
	},
	"downtime-surface-parity": {
		Title:    "Downtime Surface Parity",
		Priority: "p1",
		Summary:  "Bring the downtime move menu, short-rest versus long-rest behavior, and project advancement semantics into parity with the reference across domain and transport surfaces.",
		Boundary: "Rest and downtime move coverage.",
		ContractsToTouch: []string{
			"internal/services/game/domain/systems/daggerheart/internal/mechanics/rest.go",
			"internal/services/game/domain/systems/daggerheart/internal/mechanics/downtime.go",
			"internal/services/game/api/grpc/systems/daggerheart/recoverytransport/",
			"api/proto/systems/daggerheart/v1/state.proto",
		},
		TestsRequired: []string{
			"Domain tests for the full downtime move menu and interruption outcomes.",
			"Transport tests for all supported downtime move enums and payloads.",
			"Scenario coverage for representative rest flows with project advancement.",
		},
		RemovalCriteria: []string{
			"Remove temporary menu-shaping or fallback text once the full move set is first-class in proto and runtime.",
		},
	},
	"environment-entities": {
		Title:    "Environment Entities",
		Priority: "p2",
		Summary:  "Promote environment entries from content-only rows into first-class runtime entities with fear features and consequence hooks.",
		Boundary: "Environment runtime state and execution semantics.",
		ContractsToTouch: []string{
			"internal/services/game/domain/systems/daggerheart/",
			"internal/services/game/api/grpc/systems/daggerheart/",
			"api/proto/systems/daggerheart/v1/content.proto",
			"api/proto/systems/daggerheart/v1/state.proto",
		},
		TestsRequired: []string{
			"Domain tests for environment state and fear-feature execution.",
			"Transport tests for environment entity reads/writes if exposed.",
			"Scenario coverage for representative environment consequence flows.",
		},
		RemovalCriteria: []string{
			"Remove content-only treatment for environment mechanics once runtime environment entities own those behaviors.",
		},
	},
	"equipment-feature-parity": {
		Title:    "Equipment Feature Parity",
		Priority: "p2",
		Summary:  "Model armor and other equipment feature text that materially changes stats or rolls instead of limiting implementation to base scores and thresholds.",
		Boundary: "Equipment-derived modifiers and special rules.",
		ContractsToTouch: []string{
			"internal/services/game/domain/systems/daggerheart/character_profile_contract.go",
			"internal/services/game/domain/systems/daggerheart/",
			"api/proto/systems/daggerheart/v1/content.proto",
		},
		TestsRequired: []string{
			"Profile and combat tests for equipment-derived penalties and bonuses.",
			"Content transport tests for any newly structured equipment feature fields.",
		},
		RemovalCriteria: []string{
			"Remove reliance on free-form equipment text for mechanical effects once those effects are modeled structurally or explicitly declared unsupported.",
		},
	},
	"fear-initialization": {
		Title:    "Fear Initialization",
		Priority: "p1",
		Summary:  "Make first-session activation the canonical GM Fear seed and keep pre-activation snapshot defaults neutral until created-PC count is known.",
		Boundary: "First-session bootstrap semantics and pre-activation fear defaults.",
		ContractsToTouch: []string{
			"internal/services/game/domain/readiness/session_start_workflow.go",
			"internal/services/game/domain/systems/daggerheart/module.go",
			"internal/services/game/domain/systems/daggerheart/state.go",
			"internal/services/game/domain/systems/daggerheart/state_factory.go",
			"internal/services/game/domain/systems/daggerheart/",
		},
		TestsRequired: []string{
			"Bootstrap tests for initial fear values derived from created-PC count on first session activation.",
			"Read-side or integration coverage showing pre-activation snapshots stay neutral until bootstrap emits the fear seed.",
		},
		RemovalCriteria: []string{
			"Remove stale documentation or helper wording that implies snapshot creation owns party-size-aware fear seeding.",
		},
	},
	"gm-fear-and-moves": {
		Title:    "GM Fear And Moves",
		Priority: "p1",
		Summary:  "Expand fear spend handling into explicit GM move semantics, spotlight interruption behavior, and adversary/environment fear-feature execution.",
		Boundary: "GM-facing consequence execution and fear spend taxonomy.",
		ContractsToTouch: []string{
			"internal/services/game/api/grpc/systems/daggerheart/gmmovetransport/",
			"internal/services/game/api/grpc/systems/daggerheart/outcometransport/",
			"internal/services/game/domain/systems/daggerheart/gm_fear_rules.go",
			"api/proto/systems/daggerheart/v1/service.proto",
		},
		TestsRequired: []string{
			"Transport tests for each supported fear spend target or GM move kind.",
			"Domain tests for fear accounting and spend validation.",
			"Scenario coverage for spotlight-steal and consequence chains.",
		},
		RemovalCriteria: []string{
			"Remove generic text-only GM move handling once move semantics are typed and replay-safe.",
		},
	},
	"heritage-and-companion-modeling": {
		Title:    "Heritage And Companion Modeling",
		Priority: "p1",
		Summary:  "Add first-class mixed-heritage and companion modeling so ancestry/community features and companion-backed selections become durable runtime data instead of loosely coupled content.",
		Boundary: "Character profile shape for heritage composition and companion ownership.",
		ContractsToTouch: []string{
			"internal/services/game/domain/systems/daggerheart/character_profile_contract.go",
			"internal/services/game/api/grpc/systems/daggerheart/creationworkflow/",
			"internal/services/web/modules/campaigns/workflow/daggerheart/",
			"api/proto/systems/daggerheart/v1/state.proto",
			"api/proto/systems/daggerheart/v1/content.proto",
		},
		TestsRequired: []string{
			"Creation workflow tests for mixed heritage and companion-required subclasses.",
			"Profile/projection tests for persisted heritage and companion state.",
			"Web flow tests for the new selection paths.",
		},
		RemovalCriteria: []string{
			"Remove single-ancestry-only assumptions and any temporary companion placeholders once the profile/proto schema is authoritative.",
		},
	},
	"item-use-modeling": {
		Title:    "Item Use Modeling",
		Priority: "p3",
		Summary:  "Promote mechanically meaningful item and consumable entries into explicit runtime item-use behavior rather than leaving them as descriptive catalog text.",
		Boundary: "Inventory- and item-use semantics for Daggerheart content rows.",
		ContractsToTouch: []string{
			"internal/services/game/domain/systems/daggerheart/",
			"internal/services/game/api/grpc/systems/daggerheart/charactermutationtransport/",
			"api/proto/systems/daggerheart/v1/content.proto",
			"api/proto/systems/daggerheart/v1/state.proto",
		},
		TestsRequired: []string{
			"Domain tests for representative item/consumable effect execution.",
			"Transport tests for inventory mutation and item-use commands.",
			"Importer/content tests for any new structured item effect fields.",
		},
		RemovalCriteria: []string{
			"Remove content-only handling for mechanical items once runtime item-use commands or explicit non-goals replace it.",
		},
	},
	"subclass-feature-modeling": {
		Title:    "Subclass Feature Modeling",
		Priority: "p2",
		Summary:  "Promote subclass foundation, specialization, and mastery feature semantics into explicit runtime contracts, including companion-bound subclasses.",
		Boundary: "Subclass-owned mechanics and feature progression.",
		DependsOn: []string{
			"heritage-and-companion-modeling",
		},
		ContractsToTouch: []string{
			"internal/services/game/domain/systems/daggerheart/",
			"internal/services/game/api/grpc/systems/daggerheart/",
			"api/proto/systems/daggerheart/v1/state.proto",
			"api/proto/systems/daggerheart/v1/content.proto",
		},
		TestsRequired: []string{
			"Domain tests for representative subclass feature activation and progression.",
			"Creation and level-up tests for subclass-specific requirements.",
			"Transport tests where subclass features surface directly.",
		},
		RemovalCriteria: []string{
			"Remove subclass feature reliance on descriptive text once runtime paths or explicit non-goals are in place.",
		},
	},
}

type epicAccumulator struct {
	def             epicDefinition
	rowCount        int
	kindCounts      map[string]int
	auditAreaCounts map[string]int
	gapClassCounts  map[string]int
	sampleRefs      []string
	evidenceCodeSet map[string]struct{}
	evidenceTestSet map[string]struct{}
	evidenceDocSet  map[string]struct{}
}

func buildEpicCatalog(referenceRoot string, rows []auditMatrixRow) (generatedEpicCatalog, string, error) {
	gapRows := make([]auditMatrixRow, 0)
	for _, row := range rows {
		if row.FinalStatus == "gap" {
			gapRows = append(gapRows, row)
		}
	}

	accs := map[string]*epicAccumulator{}
	for _, row := range gapRows {
		epicID := strings.TrimSpace(row.FollowUpEpic)
		if epicID == "" {
			return generatedEpicCatalog{}, "", fmt.Errorf("gap row %q missing follow_up_epic", row.ReferenceID)
		}
		def, ok := epicDefinitions[epicID]
		if !ok {
			return generatedEpicCatalog{}, "", fmt.Errorf("gap row %q references undefined epic %q", row.ReferenceID, epicID)
		}
		acc, ok := accs[epicID]
		if !ok {
			acc = &epicAccumulator{
				def:             def,
				kindCounts:      map[string]int{},
				auditAreaCounts: map[string]int{},
				gapClassCounts:  map[string]int{},
				evidenceCodeSet: map[string]struct{}{},
				evidenceTestSet: map[string]struct{}{},
				evidenceDocSet:  map[string]struct{}{},
			}
			accs[epicID] = acc
		}
		acc.rowCount++
		acc.kindCounts[row.Kind]++
		acc.auditAreaCounts[row.AuditArea]++
		acc.gapClassCounts[row.GapClass]++
		if len(acc.sampleRefs) < 8 {
			acc.sampleRefs = append(acc.sampleRefs, row.ReferenceID)
		}
		for _, path := range row.EvidenceCode {
			acc.evidenceCodeSet[path] = struct{}{}
		}
		for _, path := range row.EvidenceTests {
			acc.evidenceTestSet[path] = struct{}{}
		}
		for _, path := range row.EvidenceDocs {
			acc.evidenceDocSet[path] = struct{}{}
		}
	}

	epics := make([]generatedEpic, 0, len(accs))
	for epicID, acc := range accs {
		epics = append(epics, generatedEpic{
			ID:                epicID,
			Title:             acc.def.Title,
			Priority:          acc.def.Priority,
			Summary:           acc.def.Summary,
			Boundary:          acc.def.Boundary,
			DependsOn:         append([]string(nil), acc.def.DependsOn...),
			ContractsToTouch:  append([]string(nil), acc.def.ContractsToTouch...),
			TestsRequired:     append([]string(nil), acc.def.TestsRequired...),
			RemovalCriteria:   append([]string(nil), acc.def.RemovalCriteria...),
			RowCount:          acc.rowCount,
			CountsByKind:      countsToSummary(acc.kindCounts),
			CountsByAuditArea: countsToSummary(acc.auditAreaCounts),
			CountsByGapClass:  countsToSummary(acc.gapClassCounts),
			SampleReferenceID: append([]string(nil), acc.sampleRefs...),
			EvidenceCode:      sortedLimitedSet(acc.evidenceCodeSet, 12),
			EvidenceTests:     sortedLimitedSet(acc.evidenceTestSet, 12),
			EvidenceDocs:      sortedLimitedSet(acc.evidenceDocSet, 8),
		})
	}
	sort.Slice(epics, func(i, j int) bool {
		if epicPriorityRank(epics[i].Priority) == epicPriorityRank(epics[j].Priority) {
			if epics[i].RowCount == epics[j].RowCount {
				return epics[i].ID < epics[j].ID
			}
			return epics[i].RowCount > epics[j].RowCount
		}
		return epicPriorityRank(epics[i].Priority) < epicPriorityRank(epics[j].Priority)
	})

	catalog := generatedEpicCatalog{
		Version:       inventoryVersion,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		ReferenceRoot: referenceRoot,
		GapRowCount:   len(gapRows),
		EpicCount:     len(epics),
		Epics:         epics,
	}
	return catalog, renderEpicBacklogMarkdown(catalog), nil
}

func validateEpicCatalog(catalog generatedEpicCatalog, rows []auditMatrixRow) error {
	gapRows := 0
	expectedIDs := map[string]int{}
	for _, row := range rows {
		if row.FinalStatus != "gap" {
			continue
		}
		gapRows++
		expectedIDs[row.FollowUpEpic]++
	}
	if catalog.GapRowCount != gapRows {
		return fmt.Errorf("epic catalog gap_row_count = %d, want %d", catalog.GapRowCount, gapRows)
	}
	if catalog.EpicCount != len(expectedIDs) {
		return fmt.Errorf("epic catalog epic_count = %d, want %d", catalog.EpicCount, len(expectedIDs))
	}
	seen := map[string]int{}
	for _, epic := range catalog.Epics {
		if strings.TrimSpace(epic.ID) == "" {
			return fmt.Errorf("epic catalog contains empty epic id")
		}
		if _, ok := expectedIDs[epic.ID]; !ok {
			return fmt.Errorf("epic catalog contains unexpected epic %q", epic.ID)
		}
		if epic.RowCount != expectedIDs[epic.ID] {
			return fmt.Errorf("epic %q row_count = %d, want %d", epic.ID, epic.RowCount, expectedIDs[epic.ID])
		}
		if strings.TrimSpace(epic.Title) == "" || strings.TrimSpace(epic.Summary) == "" || strings.TrimSpace(epic.Boundary) == "" {
			return fmt.Errorf("epic %q is missing title, summary, or boundary", epic.ID)
		}
		if len(epic.ContractsToTouch) == 0 || len(epic.TestsRequired) == 0 || len(epic.RemovalCriteria) == 0 {
			return fmt.Errorf("epic %q is missing contract, test, or removal criteria metadata", epic.ID)
		}
		if len(epic.SampleReferenceID) == 0 {
			return fmt.Errorf("epic %q has no sample reference ids", epic.ID)
		}
		seen[epic.ID]++
	}
	if len(seen) != len(expectedIDs) {
		return fmt.Errorf("epic catalog covers %d epics, want %d", len(seen), len(expectedIDs))
	}
	return nil
}

func renderEpicBacklogMarkdown(catalog generatedEpicCatalog) string {
	var b strings.Builder
	b.WriteString("# Remediation Backlog\n\n")
	b.WriteString("Generated backlog synthesized from `audit_matrix.json` gap rows.\n\n")
	b.WriteString(fmt.Sprintf("- Gap rows: %d\n", catalog.GapRowCount))
	b.WriteString(fmt.Sprintf("- Epics: %d\n\n", catalog.EpicCount))
	for _, epic := range catalog.Epics {
		b.WriteString("## " + epic.Title + "\n\n")
		b.WriteString(fmt.Sprintf("- Epic ID: `%s`\n", epic.ID))
		b.WriteString(fmt.Sprintf("- Priority: `%s`\n", epic.Priority))
		b.WriteString(fmt.Sprintf("- Gap rows: %d\n", epic.RowCount))
		b.WriteString(fmt.Sprintf("- Boundary: %s\n", epic.Boundary))
		if len(epic.DependsOn) > 0 {
			b.WriteString(fmt.Sprintf("- Depends on: `%s`\n", strings.Join(epic.DependsOn, "`, `")))
		}
		b.WriteString("- Summary: " + epic.Summary + "\n")
		b.WriteString(fmt.Sprintf("- Kinds: %s\n", formatSummaryItems(epic.CountsByKind)))
		b.WriteString(fmt.Sprintf("- Audit areas: %s\n", formatSummaryItems(epic.CountsByAuditArea)))
		b.WriteString(fmt.Sprintf("- Gap classes: %s\n", formatSummaryItems(epic.CountsByGapClass)))
		b.WriteString(fmt.Sprintf("- Sample references: `%s`\n", strings.Join(epic.SampleReferenceID, "`, `")))
		b.WriteString("\nContracts to touch:\n")
		for _, item := range epic.ContractsToTouch {
			b.WriteString("- `" + item + "`\n")
		}
		b.WriteString("\nTests required:\n")
		for _, item := range epic.TestsRequired {
			b.WriteString("- " + item + "\n")
		}
		b.WriteString("\nRemoval criteria:\n")
		for _, item := range epic.RemovalCriteria {
			b.WriteString("- " + item + "\n")
		}
		if len(epic.EvidenceCode) > 0 {
			b.WriteString("\nRepresentative code evidence:\n")
			for _, item := range epic.EvidenceCode {
				b.WriteString("- `" + item + "`\n")
			}
		}
		if len(epic.EvidenceTests) > 0 {
			b.WriteString("\nRepresentative test evidence:\n")
			for _, item := range epic.EvidenceTests {
				b.WriteString("- `" + item + "`\n")
			}
		}
		if len(epic.EvidenceDocs) > 0 {
			b.WriteString("\nRepresentative docs:\n")
			for _, item := range epic.EvidenceDocs {
				b.WriteString("- `" + item + "`\n")
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func epicPriorityRank(priority string) int {
	switch priority {
	case "p1":
		return 1
	case "p2":
		return 2
	case "p3":
		return 3
	default:
		return 99
	}
}

func sortedLimitedSet(values map[string]struct{}, limit int) []string {
	items := make([]string, 0, len(values))
	for item := range values {
		items = append(items, item)
	}
	sort.Strings(items)
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func formatSummaryItems(items []summaryItem) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%s=%d", item.Key, item.Count))
	}
	return strings.Join(parts, ", ")
}
