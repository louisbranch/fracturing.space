package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildAuditMatrixSeedsExpectedFields(t *testing.T) {
	rows := buildAuditMatrix([]corpusIndexEntry{
		{
			ID:    "character-creation",
			Title: "Character Creation",
			Kind:  "rule",
			Path:  "rules/character-creation.md",
		},
		{
			ID:    "subclass-beastbound",
			Title: "Beastbound",
			Kind:  "subclass",
			Path:  "subclasses/beastbound.md",
		},
		{
			ID:    "ability-rune-ward",
			Title: "Rune Ward",
			Kind:  "ability",
			Path:  "abilities/rune-ward.md",
		},
	})

	if len(rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3", len(rows))
	}

	if rows[0].ReferenceID != "ability-rune-ward" {
		t.Fatalf("rows sorted unexpectedly: first id = %q", rows[0].ReferenceID)
	}
	if rows[0].AuditArea != "domain_cards" {
		t.Fatalf("ability audit area = %q, want domain_cards", rows[0].AuditArea)
	}
	if rows[0].ReviewState != "reviewed" {
		t.Fatalf("ability review_state = %q, want reviewed", rows[0].ReviewState)
	}
	if rows[0].SemanticMatch != "ambiguous" {
		t.Fatalf("ability semantic match = %q, want ambiguous", rows[0].SemanticMatch)
	}
	if rows[0].FinalStatus != "gap" {
		t.Fatalf("ability final_status = %q, want gap", rows[0].FinalStatus)
	}
	if len(rows[0].RepoMappings) == 0 {
		t.Fatal("ability row missing repo mappings")
	}

	var characterRow auditMatrixRow
	for _, row := range rows {
		if row.ReferenceID == "character-creation" {
			characterRow = row
			break
		}
	}
	if characterRow.ReferenceID == "" {
		t.Fatal("character creation row not found")
	}
	if characterRow.AuditArea != "character_creation" {
		t.Fatalf("character creation audit area = %q, want character_creation", characterRow.AuditArea)
	}
	if characterRow.Normativity != "normative_rule" {
		t.Fatalf("character creation normativity = %q, want normative_rule", characterRow.Normativity)
	}
	if characterRow.ReviewState != "reviewed" {
		t.Fatalf("character creation review_state = %q, want reviewed", characterRow.ReviewState)
	}
	if characterRow.FinalStatus != "covered" {
		t.Fatalf("character creation final_status = %q, want covered", characterRow.FinalStatus)
	}
	if characterRow.FollowUpEpic != "" {
		t.Fatalf("character creation follow_up_epic = %q, want empty", characterRow.FollowUpEpic)
	}
	if len(characterRow.SurfaceApplicability) == 0 {
		t.Fatal("character creation row missing surface applicability")
	}

	var subclassRow auditMatrixRow
	for _, row := range rows {
		if row.ReferenceID == "subclass-beastbound" {
			subclassRow = row
			break
		}
	}
	if subclassRow.ReferenceID == "" {
		t.Fatal("subclass row not found")
	}
	if subclassRow.FinalStatus != "covered" {
		t.Fatalf("subclass final_status = %q, want covered", subclassRow.FinalStatus)
	}
}

func TestValidateAuditRowRejectsReviewedGapWithoutEpic(t *testing.T) {
	row := auditMatrixRow{
		ReferenceID:          "glossary-fear",
		Title:                "Fear",
		Kind:                 "glossary",
		Path:                 "glossary/glossary-fear.md",
		AuditArea:            "gm_fear_moves",
		Normativity:          "supporting_guidance",
		ReviewState:          "reviewed",
		RepoMappings:         []repoMapping{{Surface: "docs", Paths: []string{"docs/product/daggerheart-PRD.md"}}},
		SurfaceApplicability: []surfaceApplicability{{Surface: "docs", State: "expected"}},
		NameStrategy:         "canonical",
		SemanticMatch:        "partial",
		FinalStatus:          "gap",
		GapClass:             "behavior",
		Notes:                []string{"fear initialization is still wrong"},
	}

	err := validateAuditRow(row, corpusIndexEntry{
		ID:    "glossary-fear",
		Title: "Fear",
		Kind:  "glossary",
		Path:  "glossary/glossary-fear.md",
	}, false)
	if err == nil {
		t.Fatal("expected reviewed gap without follow_up_epic to fail validation")
	}
}

func TestBuildEpicCatalogAggregatesGapRows(t *testing.T) {
	rows := []auditMatrixRow{
		{
			ReferenceID:  "character-creation",
			Kind:         "rule",
			AuditArea:    "character_creation",
			FinalStatus:  "gap",
			GapClass:     "missing_model",
			FollowUpEpic: "heritage-and-companion-modeling",
			EvidenceCode: []string{"internal/services/game/domain/systems/daggerheart/creation_workflow.go"},
		},
		{
			ReferenceID:  "ability-brace",
			Kind:         "ability",
			AuditArea:    "domain_cards",
			FinalStatus:  "gap",
			GapClass:     "ambiguous_mapping",
			FollowUpEpic: "ability-mapping-and-semantic-audit",
			EvidenceDocs: []string{"docs/product/daggerheart-PRD.md"},
		},
		{
			ReferenceID: "weapon-longsword",
			Kind:        "weapon",
			AuditArea:   "equipment_items",
			FinalStatus: "covered",
		},
	}

	catalog, markdown, err := buildEpicCatalog("reference-root", rows)
	if err != nil {
		t.Fatalf("buildEpicCatalog returned error: %v", err)
	}
	if catalog.GapRowCount != 2 {
		t.Fatalf("GapRowCount = %d, want 2", catalog.GapRowCount)
	}
	if catalog.EpicCount != 2 {
		t.Fatalf("EpicCount = %d, want 2", catalog.EpicCount)
	}
	if markdown == "" {
		t.Fatal("expected non-empty backlog markdown")
	}
	if err := validateEpicCatalog(catalog, rows); err != nil {
		t.Fatalf("validateEpicCatalog returned error: %v", err)
	}
}

