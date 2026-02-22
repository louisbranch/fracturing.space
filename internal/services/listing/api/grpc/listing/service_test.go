package listing

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
	"github.com/louisbranch/fracturing.space/internal/services/listing/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreateCampaignListing_NilRequest(t *testing.T) {
	svc := NewService(nil)
	_, err := svc.CreateCampaignListing(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestCreateCampaignListing_MissingCampaignID(t *testing.T) {
	svc := NewService(newFakeCampaignListingStore())
	_, err := svc.CreateCampaignListing(context.Background(), &listingv1.CreateCampaignListingRequest{
		Title: "Starter",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestCreateCampaignListing_RejectsIncompleteDiscoveryMetadata(t *testing.T) {
	base := &listingv1.CreateCampaignListingRequest{
		CampaignId:                 "camp-1",
		Title:                      "Sunfall",
		Description:                "A haunted valley campaign",
		RecommendedParticipantsMin: 3,
		RecommendedParticipantsMax: 5,
		DifficultyTier:             listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_BEGINNER,
		ExpectedDurationLabel:      "2-3 sessions",
		System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
	}

	testCases := []struct {
		name string
		mut  func(*listingv1.CreateCampaignListingRequest)
	}{
		{
			name: "missing description",
			mut:  func(req *listingv1.CreateCampaignListingRequest) { req.Description = " " },
		},
		{
			name: "missing expected duration label",
			mut:  func(req *listingv1.CreateCampaignListingRequest) { req.ExpectedDurationLabel = " " },
		},
		{
			name: "unspecified difficulty tier",
			mut: func(req *listingv1.CreateCampaignListingRequest) {
				req.DifficultyTier = listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_UNSPECIFIED
			},
		},
		{
			name: "unspecified game system",
			mut:  func(req *listingv1.CreateCampaignListingRequest) { req.System = 0 },
		},
	}

	svc := NewService(newFakeCampaignListingStore())
	for idx, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := *base
			req.CampaignId = fmt.Sprintf("camp-%d", idx+1)
			tc.mut(&req)
			_, err := svc.CreateCampaignListing(context.Background(), &req)
			if status.Code(err) != codes.InvalidArgument {
				t.Fatalf("code = %v, want %v", status.Code(err), codes.InvalidArgument)
			}
		})
	}
}

func TestCreateCampaignListing_SuccessAndDuplicate(t *testing.T) {
	store := newFakeCampaignListingStore()
	svc := NewService(store)
	now := time.Date(2026, time.February, 22, 15, 0, 0, 0, time.UTC)
	svc.clock = func() time.Time { return now }

	req := &listingv1.CreateCampaignListingRequest{
		CampaignId:                 "camp-starter-1",
		Title:                      "Sunfall",
		Description:                "A haunted valley campaign",
		RecommendedParticipantsMin: 3,
		RecommendedParticipantsMax: 5,
		DifficultyTier:             listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_BEGINNER,
		ExpectedDurationLabel:      "2-3 sessions",
		System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
	}
	resp, err := svc.CreateCampaignListing(context.Background(), req)
	if err != nil {
		t.Fatalf("create listing: %v", err)
	}
	if got := resp.GetListing().GetCampaignId(); got != "camp-starter-1" {
		t.Fatalf("campaign_id = %q, want camp-starter-1", got)
	}
	if resp.GetListing().GetCreatedAt() == nil || resp.GetListing().GetUpdatedAt() == nil {
		t.Fatal("expected timestamps")
	}

	_, err = svc.CreateCampaignListing(context.Background(), req)
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("duplicate code = %v, want %v", status.Code(err), codes.AlreadyExists)
	}
}

func TestCreateCampaignListing_ReturnsCreatedListingWhenReadbackFails(t *testing.T) {
	store := newFakeCampaignListingStore()
	store.getCampaignListingErr = errors.New("readback timeout")
	svc := NewService(store)
	now := time.Date(2026, time.February, 22, 15, 5, 0, 0, time.UTC)
	svc.clock = func() time.Time { return now }

	resp, err := svc.CreateCampaignListing(context.Background(), &listingv1.CreateCampaignListingRequest{
		CampaignId:                 "camp-starter-readback-failure",
		Title:                      "Sunfall",
		Description:                "A haunted valley campaign",
		RecommendedParticipantsMin: 3,
		RecommendedParticipantsMax: 5,
		DifficultyTier:             listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_BEGINNER,
		ExpectedDurationLabel:      "2-3 sessions",
		System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
	})
	if err != nil {
		t.Fatalf("create listing: %v", err)
	}
	if resp.GetListing().GetCampaignId() != "camp-starter-readback-failure" {
		t.Fatalf("campaign_id = %q, want camp-starter-readback-failure", resp.GetListing().GetCampaignId())
	}
	if got := resp.GetListing().GetCreatedAt().AsTime().UTC(); !got.Equal(now) {
		t.Fatalf("created_at = %v, want %v", got, now)
	}
}

func TestGetCampaignListing_NotFound(t *testing.T) {
	svc := NewService(newFakeCampaignListingStore())
	_, err := svc.GetCampaignListing(context.Background(), &listingv1.GetCampaignListingRequest{
		CampaignId: "missing",
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.NotFound)
	}
}

