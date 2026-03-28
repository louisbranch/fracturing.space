package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

func TestOpenRequiresPath(t *testing.T) {
	if _, err := Open(""); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestOpenBootstrapsBaselineSchema(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	ctx := context.Background()

	for _, tableName := range []string{
		"ai_credentials",
		"ai_agents",
		"ai_provider_grants",
		"ai_provider_connect_sessions",
		"ai_access_requests",
		"ai_audit_events",
		"ai_campaign_artifacts",
		"ai_campaign_debug_turns",
		"ai_campaign_debug_turn_entries",
	} {
		if !sqliteObjectExists(t, store, ctx, "table", tableName) {
			t.Fatalf("missing bootstrap table %q", tableName)
		}
	}
}

func TestOpenBootstrapsTypedAgentAuthReferenceSchema(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	ctx := context.Background()
	columns := tableColumns(t, store, ctx, "ai_agents")

	if _, ok := columns["auth_reference_type"]; !ok {
		t.Fatal("missing auth_reference_type column")
	}
	if _, ok := columns["auth_reference_id"]; !ok {
		t.Fatal("missing auth_reference_id column")
	}
	// Invariant: agent auth must persist as one typed authority, not split legacy IDs.
	if _, ok := columns["credential_id"]; ok {
		t.Fatal("unexpected legacy credential_id column")
	}
	// Invariant: agent auth must persist as one typed authority, not split legacy IDs.
	if _, ok := columns["provider_grant_id"]; ok {
		t.Fatal("unexpected legacy provider_grant_id column")
	}
}

func TestOpenBootstrapsKeyIndexDefinitions(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	ctx := context.Background()

	assertNormalizedIndexSQLContains(t, store, ctx, "ai_credentials_owner_active_label_idx",
		"create unique index ai_credentials_owner_active_label_idx on ai_credentials(owner_user_id, lower(trim(label))) where revoked_at is null")
	assertNormalizedIndexSQLContains(t, store, ctx, "ai_agents_owner_auth_reference_idx",
		"create index ai_agents_owner_auth_reference_idx on ai_agents(owner_user_id, auth_reference_type, auth_reference_id)")
	assertNormalizedIndexSQLContains(t, store, ctx, "idx_ai_campaign_debug_turns_campaign_session_started",
		"create index idx_ai_campaign_debug_turns_campaign_session_started on ai_campaign_debug_turns(campaign_id, session_id, started_at desc, id desc)")
}

func sqliteObjectExists(t *testing.T, store *Store, ctx context.Context, objectType, objectName string) bool {
	t.Helper()

	var count int
	err := store.sqlDB.QueryRowContext(ctx, `
SELECT count(*)
FROM sqlite_master
WHERE type = ? AND name = ?
`, objectType, objectName).Scan(&count)
	if err != nil {
		t.Fatalf("query sqlite_master for %s %q: %v", objectType, objectName, err)
	}
	return count == 1
}

func tableColumns(t *testing.T, store *Store, ctx context.Context, tableName string) map[string]struct{} {
	t.Helper()

	rows, err := store.sqlDB.QueryContext(ctx, `PRAGMA table_info(`+tableName+`)`)
	if err != nil {
		t.Fatalf("pragma table_info(%s): %v", tableName, err)
	}
	defer rows.Close()

	columns := make(map[string]struct{})
	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			dfltValue  sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &dfltValue, &pk); err != nil {
			t.Fatalf("scan table_info(%s) row: %v", tableName, err)
		}
		columns[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table_info(%s): %v", tableName, err)
	}

	return columns
}

func assertNormalizedIndexSQLContains(t *testing.T, store *Store, ctx context.Context, indexName, want string) {
	t.Helper()

	var sqlText string
	err := store.sqlDB.QueryRowContext(ctx, `
SELECT sql
FROM sqlite_master
WHERE type = 'index' AND name = ?
`, indexName).Scan(&sqlText)
	if err != nil {
		t.Fatalf("query sqlite_master index %q: %v", indexName, err)
	}

	got := normalizeSQLiteSchemaSQL(sqlText)
	want = normalizeSQLiteSchemaSQL(want)
	if !strings.Contains(got, want) {
		t.Fatalf("index %q sql = %q, want fragment %q", indexName, got, want)
	}
}

func normalizeSQLiteSchemaSQL(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(value)), " ")
}
