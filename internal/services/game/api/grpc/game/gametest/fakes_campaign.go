package gametest

import (
	"context"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

var (
	_ storage.CampaignStore     = (*FakeCampaignStore)(nil)
	_ storage.StatisticsStore   = (*FakeStatisticsStore)(nil)
	_ storage.CampaignForkStore = (*FakeCampaignForkStore)(nil)
)

// FakeCampaignStore is a test double for storage.CampaignStore.
type FakeCampaignStore struct {
	Campaigns map[string]storage.CampaignRecord
	PutErr    error
	GetErr    error
	ListErr   error
}

// NewFakeCampaignStore returns a ready-to-use campaign store fake.
func NewFakeCampaignStore() *FakeCampaignStore {
	return &FakeCampaignStore{Campaigns: make(map[string]storage.CampaignRecord)}
}

func (s *FakeCampaignStore) Put(ctx context.Context, c storage.CampaignRecord) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	s.Campaigns[c.ID] = c
	return nil
}

func (s *FakeCampaignStore) Get(ctx context.Context, id string) (storage.CampaignRecord, error) {
	if s.GetErr != nil {
		return storage.CampaignRecord{}, s.GetErr
	}
	c, ok := s.Campaigns[id]
	if !ok {
		return storage.CampaignRecord{}, storage.ErrNotFound
	}
	return c, nil
}

func (s *FakeCampaignStore) List(ctx context.Context, pageSize int, pageToken string) (storage.CampaignPage, error) {
	if s.ListErr != nil {
		return storage.CampaignPage{}, s.ListErr
	}
	campaigns := make([]storage.CampaignRecord, 0, len(s.Campaigns))
	for _, c := range s.Campaigns {
		campaigns = append(campaigns, c)
	}
	return storage.CampaignPage{
		Campaigns:     campaigns,
		NextPageToken: "",
	}, nil
}

// FakeStatisticsStore is a test double for storage.StatisticsStore.
type FakeStatisticsStore struct {
	LastSince *time.Time
	Stats     storage.GameStatistics
	Err       error
}

func (f *FakeStatisticsStore) GetGameStatistics(_ context.Context, since *time.Time) (storage.GameStatistics, error) {
	f.LastSince = since
	return f.Stats, f.Err
}

// FakeCampaignForkStore is a test double for storage.CampaignForkStore.
type FakeCampaignForkStore struct {
	Metadata map[string]storage.ForkMetadata
	GetErr   error
	SetErr   error
}

// NewFakeCampaignForkStore returns a ready-to-use campaign fork store fake.
func NewFakeCampaignForkStore() *FakeCampaignForkStore {
	return &FakeCampaignForkStore{Metadata: make(map[string]storage.ForkMetadata)}
}

func (s *FakeCampaignForkStore) GetCampaignForkMetadata(_ context.Context, campaignID string) (storage.ForkMetadata, error) {
	if s.GetErr != nil {
		return storage.ForkMetadata{}, s.GetErr
	}
	md, ok := s.Metadata[campaignID]
	if !ok {
		return storage.ForkMetadata{}, storage.ErrNotFound
	}
	return md, nil
}

func (s *FakeCampaignForkStore) SetCampaignForkMetadata(_ context.Context, campaignID string, md storage.ForkMetadata) error {
	if s.SetErr != nil {
		return s.SetErr
	}
	s.Metadata[campaignID] = md
	return nil
}