func TestGetCampaignListing_Success(t *testing.T) {
	store := newFakeCampaignListingStore()
	now := time.Date(2026, time.February, 22, 15, 10, 0, 0, time.UTC)
	store.records["camp-1"] = storage.CampaignListing{
		CampaignID:                 "camp-1",
		Title:                      "Sunfall",
		Description:                "A haunted valley campaign",
		DifficultyTier:             listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_INTERMEDIATE,
		ExpectedDurationLabel:      "4-6 sessions",
		RecommendedParticipantsMin: 3,
		RecommendedParticipantsMax: 5,
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}
	svc := NewService(store)

	resp, err := svc.GetCampaignListing(context.Background(), &listingv1.GetCampaignListingRequest{
		CampaignId: "camp-1",
	})
	if err != nil {
		t.Fatalf("get listing: %v", err)
	}
	if got := resp.GetListing().GetTitle(); got != "Sunfall" {
		t.Fatalf("title = %q, want Sunfall", got)
	}
}

func TestListCampaignListings_Paginates(t *testing.T) {
	store := newFakeCampaignListingStore()
	now := time.Date(2026, time.February, 22, 16, 0, 0, 0, time.UTC)
	for _, id := range []string{"camp-1", "camp-2", "camp-3"} {
		store.records[id] = storage.CampaignListing{
			CampaignID:                 id,
			Title:                      "Title " + id,
			Description:                "Desc " + id,
			DifficultyTier:             listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_BEGINNER,
			ExpectedDurationLabel:      "2 sessions",
			RecommendedParticipantsMin: 3,
			RecommendedParticipantsMax: 5,
			CreatedAt:                  now,
			UpdatedAt:                  now,
		}
	}
	svc := NewService(store)

	first, err := svc.ListCampaignListings(context.Background(), &listingv1.ListCampaignListingsRequest{
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("list page 1: %v", err)
	}
	if len(first.GetListings()) != 2 {
		t.Fatalf("page 1 len = %d, want 2", len(first.GetListings()))
	}
	if first.GetNextPageToken() == "" {
		t.Fatal("expected next page token")
	}

	second, err := svc.ListCampaignListings(context.Background(), &listingv1.ListCampaignListingsRequest{
		PageSize:  2,
		PageToken: first.GetNextPageToken(),
	})
	if err != nil {
		t.Fatalf("list page 2: %v", err)
	}
	if len(second.GetListings()) != 1 {
		t.Fatalf("page 2 len = %d, want 1", len(second.GetListings()))
	}
	if second.GetNextPageToken() != "" {
		t.Fatalf("page 2 next token = %q, want empty", second.GetNextPageToken())
	}
}

func TestListCampaignListings_UnknownPageTokenUsesKeysetSemantics(t *testing.T) {
	store := newFakeCampaignListingStore()
	now := time.Date(2026, time.February, 22, 16, 20, 0, 0, time.UTC)
	for _, id := range []string{"camp-1", "camp-2", "camp-3"} {
		store.records[id] = storage.CampaignListing{
			CampaignID:                 id,
			Title:                      "Title " + id,
			Description:                "Desc " + id,
			DifficultyTier:             listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_BEGINNER,
			ExpectedDurationLabel:      "2 sessions",
			RecommendedParticipantsMin: 3,
			RecommendedParticipantsMax: 5,
			CreatedAt:                  now,
			UpdatedAt:                  now,
		}
	}
	svc := NewService(store)

	resp, err := svc.ListCampaignListings(context.Background(), &listingv1.ListCampaignListingsRequest{
		PageSize:  2,
		PageToken: "camp-1.5",
	})
	if err != nil {
		t.Fatalf("list with unknown token: %v", err)
	}
	if len(resp.GetListings()) != 2 {
		t.Fatalf("len(listings) = %d, want 2", len(resp.GetListings()))
	}
	if got := resp.GetListings()[0].GetCampaignId(); got != "camp-2" {
		t.Fatalf("first listing campaign_id = %q, want camp-2", got)
	}
}

type fakeCampaignListingStore struct {
	records                 map[string]storage.CampaignListing
	createCampaignListErr   error
	getCampaignListingErr   error
	listCampaignListingsErr error
}

func newFakeCampaignListingStore() *fakeCampaignListingStore {
	return &fakeCampaignListingStore{
		records: make(map[string]storage.CampaignListing),
	}
}

func (f *fakeCampaignListingStore) CreateCampaignListing(_ context.Context, listing storage.CampaignListing) error {
	if f.createCampaignListErr != nil {
		return f.createCampaignListErr
	}
	if _, exists := f.records[listing.CampaignID]; exists {
		return storage.ErrAlreadyExists
	}
	f.records[listing.CampaignID] = listing
	return nil
}

func (f *fakeCampaignListingStore) GetCampaignListing(_ context.Context, campaignID string) (storage.CampaignListing, error) {
	if f.getCampaignListingErr != nil {
		return storage.CampaignListing{}, f.getCampaignListingErr
	}
	if listing, ok := f.records[campaignID]; ok {
		return listing, nil
	}
	return storage.CampaignListing{}, storage.ErrNotFound
}

func (f *fakeCampaignListingStore) ListCampaignListings(_ context.Context, pageSize int, pageToken string) (storage.CampaignListingPage, error) {
	if f.listCampaignListingsErr != nil {
		return storage.CampaignListingPage{}, f.listCampaignListingsErr
	}
	ids := make([]string, 0, len(f.records))
	for id := range f.records {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	start := 0
	if pageToken != "" {
		start = sort.Search(len(ids), func(i int) bool {
			return ids[i] > pageToken
		})
	}
	if start >= len(ids) {
		return storage.CampaignListingPage{}, nil
	}
	end := start + pageSize
	if end > len(ids) {
		end = len(ids)
	}

	page := storage.CampaignListingPage{
		Listings: make([]storage.CampaignListing, 0, end-start),
	}
	for _, id := range ids[start:end] {
		page.Listings = append(page.Listings, f.records[id])
	}
	if end < len(ids) {
		page.NextPageToken = ids[end-1]
	}
	return page, nil
}