func TestExtractClausesFromRuleAndPlaybook(t *testing.T) {
	rulePath := filepath.Join("testdata", "rules", "character-creation.md")
	ruleContent, err := os.ReadFile(rulePath)
	if err != nil {
		t.Fatalf("read rule fixture: %v", err)
	}
	ruleClauses, err := extractClauses(corpusIndexEntry{
		ID:    "character-creation",
		Kind:  "rule",
		Path:  "rules/character-creation.md",
		Title: "Character Creation",
	}, string(ruleContent))
	if err != nil {
		t.Fatalf("extract rule clauses: %v", err)
	}
	if len(ruleClauses) < 3 {
		t.Fatalf("len(ruleClauses) = %d, want at least 3", len(ruleClauses))
	}
	if ruleClauses[1].Section != "Normalized Source Text" {
		t.Fatalf("rule section = %q, want Normalized Source Text", ruleClauses[1].Section)
	}

	playbookPath := filepath.Join("testdata", "playbooks", "playbook-rests-and-downtime.md")
	playbookContent, err := os.ReadFile(playbookPath)
	if err != nil {
		t.Fatalf("read playbook fixture: %v", err)
	}
	playbookClauses, err := extractClauses(corpusIndexEntry{
		ID:    "playbook-rests-and-downtime",
		Kind:  "playbook",
		Path:  "playbooks/playbook-rests-and-downtime.md",
		Title: "Rests and Downtime",
	}, string(playbookContent))
	if err != nil {
		t.Fatalf("extract playbook clauses: %v", err)
	}
	if len(playbookClauses) != 4 {
		t.Fatalf("len(playbookClauses) = %d, want 4", len(playbookClauses))
	}
	if playbookClauses[0].Section != "GM Consequences" && playbookClauses[0].Section != "Shared Rules" {
		t.Fatalf("unexpected playbook section ordering: first section = %q", playbookClauses[0].Section)
	}
}

func TestCheckRejectsMissingClauseCoverage(t *testing.T) {
	dir := t.TempDir()
	referenceRoot := filepath.Join(dir, "reference")
	if err := os.MkdirAll(referenceRoot, 0o755); err != nil {
		t.Fatalf("mkdir reference root: %v", err)
	}
	indexJSON := `[
  {"id":"glossary-conditions","title":"Conditions","kind":"glossary","path":"glossary/glossary-conditions.md","aliases":[]}
]`
	if err := os.WriteFile(filepath.Join(referenceRoot, "index.json"), []byte(indexJSON), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir out dir: %v", err)
	}
	inventory := generatedInventory{
		Version:       inventoryVersion,
		GeneratedAt:   "2026-03-13T00:00:00Z",
		ReferenceRoot: referenceRoot,
		EntryCount:    1,
		Entries: []corpusIndexEntry{
			{ID: "glossary-conditions", Title: "Conditions", Kind: "glossary", Path: "glossary/glossary-conditions.md"},
		},
	}
	matrix := generatedAuditMatrix{
		Version:       inventoryVersion,
		GeneratedAt:   "2026-03-13T00:00:00Z",
		ReferenceRoot: referenceRoot,
		RowCount:      1,
		Rows: []auditMatrixRow{
			{
				ReferenceID:          "glossary-conditions",
				Title:                "Conditions",
				Kind:                 "glossary",
				Path:                 "glossary/glossary-conditions.md",
				AuditArea:            "character_model",
				Normativity:          "supporting_guidance",
				ReviewState:          "pending",
				RepoMappings:         []repoMapping{{Surface: "docs", Paths: []string{"docs/product/daggerheart-PRD.md"}}},
				SurfaceApplicability: []surfaceApplicability{{Surface: "docs", State: "expected"}},
				SemanticMatch:        "unknown",
			},
		},
	}
	clauses := generatedRuleClauses{
		Version:       inventoryVersion,
		GeneratedAt:   "2026-03-13T00:00:00Z",
		ReferenceRoot: referenceRoot,
		ClauseCount:   0,
		Clauses:       nil,
	}
	if err := writeJSON(filepath.Join(outDir, "inventory.json"), inventory); err != nil {
		t.Fatalf("write inventory: %v", err)
	}
	if err := writeJSON(filepath.Join(outDir, "audit_matrix.json"), matrix); err != nil {
		t.Fatalf("write audit matrix: %v", err)
	}
	if err := writeJSON(filepath.Join(outDir, "rule_clauses.json"), clauses); err != nil {
		t.Fatalf("write clauses: %v", err)
	}

	if err := runCheck([]string{"-root", dir, "-reference-root", referenceRoot, "-out-dir", outDir}); err == nil {
		t.Fatal("expected runCheck to fail when glossary clause coverage is missing")
	}
}
