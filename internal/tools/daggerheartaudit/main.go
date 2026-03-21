package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/tools/cli"
)

const usage = `daggerheartaudit generates and validates the Daggerheart reference audit workspace.

Usage:
  go run ./internal/tools/daggerheartaudit generate \
    -reference-root ~/code/daggerheart/reference-corpus/v1/reference \
    -out-dir .agents/plans/daggerheart-reference-audit

  go run ./internal/tools/daggerheartaudit check \
    -reference-root ~/code/daggerheart/reference-corpus/v1/reference \
    -out-dir .agents/plans/daggerheart-reference-audit
`

const (
	inventoryVersion = 1
)

var (
	reStepSplit      = regexp.MustCompile(`(?i)\bSTEP\s+[0-9]+\b`)
	reBulletSplit    = regexp.MustCompile(`\s+•\s+`)
	reWhitespace     = regexp.MustCompile(`\s+`)
	reHeading        = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	reFrontMatterSep = regexp.MustCompile(`^---\s*$`)
)

type corpusIndexEntry struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Kind    string   `json:"kind"`
	Path    string   `json:"path"`
	Aliases []string `json:"aliases,omitempty"`
}

type repoMapping struct {
	Surface string   `json:"surface"`
	Paths   []string `json:"paths"`
	Notes   string   `json:"notes,omitempty"`
}

type surfaceApplicability struct {
	Surface string `json:"surface"`
	State   string `json:"state"`
	Notes   string `json:"notes,omitempty"`
}

type auditMatrixRow struct {
	ReferenceID          string                 `json:"reference_id"`
	Title                string                 `json:"title"`
	Kind                 string                 `json:"kind"`
	Path                 string                 `json:"path"`
	Aliases              []string               `json:"aliases,omitempty"`
	AuditArea            string                 `json:"audit_area"`
	Normativity          string                 `json:"normativity"`
	ReviewState          string                 `json:"review_state"`
	RepoMappings         []repoMapping          `json:"repo_mappings"`
	SurfaceApplicability []surfaceApplicability `json:"surface_applicability"`
	NameStrategy         string                 `json:"name_strategy,omitempty"`
	SemanticMatch        string                 `json:"semantic_match"`
	FinalStatus          string                 `json:"final_status,omitempty"`
	GapClass             string                 `json:"gap_class,omitempty"`
	EvidenceCode         []string               `json:"evidence_code,omitempty"`
	EvidenceTests        []string               `json:"evidence_tests,omitempty"`
	EvidenceDocs         []string               `json:"evidence_docs,omitempty"`
	Notes                []string               `json:"notes,omitempty"`
	FollowUpEpic         string                 `json:"follow_up_epic,omitempty"`
}

type ruleClause struct {
	ClauseID    string `json:"clause_id"`
	ReferenceID string `json:"reference_id"`
	Kind        string `json:"kind"`
	Path        string `json:"path"`
	AuditArea   string `json:"audit_area"`
	Section     string `json:"section"`
	Text        string `json:"text"`
}

type generatedInventory struct {
	Version       int                `json:"version"`
	GeneratedAt   string             `json:"generated_at"`
	ReferenceRoot string             `json:"reference_root"`
	EntryCount    int                `json:"entry_count"`
	Entries       []corpusIndexEntry `json:"entries"`
}

type generatedAuditMatrix struct {
	Version       int              `json:"version"`
	GeneratedAt   string           `json:"generated_at"`
	ReferenceRoot string           `json:"reference_root"`
	RowCount      int              `json:"row_count"`
	Rows          []auditMatrixRow `json:"rows"`
}

type generatedRuleClauses struct {
	Version       int          `json:"version"`
	GeneratedAt   string       `json:"generated_at"`
	ReferenceRoot string       `json:"reference_root"`
	ClauseCount   int          `json:"clause_count"`
	Clauses       []ruleClause `json:"clauses"`
}

type summaryItem struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

