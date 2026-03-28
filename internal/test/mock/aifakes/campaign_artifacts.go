package aifakes

import (
	"context"
	"sort"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/ai/campaignartifact"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// CampaignArtifactStore is an in-memory campaign-artifact repository fake.
type CampaignArtifactStore struct {
	CampaignArtifacts map[string]campaignartifact.Artifact
	PutErr            error
	GetErr            error
	ListErr           error
	DeleteErr         error
}

// NewCampaignArtifactStore creates an initialized campaign-artifact fake.
func NewCampaignArtifactStore() *CampaignArtifactStore {
	return &CampaignArtifactStore{CampaignArtifacts: make(map[string]campaignartifact.Artifact)}
}

func campaignArtifactKey(campaignID, path string) string {
	return strings.TrimSpace(campaignID) + "\x00" + strings.TrimSpace(path)
}

// PutCampaignArtifact stores one campaign-scoped GM artifact snapshot.
func (s *CampaignArtifactStore) PutCampaignArtifact(_ context.Context, record campaignartifact.Artifact) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	if s.CampaignArtifacts == nil {
		s.CampaignArtifacts = make(map[string]campaignartifact.Artifact)
	}
	s.CampaignArtifacts[campaignArtifactKey(record.CampaignID, record.Path)] = record
	return nil
}

// GetCampaignArtifact returns one campaign artifact by campaign and path.
func (s *CampaignArtifactStore) GetCampaignArtifact(_ context.Context, campaignID string, path string) (campaignartifact.Artifact, error) {
	if s.GetErr != nil {
		return campaignartifact.Artifact{}, s.GetErr
	}
	record, ok := s.CampaignArtifacts[campaignArtifactKey(campaignID, path)]
	if !ok {
		return campaignartifact.Artifact{}, storage.ErrNotFound
	}
	return record, nil
}

// ListCampaignArtifacts returns all artifacts for one campaign ordered by path.
func (s *CampaignArtifactStore) ListCampaignArtifacts(_ context.Context, campaignID string) ([]campaignartifact.Artifact, error) {
	if s.ListErr != nil {
		return nil, s.ListErr
	}
	records := make([]campaignartifact.Artifact, 0)
	for _, record := range s.CampaignArtifacts {
		if strings.TrimSpace(record.CampaignID) == strings.TrimSpace(campaignID) {
			records = append(records, record)
		}
	}
	sort.Slice(records, func(i, j int) bool {
		return strings.Compare(records[i].Path, records[j].Path) < 0
	})
	return records, nil
}
