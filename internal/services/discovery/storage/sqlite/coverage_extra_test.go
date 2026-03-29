package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	"github.com/louisbranch/fracturing.space/internal/services/discovery/storage"
)

type scanStub struct {
	values []any
	err    error
}

func (s scanStub) Scan(dest ...any) error {
	if s.err != nil {
		return s.err
	}
	for i := range dest {
		switch ptr := dest[i].(type) {
		case *string:
			*ptr = s.values[i].(string)
		case *int:
			*ptr = s.values[i].(int)
		case *int32:
			*ptr = s.values[i].(int32)
		case *int64:
			*ptr = s.values[i].(int64)
		default:
			return errors.New("unsupported scan type")
		}
	}
	return nil
}

func validEntry() storage.DiscoveryEntry {
	now := time.Date(2026, time.March, 28, 20, 0, 0, 0, time.UTC)
	return storage.DiscoveryEntry{
		EntryID:                    "starter:extra",
		Kind:                       discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_CAMPAIGN_STARTER,
		SourceID:                   "starter-1",
		Title:                      "Starter",
		Description:                "Description",
		CampaignTheme:              "Theme",
		RecommendedParticipantsMin: 2,
		RecommendedParticipantsMax: 4,
		DifficultyTier:             discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER,
		ExpectedDurationLabel:      "2 sessions",
		System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:                     discoveryv1.DiscoveryGmMode_DISCOVERY_GM_MODE_AI,
		Intent:                     discoveryv1.DiscoveryIntent_DISCOVERY_INTENT_STARTER,
		Level:                      1,
		CharacterCount:             1,
		Storyline:                  "Storyline",
		Tags:                       []string{"solo", "mystery"},
		PreviewHook:                "Hook",
		PreviewPlaystyleLabel:      "Guardian",
		PreviewCharacterName:       "Mira",
		PreviewCharacterSummary:    "Steadfast guardian",
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}
}

func TestStoreGuardsAndContextBranches(t *testing.T) {
	t.Parallel()

	var nilStore *Store
	if err := nilStore.Close(); err != nil {
		t.Fatalf("(*Store)(nil).Close() error = %v", err)
	}
	if _, err := nilStore.GetDiscoveryEntry(context.Background(), "entry-1"); err == nil || !strings.Contains(err.Error(), "storage is not configured") {
		t.Fatalf("nil store get error = %v", err)
	}
	if _, err := nilStore.ListDiscoveryEntries(context.Background(), 1, "", discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_UNSPECIFIED); err == nil || !strings.Contains(err.Error(), "storage is not configured") {
		t.Fatalf("nil store list error = %v", err)
	}
	if err := nilStore.CreateDiscoveryEntry(context.Background(), validEntry()); err == nil || !strings.Contains(err.Error(), "storage is not configured") {
		t.Fatalf("nil store create error = %v", err)
	}
	if err := nilStore.UpsertBuiltinDiscoveryEntry(context.Background(), validEntry()); err == nil || !strings.Contains(err.Error(), "storage is not configured") {
		t.Fatalf("nil store upsert error = %v", err)
	}
	if err := nilStore.UpdateDiscoveryEntrySourceID(context.Background(), "entry-1", "source-1", time.Now()); err == nil || !strings.Contains(err.Error(), "storage is not configured") {
		t.Fatalf("nil store update source error = %v", err)
	}

	store := openTempStore(t)
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := store.CreateDiscoveryEntry(cancelledCtx, validEntry()); err == nil {
		t.Fatal("CreateDiscoveryEntry(cancelled ctx) error = nil")
	}
	if err := store.UpsertBuiltinDiscoveryEntry(cancelledCtx, validEntry()); err == nil {
		t.Fatal("UpsertBuiltinDiscoveryEntry(cancelled ctx) error = nil")
	}
	if err := store.UpdateDiscoveryEntrySourceID(cancelledCtx, "entry-1", "source-1", time.Now()); err == nil {
		t.Fatal("UpdateDiscoveryEntrySourceID(cancelled ctx) error = nil")
	}
	if _, err := store.GetDiscoveryEntry(cancelledCtx, "entry-1"); err == nil {
		t.Fatal("GetDiscoveryEntry(cancelled ctx) error = nil")
	}
	if _, err := store.ListDiscoveryEntries(cancelledCtx, 1, "", discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_UNSPECIFIED); err == nil {
		t.Fatal("ListDiscoveryEntries(cancelled ctx) error = nil")
	}
	if _, err := store.GetDiscoveryEntry(context.Background(), " "); err == nil || !strings.Contains(err.Error(), "entry id is required") {
		t.Fatalf("GetDiscoveryEntry(empty id) error = %v", err)
	}
	if _, err := store.ListDiscoveryEntries(context.Background(), 0, "", discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_UNSPECIFIED); err == nil || !strings.Contains(err.Error(), "page size must be greater than zero") {
		t.Fatalf("ListDiscoveryEntries(page size) error = %v", err)
	}
	if err := store.UpdateDiscoveryEntrySourceID(context.Background(), "", "source-1", time.Now()); err == nil || !strings.Contains(err.Error(), "entry id is required") {
		t.Fatalf("UpdateDiscoveryEntrySourceID(empty id) error = %v", err)
	}
	if err := store.UpdateDiscoveryEntrySourceID(context.Background(), "entry-1", "", time.Now()); err == nil || !strings.Contains(err.Error(), "source id is required") {
		t.Fatalf("UpdateDiscoveryEntrySourceID(empty source) error = %v", err)
	}
}

func TestNormalizeEntryAndHelperBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mut  func(*storage.DiscoveryEntry)
		want string
	}{
		{name: "entry id", mut: func(e *storage.DiscoveryEntry) { e.EntryID = " " }, want: "entry id is required"},
		{name: "kind", mut: func(e *storage.DiscoveryEntry) {
			e.Kind = discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_UNSPECIFIED
		}, want: "entry kind is required"},
		{name: "title", mut: func(e *storage.DiscoveryEntry) { e.Title = "" }, want: "title is required"},
		{name: "description", mut: func(e *storage.DiscoveryEntry) { e.Description = "" }, want: "description is required"},
		{name: "duration", mut: func(e *storage.DiscoveryEntry) { e.ExpectedDurationLabel = "" }, want: "expected duration label is required"},
		{name: "participants min", mut: func(e *storage.DiscoveryEntry) { e.RecommendedParticipantsMin = 0 }, want: "recommended participants min must be greater than zero"},
		{name: "participants max", mut: func(e *storage.DiscoveryEntry) { e.RecommendedParticipantsMax = 1 }, want: "recommended participants max must be greater than or equal to min"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			entry := validEntry()
			tc.mut(&entry)
			_, err := normalizeEntry(entry)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("normalizeEntry(%s) error = %v, want %q", tc.name, err, tc.want)
			}
		})
	}

	entry := validEntry()
	entry.CreatedAt = time.Time{}
	entry.UpdatedAt = time.Time{}
	normalized, err := normalizeEntry(entry)
	if err != nil {
		t.Fatalf("normalizeEntry(zero times) error = %v", err)
	}
	if normalized.CreatedAt.IsZero() || normalized.UpdatedAt.IsZero() || !normalized.CreatedAt.Equal(normalized.UpdatedAt) {
		t.Fatalf("normalizeEntry(zero times) = %#v", normalized)
	}

	entry = validEntry()
	entry.CreatedAt = time.Time{}
	normalized, err = normalizeEntry(entry)
	if err != nil {
		t.Fatalf("normalizeEntry(created zero) error = %v", err)
	}
	if !normalized.CreatedAt.Equal(entry.UpdatedAt.UTC()) || !normalized.UpdatedAt.Equal(entry.UpdatedAt.UTC()) {
		t.Fatalf("normalizeEntry(created zero) = %#v", normalized)
	}

	entry = validEntry()
	entry.UpdatedAt = time.Time{}
	entry.Tags = []string{"  a  ", "b"}
	normalized, err = normalizeEntry(entry)
	if err != nil {
		t.Fatalf("normalizeEntry(updated zero) error = %v", err)
	}
	if !normalized.CreatedAt.Equal(entry.CreatedAt.UTC()) || !normalized.UpdatedAt.Equal(entry.CreatedAt.UTC()) {
		t.Fatalf("normalizeEntry(updated zero) = %#v", normalized)
	}
	if len(normalized.Tags) != 2 || normalized.Tags[0] != "  a  " {
		t.Fatalf("normalizeEntry(tags) = %#v", normalized.Tags)
	}

	if got := tagsToJSON(nil); got != "[]" {
		t.Fatalf("tagsToJSON(nil) = %q, want []", got)
	}
	if got := tagsFromJSON(""); got != nil {
		t.Fatalf("tagsFromJSON(empty) = %#v, want nil", got)
	}
	if got := tagsFromJSON("[]"); got != nil {
		t.Fatalf("tagsFromJSON([]) = %#v, want nil", got)
	}
	if got := tagsFromJSON("{not-json}"); got != nil {
		t.Fatalf("tagsFromJSON(invalid) = %#v, want nil", got)
	}
	if got := tagsFromJSON(`["solo","mystery"]`); len(got) != 2 || got[0] != "solo" {
		t.Fatalf("tagsFromJSON(valid) = %#v", got)
	}

	if got := isDiscoveryEntryUniqueViolation(nil); got {
		t.Fatal("isDiscoveryEntryUniqueViolation(nil) = true, want false")
	}
	if got := isDiscoveryEntryUniqueViolation(errors.New("UNIQUE constraint failed: discovery_entries.entry_id")); !got {
		t.Fatal("isDiscoveryEntryUniqueViolation(unique message) = false, want true")
	}
	if got := isDiscoveryEntryUniqueViolation(errors.New("other error")); got {
		t.Fatal("isDiscoveryEntryUniqueViolation(other) = true, want false")
	}
}

