package sqlite

import (
	"database/sql"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteconn"
	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqlitemigrate"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/migrations"
)

func TestEventsMigration002_AddsColumnsWithoutDataLoss(t *testing.T) {
	sqlDB := openMigrationTestDB(t)

	_, err := sqlDB.Exec(`
CREATE TABLE events (
    campaign_id TEXT NOT NULL,
    seq INTEGER NOT NULL,
    event_hash TEXT NOT NULL,
    prev_event_hash TEXT NOT NULL DEFAULT '',
    chain_hash TEXT NOT NULL,
    signature_key_id TEXT NOT NULL,
    event_signature TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    event_type TEXT NOT NULL,
    system_id TEXT NOT NULL DEFAULT '',
    system_version TEXT NOT NULL DEFAULT '',
    session_id TEXT NOT NULL DEFAULT '',
    request_id TEXT NOT NULL DEFAULT '',
    invocation_id TEXT NOT NULL DEFAULT '',
    actor_type TEXT NOT NULL,
    actor_id TEXT NOT NULL DEFAULT '',
    entity_type TEXT NOT NULL DEFAULT '',
    entity_id TEXT NOT NULL DEFAULT '',
    payload_json BLOB NOT NULL,
    PRIMARY KEY (campaign_id, seq)
);`)
	if err != nil {
		t.Fatalf("create legacy events table: %v", err)
	}

	_, err = sqlDB.Exec(`
INSERT INTO events (
    campaign_id, seq, event_hash, prev_event_hash, chain_hash,
    signature_key_id, event_signature, timestamp, event_type,
    system_id, system_version, session_id, request_id, invocation_id,
    actor_type, actor_id, entity_type, entity_id, payload_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"camp-1", 1, "hash-1", "", "chain-1",
		"key-1", "sig-1", 1735689600000, "campaign.created",
		"", "", "", "req-1", "inv-1",
		"system", "", "", "", []byte(`{"ok":true}`),
	)
	if err != nil {
		t.Fatalf("insert legacy event row: %v", err)
	}

	markMigrationsAppliedExcept(t, sqlDB, migrations.EventsFS, "events", "002_events_v2.sql")
	applyMigrationSet(t, sqlDB, migrations.EventsFS, "events")

	var sceneID, correlationID, causationID string
	err = sqlDB.QueryRow(`
SELECT scene_id, correlation_id, causation_id
FROM events
WHERE campaign_id = ? AND seq = ?`,
		"camp-1", 1,
	).Scan(&sceneID, &correlationID, &causationID)
	if err != nil {
		t.Fatalf("query migrated event row: %v", err)
	}
	if sceneID != "" || correlationID != "" || causationID != "" {
		t.Fatalf("expected default empty new columns, got scene=%q correlation=%q causation=%q", sceneID, correlationID, causationID)
	}
}

func TestProjectionMigration002_AddsConditionsWithoutDataLoss(t *testing.T) {
	sqlDB := openMigrationTestDB(t)

	_, err := sqlDB.Exec(`
CREATE TABLE daggerheart_adversaries (
    campaign_id TEXT NOT NULL,
    adversary_id TEXT NOT NULL,
    name TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT '',
    session_id TEXT,
    notes TEXT NOT NULL DEFAULT '',
    hp INTEGER NOT NULL DEFAULT 6,
    hp_max INTEGER NOT NULL DEFAULT 6,
    stress INTEGER NOT NULL DEFAULT 0,
    stress_max INTEGER NOT NULL DEFAULT 6,
    evasion INTEGER NOT NULL DEFAULT 10,
    major_threshold INTEGER NOT NULL DEFAULT 8,
    severe_threshold INTEGER NOT NULL DEFAULT 12,
    armor INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (campaign_id, adversary_id)
);`)
	if err != nil {
		t.Fatalf("create legacy adversaries table: %v", err)
	}

	_, err = sqlDB.Exec(`
INSERT INTO daggerheart_adversaries (
    campaign_id, adversary_id, name, kind, session_id, notes,
    hp, hp_max, stress, stress_max, evasion, major_threshold, severe_threshold,
    armor, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"camp-1", "adv-1", "Ogre", "brute", "sess-1", "legacy row",
		8, 10, 2, 6, 11, 9, 14, 1, 1735689600000, 1735689600000,
	)
	if err != nil {
		t.Fatalf("insert legacy adversary row: %v", err)
	}

	markMigrationsAppliedExcept(t, sqlDB, migrations.ProjectionsFS, "projections", "002_daggerheart_adversaries_conditions.sql")
	applyMigrationSet(t, sqlDB, migrations.ProjectionsFS, "projections")

	var conditions string
	err = sqlDB.QueryRow(`
SELECT conditions_json
FROM daggerheart_adversaries
WHERE campaign_id = ? AND adversary_id = ?`,
		"camp-1", "adv-1",
	).Scan(&conditions)
	if err != nil {
		t.Fatalf("query migrated adversary row: %v", err)
	}
	if conditions != "[]" {
		t.Fatalf("conditions_json = %q, want []", conditions)
	}
}

func TestProjectionMigration006_AddsExpectedNextSeqWithoutDataLoss(t *testing.T) {
	sqlDB := openMigrationTestDB(t)

	_, err := sqlDB.Exec(`
CREATE TABLE projection_watermarks (
    campaign_id TEXT PRIMARY KEY,
    applied_seq INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL
);`)
	if err != nil {
		t.Fatalf("create legacy projection_watermarks table: %v", err)
	}
	_, err = sqlDB.Exec(`
INSERT INTO projection_watermarks (campaign_id, applied_seq, updated_at)
VALUES (?, ?, ?)`,
		"camp-1", 42, 1735689600000,
	)
	if err != nil {
		t.Fatalf("insert legacy watermark row: %v", err)
	}

	markMigrationsAppliedExcept(t, sqlDB, migrations.ProjectionsFS, "projections", "006_watermark_expected_next_seq.sql")
	applyMigrationSet(t, sqlDB, migrations.ProjectionsFS, "projections")

	var appliedSeq, expectedNextSeq int64
	err = sqlDB.QueryRow(`
SELECT applied_seq, expected_next_seq
FROM projection_watermarks
WHERE campaign_id = ?`,
		"camp-1",
	).Scan(&appliedSeq, &expectedNextSeq)
	if err != nil {
		t.Fatalf("query migrated watermark row: %v", err)
	}
	if appliedSeq != 42 {
		t.Fatalf("applied_seq = %d, want 42", appliedSeq)
	}
	if expectedNextSeq != 0 {
		t.Fatalf("expected_next_seq = %d, want 0", expectedNextSeq)
	}
}

func TestProjectionMigration012_AddsSceneFKsAndPreservesRows(t *testing.T) {
	sqlDB := openMigrationTestDB(t)

	_, err := sqlDB.Exec(`
CREATE TABLE campaigns (
    id TEXT PRIMARY KEY
);

CREATE TABLE scenes (
    campaign_id TEXT NOT NULL,
    scene_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    ended_at INTEGER,
    PRIMARY KEY (campaign_id, scene_id)
);

CREATE INDEX idx_scenes_session ON scenes(campaign_id, session_id);
CREATE INDEX idx_scenes_active ON scenes(campaign_id, active);

CREATE TABLE scene_characters (
    campaign_id TEXT NOT NULL,
    scene_id TEXT NOT NULL,
    character_id TEXT NOT NULL,
    added_at INTEGER NOT NULL,
    PRIMARY KEY (campaign_id, scene_id, character_id)
);

CREATE TABLE scene_gates (
    campaign_id TEXT NOT NULL,
    scene_id TEXT NOT NULL,
    gate_id TEXT NOT NULL,
    gate_type TEXT NOT NULL,
    status TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    created_by_actor_type TEXT NOT NULL,
    created_by_actor_id TEXT NOT NULL DEFAULT '',
    resolved_at INTEGER,
    resolved_by_actor_type TEXT,
    resolved_by_actor_id TEXT,
    metadata_json BLOB,
    resolution_json BLOB,
    PRIMARY KEY (campaign_id, scene_id, gate_id)
);

CREATE INDEX idx_scene_gates_open ON scene_gates(campaign_id, scene_id, status);

CREATE TABLE scene_spotlight (
    campaign_id TEXT NOT NULL,
    scene_id TEXT NOT NULL,
    spotlight_type TEXT NOT NULL,
    character_id TEXT NOT NULL DEFAULT '',
    updated_at INTEGER NOT NULL,
    updated_by_actor_type TEXT NOT NULL,
    updated_by_actor_id TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (campaign_id, scene_id)
);`)
	if err != nil {
		t.Fatalf("create legacy scene projection tables: %v", err)
	}

	if _, err := sqlDB.Exec(`INSERT INTO campaigns (id) VALUES (?)`, "camp-1"); err != nil {
		t.Fatalf("seed legacy scene rows: %v", err)
	}
	if _, err := sqlDB.Exec(`
INSERT INTO scenes (campaign_id, scene_id, session_id, name, description, active, created_at, updated_at, ended_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"camp-1", "scene-1", "sess-1", "Scene 1", "", 1, 1735689600000, 1735689600000, nil,
	); err != nil {
		t.Fatalf("seed legacy scene: %v", err)
	}
	if _, err := sqlDB.Exec(`
INSERT INTO scene_characters (campaign_id, scene_id, character_id, added_at)
VALUES (?, ?, ?, ?)`,
		"camp-1", "scene-1", "char-1", 1735689600000,
	); err != nil {
		t.Fatalf("seed legacy scene character: %v", err)
	}
	if _, err := sqlDB.Exec(`
INSERT INTO scene_gates (
    campaign_id, scene_id, gate_id, gate_type, status, reason,
    created_at, created_by_actor_type, created_by_actor_id,
    resolved_at, resolved_by_actor_type, resolved_by_actor_id,
    metadata_json, resolution_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"camp-1", "scene-1", "gate-1", "gm_consequence", "open", "", 1735689600000, "system", "", nil, nil, nil, nil, nil,
	); err != nil {
		t.Fatalf("seed legacy scene gate: %v", err)
	}
	if _, err := sqlDB.Exec(`
INSERT INTO scene_spotlight (campaign_id, scene_id, spotlight_type, character_id, updated_at, updated_by_actor_type, updated_by_actor_id)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"camp-1", "scene-1", "character", "char-1", 1735689600000, "system", "",
	); err != nil {
		t.Fatalf("seed legacy scene spotlight: %v", err)
	}

	markMigrationsAppliedExcept(t, sqlDB, migrations.ProjectionsFS, "projections", "012_scene_projection_foreign_keys.sql")
	applyMigrationSet(t, sqlDB, migrations.ProjectionsFS, "projections")

	var count int
	err = sqlDB.QueryRow(`SELECT COUNT(*) FROM scenes WHERE campaign_id = ? AND scene_id = ?`, "camp-1", "scene-1").Scan(&count)
	if err != nil {
		t.Fatalf("count scenes after migration: %v", err)
	}
	if count != 1 {
		t.Fatalf("scene row count = %d, want 1", count)
	}

	err = sqlDB.QueryRow(`SELECT COUNT(*) FROM scene_characters WHERE campaign_id = ? AND scene_id = ?`, "camp-1", "scene-1").Scan(&count)
	if err != nil {
		t.Fatalf("count scene_characters after migration: %v", err)
	}
	if count != 1 {
		t.Fatalf("scene_character row count = %d, want 1", count)
	}

	_, err = sqlDB.Exec(`
INSERT INTO scene_characters (campaign_id, scene_id, character_id, added_at)
VALUES (?, ?, ?, ?)`,
		"camp-1", "missing-scene", "char-x", 1735689600000,
	)
	if err == nil {
		t.Fatal("expected foreign key error when inserting scene character without scene")
	}
}

func openMigrationTestDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "migration-upgrade.sqlite")
	sqlDB, err := sqliteconn.Open(path)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Fatalf("close sqlite db: %v", err)
		}
	})
	return sqlDB
}

func applyMigrationSet(t *testing.T, sqlDB *sql.DB, migrationFS fs.FS, root string) {
	t.Helper()
	if err := sqlitemigrate.ApplyMigrations(sqlDB, migrationFS, root); err != nil {
		t.Fatalf("apply migrations for %s: %v", root, err)
	}
}

func markMigrationsAppliedExcept(t *testing.T, sqlDB *sql.DB, migrationFS fs.FS, root, targetFile string) {
	t.Helper()
	entries, err := fs.ReadDir(migrationFS, root)
	if err != nil {
		t.Fatalf("read migration dir %s: %v", root, err)
	}

	_, err = sqlDB.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
    name TEXT PRIMARY KEY,
    applied_at INTEGER NOT NULL
);`)
	if err != nil {
		t.Fatalf("ensure schema_migrations: %v", err)
	}

	appliedAt := time.Now().UTC().UnixMilli()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") || entry.Name() == targetFile {
			continue
		}
		name := path.Join(root, entry.Name())
		if _, err := sqlDB.Exec(`INSERT OR IGNORE INTO schema_migrations (name, applied_at) VALUES (?, ?)`, name, appliedAt); err != nil {
			t.Fatalf("mark migration %s as applied: %v", name, err)
		}
	}
}
