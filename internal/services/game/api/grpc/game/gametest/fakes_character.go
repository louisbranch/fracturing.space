package gametest

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

var _ storage.CharacterStore = (*FakeCharacterStore)(nil)

// FakeCharacterStore is a test double for storage.CharacterStore.
type FakeCharacterStore struct {
	Characters          map[string]map[string]storage.CharacterRecord // campaignID -> characterID -> Character
	PutErr              error
	GetErr              error
	DeleteErr           error
	ListErr             error
	ListCalls           int
	ListByOwnerCalls    int
	LastListCampaignID  string
	LastOwnerCampaignID string
	LastOwnerID         string
}

// NewFakeCharacterStore returns a ready-to-use character store fake.
func NewFakeCharacterStore() *FakeCharacterStore {
	return &FakeCharacterStore{
		Characters: make(map[string]map[string]storage.CharacterRecord),
	}
}

func (s *FakeCharacterStore) PutCharacter(_ context.Context, c storage.CharacterRecord) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	if s.Characters[c.CampaignID] == nil {
		s.Characters[c.CampaignID] = make(map[string]storage.CharacterRecord)
	}
	s.Characters[c.CampaignID][c.ID] = c
	return nil
}

func (s *FakeCharacterStore) GetCharacter(_ context.Context, campaignID, characterID string) (storage.CharacterRecord, error) {
	if s.GetErr != nil {
		return storage.CharacterRecord{}, s.GetErr
	}
	byID, ok := s.Characters[campaignID]
	if !ok {
		return storage.CharacterRecord{}, storage.ErrNotFound
	}
	c, ok := byID[characterID]
	if !ok {
		return storage.CharacterRecord{}, storage.ErrNotFound
	}
	return c, nil
}

func (s *FakeCharacterStore) DeleteCharacter(_ context.Context, campaignID, characterID string) error {
	if s.DeleteErr != nil {
		return s.DeleteErr
	}
	byID, ok := s.Characters[campaignID]
	if !ok {
		return storage.ErrNotFound
	}
	if _, ok := byID[characterID]; !ok {
		return storage.ErrNotFound
	}
	delete(byID, characterID)
	return nil
}

func (s *FakeCharacterStore) ListCharactersByOwnerParticipant(_ context.Context, campaignID, participantID string) ([]storage.CharacterRecord, error) {
	if s.ListErr != nil {
		return nil, s.ListErr
	}
	s.ListByOwnerCalls++
	s.LastOwnerCampaignID = campaignID
	s.LastOwnerID = participantID

	byID, ok := s.Characters[campaignID]
	if !ok {
		return nil, nil
	}
	result := make([]storage.CharacterRecord, 0, len(byID))
	for _, c := range byID {
		if c.OwnerParticipantID == participantID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (s *FakeCharacterStore) ListCharactersByControllerParticipant(_ context.Context, campaignID, participantID string) ([]storage.CharacterRecord, error) {
	if s.ListErr != nil {
		return nil, s.ListErr
	}

	byID, ok := s.Characters[campaignID]
	if !ok {
		return nil, nil
	}
	result := make([]storage.CharacterRecord, 0, len(byID))
	for _, c := range byID {
		if c.ParticipantID == participantID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (s *FakeCharacterStore) ListCharacters(_ context.Context, campaignID string, pageSize int, pageToken string) (storage.CharacterPage, error) {
	if s.ListErr != nil {
		return storage.CharacterPage{}, s.ListErr
	}
	s.ListCalls++
	s.LastListCampaignID = campaignID
	byID, ok := s.Characters[campaignID]
	if !ok {
		return storage.CharacterPage{}, nil
	}
	result := make([]storage.CharacterRecord, 0, len(byID))
	for _, c := range byID {
		result = append(result, c)
	}
	return storage.CharacterPage{
		Characters:    result,
		NextPageToken: "",
	}, nil
}

func (s *FakeCharacterStore) CountCharacters(_ context.Context, campaignID string) (int, error) {
	byID, ok := s.Characters[campaignID]
	if !ok {
		return 0, nil
	}
	return len(byID), nil
}
