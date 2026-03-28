package coreprojection

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/migrations"
)

func TestOpenProjectionsAndOpenStoreValidation(t *testing.T) {
	t.Parallel()

	if _, err := OpenProjections(" "); err == nil || err.Error() != "storage path is required" {
		t.Fatalf("OpenProjections() error = %v", err)
	}

	path := t.TempDir() + "/broken.sqlite"
	if _, err := openStore(path, migrations.ProjectionsFS, "missing-root"); err == nil || err.Error() != "run migrations: read migrations dir: open missing-root: file does not exist" {
		t.Fatalf("openStore() error = %v, want wrapped migration failure", err)
	}
}

func TestConversionHelpersAndCounts(t *testing.T) {
	t.Parallel()

	if got, err := latestSessionMillis(nil); err != nil || got.Valid {
		t.Fatalf("latestSessionMillis(nil) = (%v, %v)", got, err)
	}
	if got, err := latestSessionMillis(int64(42)); err != nil || !got.Valid || got.Int64 != 42 {
		t.Fatalf("latestSessionMillis(int64) = (%v, %v)", got, err)
	}
	if got, err := latestSessionMillis(sql.NullInt64{Int64: 99, Valid: true}); err != nil || !got.Valid || got.Int64 != 99 {
		t.Fatalf("latestSessionMillis(null) = (%v, %v)", got, err)
	}
	if _, err := latestSessionMillis("oops"); err == nil {
		t.Fatal("latestSessionMillis() expected unsupported type error")
	}

	type payload struct{ Name string }
	var decoded payload
	if err := unmarshalOptionalJSON(" ", &decoded, "payload"); err != nil {
		t.Fatalf("unmarshalOptionalJSON(blank) error = %v", err)
	}
	if decoded.Name != "" {
		t.Fatalf("decoded after blank input = %#v", decoded)
	}
	if err := unmarshalOptionalJSON(`{"name":"aria"}`, &decoded, "payload"); err != nil || decoded.Name != "aria" {
		t.Fatalf("unmarshalOptionalJSON(valid) = (%#v, %v)", decoded, err)
	}
	if err := unmarshalOptionalJSON("{", &decoded, "payload"); err == nil || err.Error() == "" {
		t.Fatalf("unmarshalOptionalJSON(invalid) error = %v", err)
	}

	store := openTestStore(t)
	now := time.Date(2026, 3, 27, 19, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-1", now)
	if err := store.PutParticipant(context.Background(), storage.ParticipantRecord{
		ID:         "part-1",
		CampaignID: "camp-1",
		Name:       "Rook",
		CreatedAt:  now,
		UpdatedAt:  now,
	}); err != nil {
		t.Fatalf("PutParticipant() error = %v", err)
	}
	if err := store.PutParticipant(context.Background(), storage.ParticipantRecord{
		ID:         "part-2",
		CampaignID: "camp-1",
		Name:       "Vale",
		CreatedAt:  now,
		UpdatedAt:  now,
	}); err != nil {
		t.Fatalf("PutParticipant() error = %v", err)
	}

	count, err := store.CountParticipants(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("CountParticipants() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("CountParticipants() = %d, want 2", count)
	}
	if _, err := store.CountParticipants(context.Background(), " "); err == nil || err.Error() != "campaign id is required" {
		t.Fatalf("CountParticipants() validation error = %v", err)
	}
}

func TestSaveProjectionWatermarkValidation(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	err := store.SaveProjectionWatermark(context.Background(), storage.ProjectionWatermark{})
	if err == nil || err.Error() != "campaign id is required" {
		t.Fatalf("SaveProjectionWatermark() error = %v", err)
	}
}
