package gametest

import (
	"context"
	"sort"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// FakeParticipantStore is a test double for storage.ParticipantStore.
type FakeParticipantStore struct {
	Participants                      map[string]map[string]storage.ParticipantRecord // campaignID -> participantID -> Participant
	PutErr                            error
	GetErr                            error
	DeleteErr                         error
	ListErr                           error
	ListByCampaignCalls               int
	ListCampaignIDsByUserErr          error
	ListCampaignIDsByUserCalls        int
	ListCampaignIDsByParticipantErr   error
	ListCampaignIDsByParticipantCalls int
}

// NewFakeParticipantStore returns a ready-to-use participant store fake.
func NewFakeParticipantStore() *FakeParticipantStore {
	return &FakeParticipantStore{
		Participants: make(map[string]map[string]storage.ParticipantRecord),
	}
}

func (s *FakeParticipantStore) PutParticipant(_ context.Context, p storage.ParticipantRecord) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	if s.Participants[p.CampaignID] == nil {
		s.Participants[p.CampaignID] = make(map[string]storage.ParticipantRecord)
	}
	if strings.TrimSpace(p.UserID) != "" {
		for id, existing := range s.Participants[p.CampaignID] {
			if id == p.ID {
				continue
			}
			if strings.TrimSpace(existing.UserID) == p.UserID {
				return apperrors.WithMetadata(
					apperrors.CodeParticipantUserAlreadyClaimed,
					"participant User already claimed",
					map[string]string{
						"CampaignID": p.CampaignID,
						"UserID":     p.UserID,
					},
				)
			}
		}
	}
	s.Participants[p.CampaignID][p.ID] = p
	return nil
}

func (s *FakeParticipantStore) GetParticipant(_ context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
	if s.GetErr != nil {
		return storage.ParticipantRecord{}, s.GetErr
	}
	byID, ok := s.Participants[campaignID]
	if !ok {
		return storage.ParticipantRecord{}, storage.ErrNotFound
	}
	p, ok := byID[participantID]
	if !ok {
		return storage.ParticipantRecord{}, storage.ErrNotFound
	}
	return p, nil
}

func (s *FakeParticipantStore) DeleteParticipant(_ context.Context, campaignID, participantID string) error {
	if s.DeleteErr != nil {
		return s.DeleteErr
	}
	byID, ok := s.Participants[campaignID]
	if !ok {
		return storage.ErrNotFound
	}
	if _, ok := byID[participantID]; !ok {
		return storage.ErrNotFound
	}
	delete(byID, participantID)
	return nil
}

func (s *FakeParticipantStore) ListParticipantsByCampaign(_ context.Context, campaignID string) ([]storage.ParticipantRecord, error) {
	s.ListByCampaignCalls++
	if s.ListErr != nil {
		return nil, s.ListErr
	}
	byID, ok := s.Participants[campaignID]
	if !ok {
		return nil, nil
	}
	result := make([]storage.ParticipantRecord, 0, len(byID))
	for _, p := range byID {
		result = append(result, p)
	}
	return result, nil
}

func (s *FakeParticipantStore) ListCampaignIDsByUser(_ context.Context, userID string) ([]string, error) {
	s.ListCampaignIDsByUserCalls++
	if s.ListCampaignIDsByUserErr != nil {
		return nil, s.ListCampaignIDsByUserErr
	}
	userID = strings.TrimSpace(userID)
	ids := make([]string, 0)
	seen := make(map[string]struct{})
	if userID == "" {
		return ids, nil
	}
	for _, byID := range s.Participants {
		for _, participant := range byID {
			if strings.TrimSpace(participant.UserID) != userID {
				continue
			}
			campaignID := strings.TrimSpace(participant.CampaignID)
			if campaignID == "" {
				continue
			}
			if _, ok := seen[campaignID]; ok {
				continue
			}
			seen[campaignID] = struct{}{}
			ids = append(ids, campaignID)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

func (s *FakeParticipantStore) ListCampaignIDsByParticipant(_ context.Context, participantID string) ([]string, error) {
	s.ListCampaignIDsByParticipantCalls++
	if s.ListCampaignIDsByParticipantErr != nil {
		return nil, s.ListCampaignIDsByParticipantErr
	}
	participantID = strings.TrimSpace(participantID)
	ids := make([]string, 0)
	seen := make(map[string]struct{})
	if participantID == "" {
		return ids, nil
	}
	for _, byID := range s.Participants {
		for _, participant := range byID {
			if strings.TrimSpace(participant.ID) != participantID {
				continue
			}
			campaignID := strings.TrimSpace(participant.CampaignID)
			if campaignID == "" {
				continue
			}
			if _, ok := seen[campaignID]; ok {
				continue
			}
			seen[campaignID] = struct{}{}
			ids = append(ids, campaignID)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

func (s *FakeParticipantStore) ListParticipants(_ context.Context, campaignID string, pageSize int, pageToken string) (storage.ParticipantPage, error) {
	if s.ListErr != nil {
		return storage.ParticipantPage{}, s.ListErr
	}
	byID, ok := s.Participants[campaignID]
	if !ok {
		return storage.ParticipantPage{}, nil
	}
	result := make([]storage.ParticipantRecord, 0, len(byID))
	for _, p := range byID {
		result = append(result, p)
	}
	return storage.ParticipantPage{
		Participants:  result,
		NextPageToken: "",
	}, nil
}

func (s *FakeParticipantStore) CountParticipants(_ context.Context, campaignID string) (int, error) {
	byID, ok := s.Participants[campaignID]
	if !ok {
		return 0, nil
	}
	return len(byID), nil
}