func TestUpdateGetListAndScanBranches(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	entry := validEntry()
	entry.EntryID = "starter:update"
	entry.SourceID = ""
	entry.CreatedAt = time.Date(2026, time.March, 28, 20, 30, 0, 0, time.UTC)
	entry.UpdatedAt = entry.CreatedAt
	if err := store.CreateDiscoveryEntry(context.Background(), entry); err != nil {
		t.Fatalf("CreateDiscoveryEntry() error = %v", err)
	}

	if err := store.UpdateDiscoveryEntrySourceID(context.Background(), entry.EntryID, "source-updated", time.Time{}); err != nil {
		t.Fatalf("UpdateDiscoveryEntrySourceID() error = %v", err)
	}
	got, err := store.GetDiscoveryEntry(context.Background(), entry.EntryID)
	if err != nil {
		t.Fatalf("GetDiscoveryEntry() error = %v", err)
	}
	if got.SourceID != "source-updated" {
		t.Fatalf("GetDiscoveryEntry() source = %q, want source-updated", got.SourceID)
	}
	if got.UpdatedAt.IsZero() {
		t.Fatalf("GetDiscoveryEntry() updated_at = %v, want non-zero", got.UpdatedAt)
	}

	if err := store.UpdateDiscoveryEntrySourceID(context.Background(), "missing-entry", "source-1", time.Now()); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("UpdateDiscoveryEntrySourceID(missing) error = %v, want %v", err, storage.ErrNotFound)
	}
	if _, err := store.GetDiscoveryEntry(context.Background(), "missing-entry"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("GetDiscoveryEntry(missing) error = %v, want %v", err, storage.ErrNotFound)
	}
	if page, err := store.ListDiscoveryEntries(context.Background(), 10, "zzzz", discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_UNSPECIFIED); err != nil || len(page.Entries) != 0 || page.NextPageToken != "" {
		t.Fatalf("ListDiscoveryEntries(after end) = (%#v, %v)", page, err)
	}

	scanned, err := scanEntry(scanStub{values: []any{
		"entry-9",
		int32(discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_STORYLINE),
		"source-9",
		"Title",
		"Description",
		"Theme",
		1,
		2,
		int32(discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER),
		"1 session",
		int32(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART),
		int32(discoveryv1.DiscoveryGmMode_DISCOVERY_GM_MODE_AI),
		int32(discoveryv1.DiscoveryIntent_DISCOVERY_INTENT_STARTER),
		1,
		1,
		"Storyline",
		`["solo"]`,
		" Hook ",
		" Guardian ",
		" Mira ",
		" Summary ",
		time.Date(2026, time.March, 28, 21, 0, 0, 0, time.UTC).UnixMilli(),
		time.Date(2026, time.March, 28, 22, 0, 0, 0, time.UTC).UnixMilli(),
	}})
	if err != nil {
		t.Fatalf("scanEntry() error = %v", err)
	}
	if scanned.Kind != discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_STORYLINE || scanned.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatalf("scanEntry() enums = %#v", scanned)
	}
	if scanned.PreviewHook != "Hook" || scanned.PreviewCharacterName != "Mira" || len(scanned.Tags) != 1 || scanned.Tags[0] != "solo" {
		t.Fatalf("scanEntry() trimmed fields = %#v", scanned)
	}
	if _, err := scanEntry(scanStub{err: sql.ErrNoRows}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("scanEntry(scan err) = %v, want %v", err, sql.ErrNoRows)
	}
}