type generatedSummary struct {
	Version             int           `json:"version"`
	GeneratedAt         string        `json:"generated_at"`
	ReferenceRoot       string        `json:"reference_root"`
	EntryCount          int           `json:"entry_count"`
	RuleClauseCount     int           `json:"rule_clause_count"`
	CountsByKind        []summaryItem `json:"counts_by_kind"`
	CountsByAuditArea   []summaryItem `json:"counts_by_audit_area"`
	CountsByNormativity []summaryItem `json:"counts_by_normativity"`
	CountsByReviewState []summaryItem `json:"counts_by_review_state"`
	CountsByFinalStatus []summaryItem `json:"counts_by_final_status"`
}

type generatedEpicCatalog struct {
	Version       int             `json:"version"`
	GeneratedAt   string          `json:"generated_at"`
	ReferenceRoot string          `json:"reference_root"`
	GapRowCount   int             `json:"gap_row_count"`
	EpicCount     int             `json:"epic_count"`
	Epics         []generatedEpic `json:"epics"`
}

type generatedEpic struct {
	ID                string        `json:"id"`
	Title             string        `json:"title"`
	Priority          string        `json:"priority"`
	Summary           string        `json:"summary"`
	Boundary          string        `json:"boundary"`
	DependsOn         []string      `json:"depends_on,omitempty"`
	ContractsToTouch  []string      `json:"contracts_to_touch"`
	TestsRequired     []string      `json:"tests_required"`
	RemovalCriteria   []string      `json:"removal_criteria"`
	RowCount          int           `json:"row_count"`
	CountsByKind      []summaryItem `json:"counts_by_kind"`
	CountsByAuditArea []summaryItem `json:"counts_by_audit_area"`
	CountsByGapClass  []summaryItem `json:"counts_by_gap_class"`
	SampleReferenceID []string      `json:"sample_reference_ids"`
	EvidenceCode      []string      `json:"evidence_code,omitempty"`
	EvidenceTests     []string      `json:"evidence_tests,omitempty"`
	EvidenceDocs      []string      `json:"evidence_docs,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		fatalf("%s", usage)
	}
	switch os.Args[1] {
	case "generate":
		if err := runGenerate(os.Args[2:]); err != nil {
			fatalf("%v", err)
		}
	case "check":
		if err := runCheck(os.Args[2:]); err != nil {
			fatalf("%v", err)
		}
	default:
		fatalf("unknown subcommand %q\n\n%s", os.Args[1], usage)
	}
}

func runGenerate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	rootFlag := fs.String("root", "", "repo root (defaults to locating go.mod)")
	referenceRootFlag := fs.String("reference-root", defaultReferenceRoot(), "reference corpus root")
	outDirFlag := fs.String("out-dir", ".agents/plans/daggerheart-reference-audit", "generated audit workspace")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	root, err := cli.ResolveRoot(*rootFlag)
	if err != nil {
		return fmt.Errorf("resolve repo root: %w", err)
	}
	referenceRoot := filepath.Clean(*referenceRootFlag)
	outDir := cli.ResolvePath(root, *outDirFlag)

	entries, err := loadIndexEntries(referenceRoot)
	if err != nil {
		return err
	}
	rows := buildAuditMatrix(entries)
	clauses, err := buildRuleClauses(referenceRoot, entries)
	if err != nil {
		return err
	}
	inventory := generatedInventory{
		Version:       inventoryVersion,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		ReferenceRoot: referenceRoot,
		EntryCount:    len(entries),
		Entries:       entries,
	}
	matrix := generatedAuditMatrix{
		Version:       inventoryVersion,
		GeneratedAt:   inventory.GeneratedAt,
		ReferenceRoot: referenceRoot,
		RowCount:      len(rows),
		Rows:          rows,
	}
	ruleSet := generatedRuleClauses{
		Version:       inventoryVersion,
		GeneratedAt:   inventory.GeneratedAt,
		ReferenceRoot: referenceRoot,
		ClauseCount:   len(clauses),
		Clauses:       clauses,
	}
	summary := buildSummary(referenceRoot, rows, clauses)
	epics, backlogMarkdown, err := buildEpicCatalog(referenceRoot, rows)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	if err := writeJSON(filepath.Join(outDir, "inventory.json"), inventory); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outDir, "audit_matrix.json"), matrix); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outDir, "rule_clauses.json"), ruleSet); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outDir, "summary.json"), summary); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outDir, "epics.json"), epics); err != nil {
		return err
	}
	if err := writeText(filepath.Join(outDir, "remediation_backlog.md"), backlogMarkdown); err != nil {
		return err
	}
	return nil
}

func runCheck(args []string) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	rootFlag := fs.String("root", "", "repo root (defaults to locating go.mod)")
	referenceRootFlag := fs.String("reference-root", defaultReferenceRoot(), "reference corpus root")
	outDirFlag := fs.String("out-dir", ".agents/plans/daggerheart-reference-audit", "generated audit workspace")
	requireFinalStatus := fs.Bool("require-final-status", false, "require every row to have a final status")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	root, err := cli.ResolveRoot(*rootFlag)
	if err != nil {
		return fmt.Errorf("resolve repo root: %w", err)
	}
	referenceRoot := filepath.Clean(*referenceRootFlag)
	outDir := cli.ResolvePath(root, *outDirFlag)

	entries, err := loadIndexEntries(referenceRoot)
	if err != nil {
		return err
	}

	var inventory generatedInventory
	if err := loadJSON(filepath.Join(outDir, "inventory.json"), &inventory); err != nil {
		return err
	}
	var matrix generatedAuditMatrix
	if err := loadJSON(filepath.Join(outDir, "audit_matrix.json"), &matrix); err != nil {
		return err
	}
	var clauses generatedRuleClauses
	if err := loadJSON(filepath.Join(outDir, "rule_clauses.json"), &clauses); err != nil {
		return err
	}
	var epics generatedEpicCatalog
	if err := loadJSON(filepath.Join(outDir, "epics.json"), &epics); err != nil {
		return err
	}

	if inventory.EntryCount != len(entries) {
		return fmt.Errorf("inventory entry count = %d, want %d", inventory.EntryCount, len(entries))
	}
	if matrix.RowCount != len(entries) {
		return fmt.Errorf("audit matrix row count = %d, want %d", matrix.RowCount, len(entries))
	}

	entryByID := make(map[string]corpusIndexEntry, len(entries))
	for _, entry := range entries {
		entryByID[entry.ID] = entry
	}

	seenRows := map[string]struct{}{}
	for _, row := range matrix.Rows {
		entry, ok := entryByID[row.ReferenceID]
		if !ok {
			return fmt.Errorf("audit row %q does not exist in reference index", row.ReferenceID)
		}
		if _, dup := seenRows[row.ReferenceID]; dup {
			return fmt.Errorf("duplicate audit row for %q", row.ReferenceID)
		}
		seenRows[row.ReferenceID] = struct{}{}
		if err := validateAuditRow(row, entry, *requireFinalStatus); err != nil {
			return err
		}
	}
	if len(seenRows) != len(entries) {
		return fmt.Errorf("audit matrix covers %d entries, want %d", len(seenRows), len(entries))
	}

	clauseCounts := map[string]int{}
	for _, clause := range clauses.Clauses {
		if _, ok := entryByID[clause.ReferenceID]; !ok {
			return fmt.Errorf("rule clause %q references unknown entry %q", clause.ClauseID, clause.ReferenceID)
		}
		if strings.TrimSpace(clause.Text) == "" {
			return fmt.Errorf("rule clause %q is empty", clause.ClauseID)
		}
		clauseCounts[clause.ReferenceID]++
	}
	for _, entry := range entries {
		if requiresClauses(entry.Kind) && clauseCounts[entry.ID] == 0 {
			return fmt.Errorf("reference %q (%s) has no generated clauses", entry.ID, entry.Kind)
		}
	}
	if err := validateEpicCatalog(epics, matrix.Rows); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(outDir, "remediation_backlog.md")); err != nil {
		return fmt.Errorf("read remediation_backlog.md: %w", err)
	}
	return nil
}

func buildSummary(referenceRoot string, rows []auditMatrixRow, clauses []ruleClause) generatedSummary {
	kindCounts := map[string]int{}
	auditAreaCounts := map[string]int{}
	normativityCounts := map[string]int{}
	reviewStateCounts := map[string]int{}
	finalStatusCounts := map[string]int{}
	for _, row := range rows {
		kindCounts[row.Kind]++
		auditAreaCounts[row.AuditArea]++
		normativityCounts[row.Normativity]++
		reviewStateCounts[row.ReviewState]++
		finalStatusCounts[summaryBucket(row.FinalStatus)]++
	}
	return generatedSummary{
		Version:             inventoryVersion,
		GeneratedAt:         time.Now().UTC().Format(time.RFC3339),
		ReferenceRoot:       referenceRoot,
		EntryCount:          len(rows),
		RuleClauseCount:     len(clauses),
		CountsByKind:        countsToSummary(kindCounts),
		CountsByAuditArea:   countsToSummary(auditAreaCounts),
		CountsByNormativity: countsToSummary(normativityCounts),
		CountsByReviewState: countsToSummary(reviewStateCounts),
		CountsByFinalStatus: countsToSummary(finalStatusCounts),
	}
}

func countsToSummary(values map[string]int) []summaryItem {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]summaryItem, 0, len(keys))
	for _, key := range keys {
		items = append(items, summaryItem{Key: key, Count: values[key]})
	}
	return items
}

func buildAuditMatrix(entries []corpusIndexEntry) []auditMatrixRow {
	rows := make([]auditMatrixRow, 0, len(entries))
	for _, entry := range entries {
		row := auditMatrixRow{
			ReferenceID:          entry.ID,
			Title:                strings.TrimSpace(entry.Title),
			Kind:                 strings.TrimSpace(entry.Kind),
			Path:                 strings.TrimSpace(entry.Path),
			Aliases:              append([]string(nil), entry.Aliases...),
			AuditArea:            deriveAuditArea(entry),
			Normativity:          deriveNormativity(entry),
			ReviewState:          "pending",
			RepoMappings:         buildRepoMappings(entry),
			SurfaceApplicability: buildSurfaceApplicability(entry),
			SemanticMatch:        "unknown",
		}
		applyCuratedAssessment(&row)
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].ReferenceID < rows[j].ReferenceID
	})
	return rows
}

func deriveAuditArea(entry corpusIndexEntry) string {
	switch strings.TrimSpace(entry.Kind) {
	case "class", "subclass", "ancestry", "community":
		return "character_creation"
	case "weapon", "armor", "item", "consumable":
		return "equipment_items"
	case "domain", "ability":
		return "domain_cards"
	case "adversary":
		return "adversaries"
	case "environment":
		return "environments"
	case "beastform":
		return "companions_beastforms"
	}

	switch strings.TrimSpace(entry.ID) {
	case "character-creation":
		return "character_creation"
	case "core-materials", "core-mechanics", "playbook-session-flow":
		return "multi_area"
	case "running-an-adventure", "glossary-fear", "playbook-gm-fear-and-moves":
		return "gm_fear_moves"
	case "glossary-stress", "glossary-hope", "glossary-conditions":
		return "character_model"
	case "glossary-downtime", "playbook-rests-and-downtime":
		return "recovery_downtime"
	case "glossary-death":
		return "death_scars"
	case "glossary-session-zero", "playbook-witherwild-session-zero", "introduction":
		return "editorial_reference"
	default:
		return "editorial_reference"
	}
}

func deriveNormativity(entry corpusIndexEntry) string {
	switch strings.TrimSpace(entry.Kind) {
	case "rule":
		return "normative_rule"
	case "playbook", "glossary":
		return "supporting_guidance"
	default:
		return "content_instance"
	}
}

func buildRepoMappings(entry corpusIndexEntry) []repoMapping {
	kind := strings.TrimSpace(entry.Kind)
	auditArea := deriveAuditArea(entry)
	var mappings []repoMapping

	switch kind {
	case "class", "subclass", "ancestry", "community", "domain", "weapon", "armor", "item", "adversary", "environment", "beastform":
		mappings = append(mappings,
			repoMapping{
				Surface: "content",
				Paths: []string{
					"api/proto/systems/daggerheart/v1/content.proto",
					"internal/services/game/domain/systems/daggerheart/contentstore/contracts.go",
					"internal/services/game/api/grpc/systems/daggerheart/contenttransport/",
					"internal/services/game/storage/sqlite/daggerheartcontent/",
					"internal/tools/importer/content/daggerheart/v1/",
				},
			},
			repoMapping{
				Surface: "tests",
				Paths: []string{
					"internal/services/game/api/grpc/systems/daggerheart/contenttransport/*_test.go",
					"internal/services/game/storage/sqlite/daggerheartcontent/*_test.go",
					"internal/tools/importer/content/daggerheart/v1/*_test.go",
				},
			},
		)
	case "consumable":
		mappings = append(mappings,
			repoMapping{
				Surface: "content",
				Paths: []string{
					"internal/tools/importer/content/daggerheart/v1/",
					"internal/services/game/storage/sqlite/daggerheartcontent/",
				},
				Notes: "Consumables are modeled through item and inventory surfaces rather than a dedicated reference kind.",
			},
			repoMapping{
				Surface: "grpc_system",
				Paths: []string{
					"internal/services/game/api/grpc/systems/daggerheart/charactermutationtransport/",
				},
			},
		)
	case "ability":
		mappings = append(mappings,
			repoMapping{
				Surface: "content",
				Paths: []string{
					"api/proto/systems/daggerheart/v1/content.proto",
					"internal/services/game/domain/systems/daggerheart/contentstore/contracts.go",
				},
				Notes: "Ability semantics may resolve through domain cards, class features, subclass features, or genericized system behavior.",
			},
			repoMapping{
				Surface: "domain_module",
				Paths: []string{
					"internal/services/game/domain/systems/daggerheart/",
					"internal/services/game/api/grpc/systems/daggerheart/",
				},
			},
		)
	}

	switch auditArea {
	case "character_creation":
		mappings = append(mappings,
			repoMapping{
				Surface: "domain_module",
				Paths: []string{
					"internal/services/game/domain/systems/daggerheart/creation_workflow.go",
					"internal/services/game/api/grpc/systems/daggerheart/creationworkflow/",
				},
			},
			repoMapping{
				Surface: "web",
				Paths: []string{
					"internal/services/web/modules/campaigns/workflow/daggerheart/",
					"internal/services/web/modules/campaigns/render/character_creation.templ",
					"internal/services/web/modules/campaigns/gateway/grpc_creation_*.go",
				},
			},
			repoMapping{
				Surface: "grpc_core",
				Paths: []string{
					"internal/services/game/api/grpc/game/charactertransport/character_workflow*.go",
					"api/proto/game/v1/character.proto",
					"api/proto/systems/daggerheart/v1/state.proto",
				},
			},
		)
	case "character_model", "death_scars", "recovery_downtime", "gm_fear_moves", "resolution_rolls", "progression", "multi_area":
		mappings = append(mappings,
			repoMapping{
				Surface: "domain_module",
				Paths: []string{
					"internal/services/game/domain/systems/daggerheart/",
				},
			},
			repoMapping{
				Surface: "grpc_system",
				Paths: []string{
					"internal/services/game/api/grpc/systems/daggerheart/",
				},
			},
			repoMapping{
				Surface: "mcp",
				Paths: []string{
					"internal/services/mcp/domain/",
				},
			},
		)
	case "adversaries", "environments", "domain_cards", "equipment_items", "companions_beastforms":
		mappings = append(mappings,
			repoMapping{
				Surface: "domain_module",
				Paths: []string{
					"internal/services/game/domain/systems/daggerheart/",
				},
			},
			repoMapping{
				Surface: "grpc_system",
				Paths: []string{
					"internal/services/game/api/grpc/systems/daggerheart/",
				},
			},
			repoMapping{
				Surface: "web",
				Paths: []string{
					"internal/services/web/modules/campaigns/",
				},
			},
		)
	}

	mappings = append(mappings,
		repoMapping{
			Surface: "docs",
			Paths: []string{
				"docs/product/daggerheart-PRD.md",
				"docs/reference/daggerheart-creation-workflow.md",
				"docs/reference/daggerheart-event-timeline-contract.md",
			},
			Notes: "Docs are advisory evidence and can themselves be audit findings.",
		},
	)

	return dedupeMappings(mappings)
}

func buildSurfaceApplicability(entry corpusIndexEntry) []surfaceApplicability {
	kind := strings.TrimSpace(entry.Kind)
	auditArea := deriveAuditArea(entry)
	apply := func(surface, state, notes string) surfaceApplicability {
		return surfaceApplicability{Surface: surface, State: state, Notes: notes}
	}

	surfaces := []surfaceApplicability{
		apply("domain_module", "expected", ""),
		apply("projection_adapter", "conditional", ""),
		apply("grpc_system", "expected", ""),
		apply("grpc_core", "conditional", ""),
		apply("web", "conditional", ""),
		apply("mcp", "conditional", ""),
		apply("importer_content", "conditional", ""),
		apply("docs", "expected", ""),
		apply("tests", "expected", ""),
	}

	switch kind {
	case "class", "subclass", "ancestry", "community", "domain", "weapon", "armor", "item", "adversary", "environment", "beastform", "consumable":
		return []surfaceApplicability{
			apply("domain_module", "conditional", "Becomes expected when the content entry implies mechanics or stateful rules."),
			apply("projection_adapter", "conditional", ""),
			apply("grpc_system", "expected", "Content endpoints and item-specific runtime surfaces are in scope."),
			apply("grpc_core", "conditional", ""),
			apply("web", "conditional", ""),
			apply("mcp", "conditional", ""),
			apply("importer_content", "expected", ""),
			apply("docs", "expected", ""),
			apply("tests", "expected", ""),
		}
	case "ability":
		return []surfaceApplicability{
			apply("domain_module", "expected", "Abilities can map to domain cards, class features, subclass features, or genericized mechanics."),
			apply("projection_adapter", "conditional", ""),
			apply("grpc_system", "expected", ""),
			apply("grpc_core", "conditional", ""),
			apply("web", "conditional", ""),
			apply("mcp", "conditional", ""),
			apply("importer_content", "supporting", "No direct ability catalog exists today; mappings may resolve through other content kinds."),
			apply("docs", "expected", ""),
			apply("tests", "expected", ""),
		}
	}

	switch auditArea {
	case "editorial_reference":
		return []surfaceApplicability{
			apply("domain_module", "conditional", ""),
			apply("projection_adapter", "n/a", ""),
			apply("grpc_system", "conditional", ""),
			apply("grpc_core", "conditional", ""),
			apply("web", "conditional", ""),
			apply("mcp", "conditional", ""),
			apply("importer_content", "n/a", ""),
			apply("docs", "expected", ""),
			apply("tests", "conditional", ""),
		}
	case "character_creation":
		return []surfaceApplicability{
			apply("domain_module", "expected", ""),
			apply("projection_adapter", "expected", ""),
			apply("grpc_system", "expected", ""),
			apply("grpc_core", "expected", ""),
			apply("web", "expected", ""),
			apply("mcp", "expected", ""),
			apply("importer_content", "expected", ""),
			apply("docs", "expected", ""),
			apply("tests", "expected", ""),
		}
	}

	return surfaces
}

func dedupeMappings(mappings []repoMapping) []repoMapping {
	if len(mappings) == 0 {
		return nil
	}
	type key struct {
		surface string
		paths   string
		notes   string
	}
	seen := map[key]struct{}{}
	result := make([]repoMapping, 0, len(mappings))
	for _, mapping := range mappings {
		sort.Strings(mapping.Paths)
		k := key{
			surface: mapping.Surface,
			paths:   strings.Join(mapping.Paths, "\n"),
			notes:   mapping.Notes,
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		result = append(result, mapping)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Surface == result[j].Surface {
			return strings.Join(result[i].Paths, ",") < strings.Join(result[j].Paths, ",")
		}
		return result[i].Surface < result[j].Surface
	})
	return result
}

func buildRuleClauses(referenceRoot string, entries []corpusIndexEntry) ([]ruleClause, error) {
	clauses := []ruleClause{}
	for _, entry := range entries {
		if !requiresClauses(entry.Kind) {
			continue
		}
		path := filepath.Join(referenceRoot, filepath.FromSlash(entry.Path))
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Path, err)
		}
		extracted, err := extractClauses(entry, string(content))
		if err != nil {
			return nil, fmt.Errorf("extract clauses for %s: %w", entry.Path, err)
		}
		clauses = append(clauses, extracted...)
	}
	sort.Slice(clauses, func(i, j int) bool {
		if clauses[i].ReferenceID == clauses[j].ReferenceID {
			return clauses[i].ClauseID < clauses[j].ClauseID
		}
		return clauses[i].ReferenceID < clauses[j].ReferenceID
	})
	return clauses, nil
}

func extractClauses(entry corpusIndexEntry, content string) ([]ruleClause, error) {
	sections, err := parseMarkdownSections(content)
	if err != nil {
		return nil, err
	}
	switch strings.TrimSpace(entry.Kind) {
	case "rule", "glossary":
		target := "Normalized Source Text"
		if entry.Kind == "glossary" {
			target = "Governing Rule Text"
		}
		text, ok := sections[target]
		if !ok {
			return nil, fmt.Errorf("missing %q section", target)
		}
		items := splitNormativeText(text)
		return buildClauseList(entry, target, items), nil
	case "playbook":
		items := []ruleClause{}
		sectionNames := sortedSectionNames(sections)
		clauseCounter := 1
		for _, section := range sectionNames {
			if section == "Query Use" {
				continue
			}
			text := sections[section]
			parts := splitParagraphs(text)
			for _, part := range parts {
				items = append(items, ruleClause{
					ClauseID:    fmt.Sprintf("%s-%03d", entry.ID, clauseCounter),
					ReferenceID: entry.ID,
					Kind:        entry.Kind,
					Path:        entry.Path,
					AuditArea:   deriveAuditArea(entry),
					Section:     section,
					Text:        part,
				})
				clauseCounter++
			}
		}
		if len(items) == 0 {
			return nil, errors.New("no playbook clauses found")
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported clause extraction kind %q", entry.Kind)
	}
}

func buildClauseList(entry corpusIndexEntry, section string, parts []string) []ruleClause {
	clauses := make([]ruleClause, 0, len(parts))
	for i, part := range parts {
		clauses = append(clauses, ruleClause{
			ClauseID:    fmt.Sprintf("%s-%03d", entry.ID, i+1),
			ReferenceID: entry.ID,
			Kind:        entry.Kind,
			Path:        entry.Path,
			AuditArea:   deriveAuditArea(entry),
			Section:     section,
			Text:        part,
		})
	}
	return clauses
}

func splitNormativeText(text string) []string {
	normalized := normalizeText(text)
	if normalized == "" {
		return nil
	}

	stepSegments := splitByPattern(normalized, reStepSplit)
	if len(stepSegments) > 1 {
		return compactSegments(stepSegments)
	}

	bulletSegments := reBulletSplit.Split(normalized, -1)
	if len(bulletSegments) > 1 {
		return compactSegments(bulletSegments)
	}

	sentences := splitSentences(normalized)
	if len(sentences) <= 3 {
		return []string{normalized}
	}

	parts := []string{}
	chunk := make([]string, 0, 3)
	for _, sentence := range sentences {
		chunk = append(chunk, sentence)
		if len(chunk) == 3 {
			parts = append(parts, strings.Join(chunk, " "))
			chunk = chunk[:0]
		}
	}
	if len(chunk) > 0 {
		parts = append(parts, strings.Join(chunk, " "))
	}
	return compactSegments(parts)
}

func splitByPattern(input string, pattern *regexp.Regexp) []string {
	indexes := pattern.FindAllStringIndex(input, -1)
	if len(indexes) == 0 {
		return []string{input}
	}
	result := make([]string, 0, len(indexes))
	for i, idx := range indexes {
		start := idx[0]
		end := len(input)
		if i+1 < len(indexes) {
			end = indexes[i+1][0]
		}
		segment := strings.TrimSpace(input[start:end])
		if segment != "" {
			result = append(result, segment)
		}
	}
	prefix := strings.TrimSpace(input[:indexes[0][0]])
	if prefix != "" {
		result = append([]string{prefix}, result...)
	}
	return result
}

func splitSentences(input string) []string {
	if strings.TrimSpace(input) == "" {
		return nil
	}
	var sentences []string
	var buf strings.Builder
	for _, r := range input {
		buf.WriteRune(r)
		if r == '.' || r == '!' || r == '?' {
			sentence := normalizeText(buf.String())
			if sentence != "" {
				sentences = append(sentences, sentence)
			}
			buf.Reset()
		}
	}
	tail := normalizeText(buf.String())
	if tail != "" {
		sentences = append(sentences, tail)
	}
	if len(sentences) == 0 {
		return []string{normalizeText(input)}
	}
	return sentences
}

func compactSegments(parts []string) []string {
	compacted := make([]string, 0, len(parts))
	for _, part := range parts {
		part = normalizeText(part)
		if part == "" {
			continue
		}
		compacted = append(compacted, part)
	}
	return compacted
}

func parseMarkdownSections(content string) (map[string]string, error) {
	lines := strings.Split(content, "\n")
	sections := map[string]string{}
	var current string
	var buf bytes.Buffer
	inFrontMatter := false
	frontMatterSeen := 0

	flush := func() {
		if current == "" {
			buf.Reset()
			return
		}
		sections[current] = strings.TrimSpace(buf.String())
		buf.Reset()
	}

	for _, line := range lines {
		if frontMatterSeen < 2 && reFrontMatterSep.MatchString(strings.TrimSpace(line)) {
			inFrontMatter = !inFrontMatter
			frontMatterSeen++
			continue
		}
		if inFrontMatter {
			continue
		}
		matches := reHeading.FindStringSubmatch(line)
		if matches != nil && len(matches[1]) == 2 {
			flush()
			current = strings.TrimSpace(matches[2])
			continue
		}
		if current == "" {
			continue
		}
		buf.WriteString(line)
		buf.WriteString("\n")
	}
	flush()
	if len(sections) == 0 {
		return nil, errors.New("no sections found")
	}
	return sections, nil
}

func splitParagraphs(text string) []string {
	raw := strings.Split(text, "\n")
	parts := []string{}
	var buf []string
	flush := func() {
		if len(buf) == 0 {
			return
		}
		part := normalizeText(strings.Join(buf, " "))
		if part != "" {
			parts = append(parts, part)
		}
		buf = buf[:0]
	}
	for _, line := range raw {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			flush()
			continue
		}
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			flush()
			parts = append(parts, normalizeText(strings.TrimSpace(trimmed[2:])))
			continue
		}
		buf = append(buf, trimmed)
	}
	flush()
	return parts
}

func sortedSectionNames(sections map[string]string) []string {
	names := make([]string, 0, len(sections))
	for name := range sections {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func normalizeText(text string) string {
	return strings.TrimSpace(reWhitespace.ReplaceAllString(text, " "))
}

func loadIndexEntries(referenceRoot string) ([]corpusIndexEntry, error) {
	indexPath := filepath.Join(referenceRoot, "index.json")
	file, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("read index %s: %w", indexPath, err)
	}
	var entries []corpusIndexEntry
	if err := json.Unmarshal(file, &entries); err != nil {
		return nil, fmt.Errorf("decode index %s: %w", indexPath, err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ID < entries[j].ID
	})
	return entries, nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func writeText(path string, content string) error {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func loadJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func requiresClauses(kind string) bool {
	switch strings.TrimSpace(kind) {
	case "rule", "playbook", "glossary":
		return true
	default:
		return false
	}
}

func isAllowedNameStrategy(value string) bool {
	switch value {
	case "canonical", "ip_safe_reword", "genericized", "not_applicable":
		return true
	default:
		return false
	}
}

func defaultReferenceRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "reference-corpus/v1/reference"
	}
	return filepath.Join(home, "code", "daggerheart", "reference-corpus", "v1", "reference")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
