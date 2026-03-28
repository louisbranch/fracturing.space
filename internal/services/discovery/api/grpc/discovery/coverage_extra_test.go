package discovery

import (
	"context"
	"errors"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	"github.com/louisbranch/fracturing.space/internal/services/discovery/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubDiscoveryStore struct {
	createFunc func(context.Context, storage.DiscoveryEntry) error
	getFunc    func(context.Context, string) (storage.DiscoveryEntry, error)
	listFunc   func(context.Context, int, string, discoveryv1.DiscoveryEntryKind) (storage.DiscoveryEntryPage, error)

	lastCreate storage.DiscoveryEntry
	lastList   struct {
		pageSize  int
		pageToken string
		kind      discoveryv1.DiscoveryEntryKind
	}
}

func (s *stubDiscoveryStore) CreateDiscoveryEntry(ctx context.Context, entry storage.DiscoveryEntry) error {
	s.lastCreate = entry
	if s.createFunc != nil {
		return s.createFunc(ctx, entry)
	}
	return nil
}

func (s *stubDiscoveryStore) GetDiscoveryEntry(ctx context.Context, entryID string) (storage.DiscoveryEntry, error) {
	if s.getFunc != nil {
		return s.getFunc(ctx, entryID)
	}
	return storage.DiscoveryEntry{}, nil
}

func (s *stubDiscoveryStore) ListDiscoveryEntries(ctx context.Context, pageSize int, pageToken string, kind discoveryv1.DiscoveryEntryKind) (storage.DiscoveryEntryPage, error) {
	s.lastList.pageSize = pageSize
	s.lastList.pageToken = pageToken
	s.lastList.kind = kind
	if s.listFunc != nil {
		return s.listFunc(ctx, pageSize, pageToken, kind)
	}
	return storage.DiscoveryEntryPage{}, nil
}

func validDiscoveryEntry() *discoveryv1.DiscoveryEntry {
	return &discoveryv1.DiscoveryEntry{
		EntryId:                    "starter:extra",
		Kind:                       discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_CAMPAIGN_STARTER,
		SourceId:                   "starter-1",
		Title:                      "Starter",
		Description:                "Short description",
		RecommendedParticipantsMin: 2,
		RecommendedParticipantsMax: 4,
		DifficultyTier:             discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER,
		ExpectedDurationLabel:      "2 sessions",
		System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:                     discoveryv1.DiscoveryGmMode_DISCOVERY_GM_MODE_AI,
		Intent:                     discoveryv1.DiscoveryIntent_DISCOVERY_INTENT_STARTER,
	}
}

