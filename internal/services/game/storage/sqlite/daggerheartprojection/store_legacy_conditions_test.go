package daggerheartprojection_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteconn"
	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	sqlitecoreprojection "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/coreprojection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/migrations"
)

func TestOpenProjectionsRepairsLegacyConditionStateEncoding(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.sqlite")
	seedLegacyProjectionDB(t, path)

	store, err := sqlitecoreprojection.Open(path)
	if err != nil {
		t.Fatalf("open projections store: %v", err)
	}

	daggerheartStore := store.ProjectionStores().Daggerheart
	if daggerheartStore == nil {
		t.Fatal("expected daggerheart projection backend")
	}

	state, err := daggerheartStore.GetDaggerheartCharacterState(context.Background(), "camp-legacy", "char-legacy")
	if err != nil {
		t.Fatalf("get repaired character state: %v", err)
	}
	if len(state.Conditions) != 1 || state.Conditions[0].Code != "hidden" || state.Conditions[0].Class != "standard" {
		t.Fatalf("expected repaired hidden condition, got %v", state.Conditions)
	}

	adversary, err := daggerheartStore.GetDaggerheartAdversary(context.Background(), "camp-legacy", "adv-legacy")
	if err != nil {
		t.Fatalf("get repaired adversary: %v", err)
	}
	if len(adversary.Conditions) != 1 || adversary.Conditions[0].Code != "restrained" || adversary.Conditions[0].Class != "standard" {
		t.Fatalf("expected repaired restrained condition, got %v", adversary.Conditions)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close projections store: %v", err)
	}

	rawDB, err := sqliteconn.Open(path)
	if err != nil {
		t.Fatalf("reopen raw sqlite db: %v", err)
	}
	defer func() {
		if err := rawDB.Close(); err != nil {
			t.Fatalf("close raw sqlite db: %v", err)
		}
	}()

	assertStructuredConditionsJSON(t, rawDB, `
SELECT conditions_json
FROM daggerheart_character_states
WHERE campaign_id = ? AND character_id = ?
`, "camp-legacy", "char-legacy")
	assertStructuredConditionsJSON(t, rawDB, `
SELECT conditions_json
FROM daggerheart_adversaries
WHERE campaign_id = ? AND adversary_id = ?
`, "camp-legacy", "adv-legacy")
}

func seedLegacyProjectionDB(t *testing.T, path string) {
	t.Helper()

	sqlDB, err := sqliteconn.Open(path)
	if err != nil {
		t.Fatalf("open legacy sqlite db: %v", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			t.Fatalf("close legacy sqlite db: %v", err)
		}
	}()

	content, err := fs.ReadFile(migrations.ProjectionsFS, "projections/001_projections.sql")
	if err != nil {
		t.Fatalf("read baseline projections migration: %v", err)
	}
	if _, err := sqlDB.Exec(sqlitemigrate.ExtractUpMigration(string(content))); err != nil {
		t.Fatalf("apply baseline projections migration: %v", err)
	}

	now := time.Date(2026, 2, 3, 11, 0, 0, 0, time.UTC).UnixMilli()
	if _, err := sqlDB.Exec(`
INSERT INTO campaigns (id, name, locale, game_system, status, gm_mode, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`, "camp-legacy", "Legacy Campaign", "en-US", "daggerheart", "ACTIVE", "HUMAN", now, now); err != nil {
		t.Fatalf("insert legacy campaign: %v", err)
	}
	if _, err := sqlDB.Exec(`
INSERT INTO characters (campaign_id, id, name, kind, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
`, "camp-legacy", "char-legacy", "Brim", "PC", now, now); err != nil {
		t.Fatalf("insert legacy character: %v", err)
	}
	if _, err := sqlDB.Exec(`
INSERT INTO daggerheart_character_states (
	campaign_id, character_id, hp, hope, hope_max, stress, armor, conditions_json,
	temporary_armor_json, life_state, class_state_json, subclass_state_json,
	companion_state_json, impenetrable_used_this_short_rest, stat_modifiers_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, "camp-legacy", "char-legacy", 10, 2, 6, 0, 0, `["hidden"]`, `[]`, "alive", `{}`, `{}`, `{}`, 0, `[]`); err != nil {
		t.Fatalf("insert legacy character state: %v", err)
	}
	if _, err := sqlDB.Exec(`
INSERT INTO daggerheart_adversaries (
	campaign_id, adversary_id, adversary_entry_id, name, kind, session_id, scene_id, notes,
	hp, hp_max, stress, stress_max, evasion, major_threshold, severe_threshold, armor,
	conditions_json, feature_state_json, pending_experience_json, spotlight_gate_id,
	spotlight_count, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, "camp-legacy", "adv-legacy", "entry-1", "Shade", "solo", "sess-1", "", "", 6, 6, 0, 6, 10, 8, 12, 0, `["restrained"]`, `[]`, "", "", 0, now, now); err != nil {
		t.Fatalf("insert legacy adversary: %v", err)
	}
}

func assertStructuredConditionsJSON(t *testing.T, sqlDB *sql.DB, query string, args ...any) {
	t.Helper()

	var raw string
	if err := sqlDB.QueryRow(query, args...).Scan(&raw); err != nil {
		t.Fatalf("query repaired conditions json: %v", err)
	}

	var structured []map[string]any
	if err := json.Unmarshal([]byte(raw), &structured); err != nil {
		t.Fatalf("unmarshal repaired conditions json: %v", err)
	}
	if len(structured) != 1 {
		t.Fatalf("expected 1 repaired condition row, got %d", len(structured))
	}
	if structured[0]["Code"] == "" || structured[0]["Class"] != "standard" {
		t.Fatalf("expected structured repaired condition row, got %+v", structured[0])
	}
}
