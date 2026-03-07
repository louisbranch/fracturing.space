package discovery

import (
	"context"
	"sort"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	"github.com/louisbranch/fracturing.space/internal/services/discovery/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreateDiscoveryEntry_ValidatesRequiredFields(t *testing.T) {
	svc := NewService(newFakeStore())

	_, err := svc.CreateDiscoveryEntry(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("nil request code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}

	_, err = svc.CreateDiscoveryEntry(context.Background(), &discoveryv1.CreateDiscoveryEntryRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("missing entry code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestCreateGetDiscoveryEntry_RoundTrip(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)
	now := time.Date(2026, time.March, 6, 12, 0, 0, 0, time.UTC)
	svc.clock = func() time.Time { return now }

	createReq := &discoveryv1.CreateDiscoveryEntryRequest{Entry: &discoveryv1.DiscoveryEntry{
		EntryId:                    "starter:camp-1",
		Kind:                       discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_CAMPAIGN_STARTER,
		SourceId:                   "camp-1",
		Title:                      "Sunfall",
		Description:                "A haunted valley campaign",
		RecommendedParticipantsMin: 2,
		RecommendedParticipantsMax: 4,
		DifficultyTier:             discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER,
		ExpectedDurationLabel:      "2-3 sessions",
		System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:                     discoveryv1.DiscoveryGmMode_DISCOVERY_GM_MODE_AI,
		Intent:                     discoveryv1.DiscoveryIntent_DISCOVERY_INTENT_STARTER,
		Level:                      1,
		CharacterCount:             1,
		Storyline:                  "# Sunfall",
		Tags:                       []string{"solo", "mystery"},
	}}
	createResp, err := svc.CreateDiscoveryEntry(context.Background(), createReq)
	if err != nil {
		t.Fatalf("create discovery entry: %v", err)
	}
	if got := createResp.GetEntry().GetEntryId(); got != "starter:camp-1" {
		t.Fatalf("entry_id = %q, want starter:camp-1", got)
	}

	getResp, err := svc.GetDiscoveryEntry(context.Background(), &discoveryv1.GetDiscoveryEntryRequest{EntryId: "starter:camp-1"})
	if err != nil {
		t.Fatalf("get discovery entry: %v", err)
	}
	if got := getResp.GetEntry().GetSourceId(); got != "camp-1" {
		t.Fatalf("source_id = %q, want camp-1", got)
	}
}

func TestListDiscoveryEntries_FiltersByKind(t *testing.T) {
	store := newFakeStore()
	store.records["entry-1"] = storage.DiscoveryEntry{
		EntryID:                    "entry-1",
		Kind:                       discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_CAMPAIGN_STARTER,
		SourceID:                   "camp-1",
		Title:                      "Starter",
		Description:                "Starter",
		RecommendedParticipantsMin: 1,
		RecommendedParticipantsMax: 1,
		DifficultyTier:             discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER,
		ExpectedDurationLabel:      "1 session",
		System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
	}
	store.records["entry-2"] = storage.DiscoveryEntry{
		EntryID:                    "entry-2",
		Kind:                       discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_STORYLINE,
		SourceID:                   "story-1",
		Title:                      "Storyline",
		Description:                "Storyline",
		RecommendedParticipantsMin: 1,
		RecommendedParticipantsMax: 1,
		DifficultyTier:             discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER,
		ExpectedDurationLabel:      "1 session",
		System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
	}

	svc := NewService(store)
	resp, err := svc.ListDiscoveryEntries(context.Background(), &discoveryv1.ListDiscoveryEntriesRequest{
		PageSize: 10,
		Kind:     discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_CAMPAIGN_STARTER,
	})
	if err != nil {
		t.Fatalf("list discovery entries: %v", err)
	}
	if len(resp.GetEntries()) != 1 {
		t.Fatalf("entries len = %d, want 1", len(resp.GetEntries()))
	}
	if got := resp.GetEntries()[0].GetEntryId(); got != "entry-1" {
		t.Fatalf("entry_id = %q, want entry-1", got)
	}
}

type fakeStore struct {
	records map[string]storage.DiscoveryEntry
}

func newFakeStore() *fakeStore {
	return &fakeStore{records: map[string]storage.DiscoveryEntry{}}
}

func (f *fakeStore) CreateDiscoveryEntry(_ context.Context, entry storage.DiscoveryEntry) error {
	if _, exists := f.records[entry.EntryID]; exists {
		return storage.ErrAlreadyExists
	}
	f.records[entry.EntryID] = entry
	return nil
}

func (f *fakeStore) GetDiscoveryEntry(_ context.Context, entryID string) (storage.DiscoveryEntry, error) {
	entry, ok := f.records[entryID]
	if !ok {
		return storage.DiscoveryEntry{}, storage.ErrNotFound
	}
	return entry, nil
}

func (f *fakeStore) ListDiscoveryEntries(
	_ context.Context,
	pageSize int,
	pageToken string,
	kind discoveryv1.DiscoveryEntryKind,
) (storage.DiscoveryEntryPage, error) {
	ids := make([]string, 0, len(f.records))
	for id := range f.records {
		if kind != discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_UNSPECIFIED && f.records[id].Kind != kind {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)

	start := 0
	if pageToken != "" {
		start = sort.Search(len(ids), func(i int) bool { return ids[i] > pageToken })
	}
	if start >= len(ids) {
		return storage.DiscoveryEntryPage{}, nil
	}

	end := start + pageSize
	if end > len(ids) {
		end = len(ids)
	}
	page := storage.DiscoveryEntryPage{Entries: make([]storage.DiscoveryEntry, 0, end-start)}
	for _, id := range ids[start:end] {
		page.Entries = append(page.Entries, f.records[id])
	}
	if end < len(ids) {
		page.NextPageToken = ids[end-1]
	}
	return page, nil
}