func TestCreateDiscoveryEntryValidationAndStoreErrors(t *testing.T) {
	t.Parallel()

	var nilService *Service
	if _, err := nilService.CreateDiscoveryEntry(context.Background(), &discoveryv1.CreateDiscoveryEntryRequest{}); status.Code(err) != codes.Internal {
		t.Fatalf("nil service code = %v, want %v", status.Code(err), codes.Internal)
	}

	svc := NewService(&stubDiscoveryStore{})
	tests := []struct {
		name string
		mut  func(*discoveryv1.DiscoveryEntry)
	}{
		{name: "entry id", mut: func(entry *discoveryv1.DiscoveryEntry) { entry.EntryId = " " }},
		{name: "kind", mut: func(entry *discoveryv1.DiscoveryEntry) {
			entry.Kind = discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_UNSPECIFIED
		}},
		{name: "title", mut: func(entry *discoveryv1.DiscoveryEntry) { entry.Title = "" }},
		{name: "description", mut: func(entry *discoveryv1.DiscoveryEntry) { entry.Description = "" }},
		{name: "expected duration", mut: func(entry *discoveryv1.DiscoveryEntry) { entry.ExpectedDurationLabel = "" }},
		{name: "difficulty tier", mut: func(entry *discoveryv1.DiscoveryEntry) {
			entry.DifficultyTier = discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_UNSPECIFIED
		}},
		{name: "system", mut: func(entry *discoveryv1.DiscoveryEntry) { entry.System = commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED }},
		{name: "participants min", mut: func(entry *discoveryv1.DiscoveryEntry) { entry.RecommendedParticipantsMin = 0 }},
		{name: "participants max", mut: func(entry *discoveryv1.DiscoveryEntry) { entry.RecommendedParticipantsMax = 1 }},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			entry := validDiscoveryEntry()
			tc.mut(entry)
			_, err := svc.CreateDiscoveryEntry(context.Background(), &discoveryv1.CreateDiscoveryEntryRequest{Entry: entry})
			if status.Code(err) != codes.InvalidArgument {
				t.Fatalf("CreateDiscoveryEntry(%s) code = %v, want %v", tc.name, status.Code(err), codes.InvalidArgument)
			}
		})
	}

	store := &stubDiscoveryStore{
		createFunc: func(context.Context, storage.DiscoveryEntry) error { return storage.ErrAlreadyExists },
	}
	svc = NewService(store)
	_, err := svc.CreateDiscoveryEntry(context.Background(), &discoveryv1.CreateDiscoveryEntryRequest{Entry: validDiscoveryEntry()})
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("CreateDiscoveryEntry(already exists) code = %v, want %v", status.Code(err), codes.AlreadyExists)
	}

	store.createFunc = func(context.Context, storage.DiscoveryEntry) error { return errors.New("boom") }
	_, err = svc.CreateDiscoveryEntry(context.Background(), &discoveryv1.CreateDiscoveryEntryRequest{Entry: validDiscoveryEntry()})
	if status.Code(err) != codes.Internal {
		t.Fatalf("CreateDiscoveryEntry(internal) code = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestCreateDiscoveryEntryUsesUTCClockFallback(t *testing.T) {
	t.Parallel()

	store := &stubDiscoveryStore{}
	svc := NewService(store)
	svc.clock = nil

	_, err := svc.CreateDiscoveryEntry(context.Background(), &discoveryv1.CreateDiscoveryEntryRequest{Entry: validDiscoveryEntry()})
	if err != nil {
		t.Fatalf("CreateDiscoveryEntry() error = %v", err)
	}
	if store.lastCreate.EntryID != "starter:extra" {
		t.Fatalf("stored entry id = %q, want starter:extra", store.lastCreate.EntryID)
	}
	if store.lastCreate.CreatedAt.IsZero() || store.lastCreate.UpdatedAt.IsZero() {
		t.Fatalf("stored timestamps = %+v, want non-zero", store.lastCreate)
	}
	if store.lastCreate.CreatedAt.Location() != time.UTC || store.lastCreate.UpdatedAt.Location() != time.UTC {
		t.Fatalf("stored timestamps must be UTC, got created=%v updated=%v", store.lastCreate.CreatedAt.Location(), store.lastCreate.UpdatedAt.Location())
	}
}

func TestGetAndListDiscoveryEntriesErrorMappingAndPaging(t *testing.T) {
	t.Parallel()

	var nilService *Service
	if _, err := nilService.GetDiscoveryEntry(context.Background(), &discoveryv1.GetDiscoveryEntryRequest{}); status.Code(err) != codes.Internal {
		t.Fatalf("nil service get code = %v, want %v", status.Code(err), codes.Internal)
	}
	if _, err := nilService.ListDiscoveryEntries(context.Background(), &discoveryv1.ListDiscoveryEntriesRequest{}); status.Code(err) != codes.Internal {
		t.Fatalf("nil service list code = %v, want %v", status.Code(err), codes.Internal)
	}

	store := &stubDiscoveryStore{
		getFunc: func(context.Context, string) (storage.DiscoveryEntry, error) {
			return storage.DiscoveryEntry{}, storage.ErrNotFound
		},
		listFunc: func(context.Context, int, string, discoveryv1.DiscoveryEntryKind) (storage.DiscoveryEntryPage, error) {
			return storage.DiscoveryEntryPage{}, errors.New("list boom")
		},
	}
	svc := NewService(store)

	if _, err := svc.GetDiscoveryEntry(context.Background(), nil); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("GetDiscoveryEntry(nil) code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
	if _, err := svc.GetDiscoveryEntry(context.Background(), &discoveryv1.GetDiscoveryEntryRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("GetDiscoveryEntry(empty) code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
	if _, err := svc.GetDiscoveryEntry(context.Background(), &discoveryv1.GetDiscoveryEntryRequest{EntryId: "missing"}); status.Code(err) != codes.NotFound {
		t.Fatalf("GetDiscoveryEntry(not found) code = %v, want %v", status.Code(err), codes.NotFound)
	}

	store.getFunc = func(context.Context, string) (storage.DiscoveryEntry, error) {
		return storage.DiscoveryEntry{}, errors.New("get boom")
	}
	if _, err := svc.GetDiscoveryEntry(context.Background(), &discoveryv1.GetDiscoveryEntryRequest{EntryId: "broken"}); status.Code(err) != codes.Internal {
		t.Fatalf("GetDiscoveryEntry(internal) code = %v, want %v", status.Code(err), codes.Internal)
	}

	if _, err := svc.ListDiscoveryEntries(context.Background(), nil); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("ListDiscoveryEntries(nil) code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
	if _, err := svc.ListDiscoveryEntries(context.Background(), &discoveryv1.ListDiscoveryEntriesRequest{}); status.Code(err) != codes.Internal {
		t.Fatalf("ListDiscoveryEntries(internal) code = %v, want %v", status.Code(err), codes.Internal)
	}

	store.listFunc = func(_ context.Context, pageSize int, pageToken string, kind discoveryv1.DiscoveryEntryKind) (storage.DiscoveryEntryPage, error) {
		return storage.DiscoveryEntryPage{
			Entries: []storage.DiscoveryEntry{
				{
					EntryID:                    "entry-9",
					Kind:                       discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_STORYLINE,
					SourceID:                   "story-9",
					Title:                      "Storyline",
					Description:                "Storyline description",
					RecommendedParticipantsMin: 1,
					RecommendedParticipantsMax: 2,
					DifficultyTier:             discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER,
					ExpectedDurationLabel:      "1 session",
					System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
				},
			},
			NextPageToken: "token-9",
		}, nil
	}

	resp, err := svc.ListDiscoveryEntries(context.Background(), &discoveryv1.ListDiscoveryEntriesRequest{
		PageSize:  99,
		PageToken: "start",
		Kind:      discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_STORYLINE,
	})
	if err != nil {
		t.Fatalf("ListDiscoveryEntries(success) error = %v", err)
	}
	if store.lastList.pageSize != maxListDiscoveryEntriesPageSize || store.lastList.pageToken != "start" || store.lastList.kind != discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_STORYLINE {
		t.Fatalf("list args = %+v", store.lastList)
	}
	if len(resp.GetEntries()) != 1 || resp.GetEntries()[0].GetEntryId() != "entry-9" || resp.GetNextPageToken() != "token-9" {
		t.Fatalf("ListDiscoveryEntries(success) resp = %#v", resp)
	}
}
