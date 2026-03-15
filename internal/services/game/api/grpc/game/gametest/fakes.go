package gametest

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// FakeCampaignStore is a test double for storage.CampaignStore.
type FakeCampaignStore struct {
	Campaigns map[string]storage.CampaignRecord
	PutErr    error
	GetErr    error
	ListErr   error
}

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

// FakeInviteStore is a test double for storage.InviteStore.
type FakeInviteStore struct {
	Invites   map[string]storage.InviteRecord
	PutErr    error
	GetErr    error
	ListErr   error
	UpdateErr error
}

func NewFakeInviteStore() *FakeInviteStore {
	return &FakeInviteStore{Invites: make(map[string]storage.InviteRecord)}
}

func (s *FakeInviteStore) PutInvite(_ context.Context, inv storage.InviteRecord) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	s.Invites[inv.ID] = inv
	return nil
}

func (s *FakeInviteStore) GetInvite(_ context.Context, inviteID string) (storage.InviteRecord, error) {
	if s.GetErr != nil {
		return storage.InviteRecord{}, s.GetErr
	}
	inv, ok := s.Invites[inviteID]
	if !ok {
		return storage.InviteRecord{}, storage.ErrNotFound
	}
	return inv, nil
}

func (s *FakeInviteStore) ListInvites(_ context.Context, campaignID string, recipientUserID string, status invite.Status, pageSize int, pageToken string) (storage.InvitePage, error) {
	if s.ListErr != nil {
		return storage.InvitePage{}, s.ListErr
	}
	result := make([]storage.InviteRecord, 0)
	for _, inv := range s.Invites {
		if inv.CampaignID != campaignID {
			continue
		}
		if recipientUserID != "" && inv.RecipientUserID != recipientUserID {
			continue
		}
		if status != invite.StatusUnspecified && inv.Status != status {
			continue
		}
		result = append(result, inv)
	}
	return storage.InvitePage{Invites: result, NextPageToken: ""}, nil
}

func (s *FakeInviteStore) ListPendingInvites(_ context.Context, campaignID string, pageSize int, pageToken string) (storage.InvitePage, error) {
	if s.ListErr != nil {
		return storage.InvitePage{}, s.ListErr
	}
	result := make([]storage.InviteRecord, 0)
	for _, inv := range s.Invites {
		if inv.CampaignID == campaignID && inv.Status == invite.StatusPending {
			result = append(result, inv)
		}
	}
	return storage.InvitePage{Invites: result, NextPageToken: ""}, nil
}

func (s *FakeInviteStore) ListPendingInvitesForRecipient(_ context.Context, userID string, pageSize int, pageToken string) (storage.InvitePage, error) {
	if s.ListErr != nil {
		return storage.InvitePage{}, s.ListErr
	}
	result := make([]storage.InviteRecord, 0)
	for _, inv := range s.Invites {
		if inv.RecipientUserID == userID && inv.Status == invite.StatusPending {
			result = append(result, inv)
		}
	}
	return storage.InvitePage{Invites: result, NextPageToken: ""}, nil
}

func (s *FakeInviteStore) UpdateInviteStatus(_ context.Context, inviteID string, status invite.Status, updatedAt time.Time) error {
	if s.UpdateErr != nil {
		return s.UpdateErr
	}
	inv, ok := s.Invites[inviteID]
	if !ok {
		return storage.ErrNotFound
	}
	inv.Status = status
	inv.UpdatedAt = updatedAt
	s.Invites[inviteID] = inv
	return nil
}

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

// FakeDaggerheartStore is a test double for projectionstore.Store.
type FakeDaggerheartStore struct {
	Profiles    map[string]map[string]projectionstore.DaggerheartCharacterProfile // campaignID -> characterID -> Profile
	States      map[string]map[string]projectionstore.DaggerheartCharacterState   // campaignID -> characterID -> state
	Snapshots   map[string]projectionstore.DaggerheartSnapshot                    // campaignID -> snapshot
	Countdowns  map[string]map[string]projectionstore.DaggerheartCountdown        // campaignID -> countdownID -> countdown
	Adversaries map[string]map[string]projectionstore.DaggerheartAdversary        // campaignID -> adversaryID -> adversary
	StatePuts   map[string]int
	SnapPuts    map[string]int
	PutErr      error
	GetErr      error
}

func NewFakeDaggerheartStore() *FakeDaggerheartStore {
	return &FakeDaggerheartStore{
		Profiles:    make(map[string]map[string]projectionstore.DaggerheartCharacterProfile),
		States:      make(map[string]map[string]projectionstore.DaggerheartCharacterState),
		Snapshots:   make(map[string]projectionstore.DaggerheartSnapshot),
		Countdowns:  make(map[string]map[string]projectionstore.DaggerheartCountdown),
		Adversaries: make(map[string]map[string]projectionstore.DaggerheartAdversary),
		StatePuts:   make(map[string]int),
		SnapPuts:    make(map[string]int),
	}
}

func (s *FakeDaggerheartStore) PutDaggerheartCharacterProfile(_ context.Context, p projectionstore.DaggerheartCharacterProfile) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	if s.Profiles[p.CampaignID] == nil {
		s.Profiles[p.CampaignID] = make(map[string]projectionstore.DaggerheartCharacterProfile)
	}
	s.Profiles[p.CampaignID][p.CharacterID] = p
	return nil
}

func (s *FakeDaggerheartStore) GetDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterProfile, error) {
	if s.GetErr != nil {
		return projectionstore.DaggerheartCharacterProfile{}, s.GetErr
	}
	byID, ok := s.Profiles[campaignID]
	if !ok {
		return projectionstore.DaggerheartCharacterProfile{}, storage.ErrNotFound
	}
	p, ok := byID[characterID]
	if !ok {
		return projectionstore.DaggerheartCharacterProfile{}, storage.ErrNotFound
	}
	return p, nil
}

func (s *FakeDaggerheartStore) ListDaggerheartCharacterProfiles(_ context.Context, campaignID string, pageSize int, pageToken string) (projectionstore.DaggerheartCharacterProfilePage, error) {
	if s.GetErr != nil {
		return projectionstore.DaggerheartCharacterProfilePage{}, s.GetErr
	}
	if pageSize <= 0 {
		return projectionstore.DaggerheartCharacterProfilePage{}, fmt.Errorf("page size must be greater than zero")
	}
	byID, ok := s.Profiles[campaignID]
	if !ok {
		return projectionstore.DaggerheartCharacterProfilePage{}, nil
	}
	ids := make([]string, 0, len(byID))
	for characterID := range byID {
		if pageToken != "" && characterID <= pageToken {
			continue
		}
		ids = append(ids, characterID)
	}
	sort.Strings(ids)

	page := projectionstore.DaggerheartCharacterProfilePage{
		Profiles: make([]projectionstore.DaggerheartCharacterProfile, 0, min(pageSize, len(ids))),
	}
	for index, characterID := range ids {
		if index >= pageSize {
			page.NextPageToken = ids[pageSize-1]
			break
		}
		page.Profiles = append(page.Profiles, byID[characterID])
	}
	return page, nil
}

func (s *FakeDaggerheartStore) DeleteDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	byID, ok := s.Profiles[campaignID]
	if !ok {
		return nil
	}
	delete(byID, characterID)
	if len(byID) == 0 {
		delete(s.Profiles, campaignID)
	}
	return nil
}

func (s *FakeDaggerheartStore) PutDaggerheartCharacterState(_ context.Context, st projectionstore.DaggerheartCharacterState) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	if s.States[st.CampaignID] == nil {
		s.States[st.CampaignID] = make(map[string]projectionstore.DaggerheartCharacterState)
	}
	s.States[st.CampaignID][st.CharacterID] = st
	s.StatePuts[st.CampaignID]++
	return nil
}

func (s *FakeDaggerheartStore) GetDaggerheartCharacterState(_ context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error) {
	if s.GetErr != nil {
		return projectionstore.DaggerheartCharacterState{}, s.GetErr
	}
	byID, ok := s.States[campaignID]
	if !ok {
		return projectionstore.DaggerheartCharacterState{}, storage.ErrNotFound
	}
	st, ok := byID[characterID]
	if !ok {
		return projectionstore.DaggerheartCharacterState{}, storage.ErrNotFound
	}
	return st, nil
}

func (s *FakeDaggerheartStore) PutDaggerheartSnapshot(_ context.Context, snap projectionstore.DaggerheartSnapshot) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	s.Snapshots[snap.CampaignID] = snap
	s.SnapPuts[snap.CampaignID]++
	return nil
}

func (s *FakeDaggerheartStore) GetDaggerheartSnapshot(_ context.Context, campaignID string) (projectionstore.DaggerheartSnapshot, error) {
	if s.GetErr != nil {
		return projectionstore.DaggerheartSnapshot{}, s.GetErr
	}
	snap, ok := s.Snapshots[campaignID]
	if !ok {
		return projectionstore.DaggerheartSnapshot{}, storage.ErrNotFound
	}
	return snap, nil
}

func (s *FakeDaggerheartStore) PutDaggerheartCountdown(_ context.Context, countdown projectionstore.DaggerheartCountdown) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	if s.Countdowns[countdown.CampaignID] == nil {
		s.Countdowns[countdown.CampaignID] = make(map[string]projectionstore.DaggerheartCountdown)
	}
	s.Countdowns[countdown.CampaignID][countdown.CountdownID] = countdown
	return nil
}

func (s *FakeDaggerheartStore) GetDaggerheartCountdown(_ context.Context, campaignID, countdownID string) (projectionstore.DaggerheartCountdown, error) {
	if s.GetErr != nil {
		return projectionstore.DaggerheartCountdown{}, s.GetErr
	}
	byID, ok := s.Countdowns[campaignID]
	if !ok {
		return projectionstore.DaggerheartCountdown{}, storage.ErrNotFound
	}
	countdown, ok := byID[countdownID]
	if !ok {
		return projectionstore.DaggerheartCountdown{}, storage.ErrNotFound
	}
	return countdown, nil
}

func (s *FakeDaggerheartStore) ListDaggerheartCountdowns(_ context.Context, campaignID string) ([]projectionstore.DaggerheartCountdown, error) {
	if s.GetErr != nil {
		return nil, s.GetErr
	}
	byID, ok := s.Countdowns[campaignID]
	if !ok {
		return nil, nil
	}
	result := make([]projectionstore.DaggerheartCountdown, 0, len(byID))
	for _, countdown := range byID {
		result = append(result, countdown)
	}
	return result, nil
}

func (s *FakeDaggerheartStore) DeleteDaggerheartCountdown(_ context.Context, campaignID, countdownID string) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	byID, ok := s.Countdowns[campaignID]
	if !ok {
		return storage.ErrNotFound
	}
	if _, ok := byID[countdownID]; !ok {
		return storage.ErrNotFound
	}
	delete(byID, countdownID)
	return nil
}

func (s *FakeDaggerheartStore) PutDaggerheartAdversary(_ context.Context, adversary projectionstore.DaggerheartAdversary) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	if s.Adversaries[adversary.CampaignID] == nil {
		s.Adversaries[adversary.CampaignID] = make(map[string]projectionstore.DaggerheartAdversary)
	}
	s.Adversaries[adversary.CampaignID][adversary.AdversaryID] = adversary
	return nil
}

func (s *FakeDaggerheartStore) GetDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
	if s.GetErr != nil {
		return projectionstore.DaggerheartAdversary{}, s.GetErr
	}
	byID, ok := s.Adversaries[campaignID]
	if !ok {
		return projectionstore.DaggerheartAdversary{}, storage.ErrNotFound
	}
	adversary, ok := byID[adversaryID]
	if !ok {
		return projectionstore.DaggerheartAdversary{}, storage.ErrNotFound
	}
	return adversary, nil
}

func (s *FakeDaggerheartStore) ListDaggerheartAdversaries(_ context.Context, campaignID, sessionID string) ([]projectionstore.DaggerheartAdversary, error) {
	if s.GetErr != nil {
		return nil, s.GetErr
	}
	byID, ok := s.Adversaries[campaignID]
	if !ok {
		return nil, nil
	}
	result := make([]projectionstore.DaggerheartAdversary, 0, len(byID))
	for _, adversary := range byID {
		if strings.TrimSpace(sessionID) != "" && adversary.SessionID != sessionID {
			continue
		}
		result = append(result, adversary)
	}
	return result, nil
}

func (s *FakeDaggerheartStore) DeleteDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	byID, ok := s.Adversaries[campaignID]
	if !ok {
		return storage.ErrNotFound
	}
	if _, ok := byID[adversaryID]; !ok {
		return storage.ErrNotFound
	}
	delete(byID, adversaryID)
	return nil
}

// FakeSessionGateStore is a test double for storage.SessionGateStore.
type FakeSessionGateStore struct {
	Gates  map[string]storage.SessionGate
	PutErr error
	GetErr error
}

func NewFakeSessionGateStore() *FakeSessionGateStore {
	return &FakeSessionGateStore{Gates: make(map[string]storage.SessionGate)}
}

func (s *FakeSessionGateStore) PutSessionGate(_ context.Context, gate storage.SessionGate) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	key := gate.CampaignID + ":" + gate.SessionID + ":" + gate.GateID
	s.Gates[key] = gate
	return nil
}

func (s *FakeSessionGateStore) GetSessionGate(_ context.Context, campaignID, sessionID, gateID string) (storage.SessionGate, error) {
	if s.GetErr != nil {
		return storage.SessionGate{}, s.GetErr
	}
	key := campaignID + ":" + sessionID + ":" + gateID
	gate, ok := s.Gates[key]
	if !ok {
		return storage.SessionGate{}, storage.ErrNotFound
	}
	return gate, nil
}

func (s *FakeSessionGateStore) GetOpenSessionGate(_ context.Context, campaignID, sessionID string) (storage.SessionGate, error) {
	if s.GetErr != nil {
		return storage.SessionGate{}, s.GetErr
	}
	for _, gate := range s.Gates {
		if gate.CampaignID == campaignID && gate.SessionID == sessionID && gate.Status == session.GateStatusOpen {
			return gate, nil
		}
	}
	return storage.SessionGate{}, storage.ErrNotFound
}

// FakeSessionStore is a test double for storage.SessionStore.
type FakeSessionStore struct {
	Sessions      map[string]map[string]storage.SessionRecord // campaignID -> sessionID -> Session
	ActiveSession map[string]string                           // campaignID -> sessionID (active session ID)
	PutErr        error
	GetErr        error
	EndErr        error
	ActiveErr     error
	ListErr       error
}

// FakeSessionSpotlightStore is a test double for storage.SessionSpotlightStore.
type FakeSessionSpotlightStore struct {
	Spotlights map[string]map[string]storage.SessionSpotlight
	PutErr     error
	GetErr     error
	ClearErr   error
}

func NewFakeSessionSpotlightStore() *FakeSessionSpotlightStore {
	return &FakeSessionSpotlightStore{
		Spotlights: make(map[string]map[string]storage.SessionSpotlight),
	}
}

func (s *FakeSessionSpotlightStore) PutSessionSpotlight(_ context.Context, spotlight storage.SessionSpotlight) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	if s.Spotlights[spotlight.CampaignID] == nil {
		s.Spotlights[spotlight.CampaignID] = make(map[string]storage.SessionSpotlight)
	}
	s.Spotlights[spotlight.CampaignID][spotlight.SessionID] = spotlight
	return nil
}

func (s *FakeSessionSpotlightStore) GetSessionSpotlight(_ context.Context, campaignID, sessionID string) (storage.SessionSpotlight, error) {
	if s.GetErr != nil {
		return storage.SessionSpotlight{}, s.GetErr
	}
	bySession, ok := s.Spotlights[campaignID]
	if !ok {
		return storage.SessionSpotlight{}, storage.ErrNotFound
	}
	spotlight, ok := bySession[sessionID]
	if !ok {
		return storage.SessionSpotlight{}, storage.ErrNotFound
	}
	return spotlight, nil
}

func (s *FakeSessionSpotlightStore) ClearSessionSpotlight(_ context.Context, campaignID, sessionID string) error {
	if s.ClearErr != nil {
		return s.ClearErr
	}
	bySession, ok := s.Spotlights[campaignID]
	if !ok {
		return storage.ErrNotFound
	}
	if _, ok := bySession[sessionID]; !ok {
		return storage.ErrNotFound
	}
	delete(bySession, sessionID)
	return nil
}

func NewFakeSessionStore() *FakeSessionStore {
	return &FakeSessionStore{
		Sessions:      make(map[string]map[string]storage.SessionRecord),
		ActiveSession: make(map[string]string),
	}
}

func (s *FakeSessionStore) PutSession(_ context.Context, sess storage.SessionRecord) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	// Check for active session
	if activeID, ok := s.ActiveSession[sess.CampaignID]; ok && activeID != "" {
		return storage.ErrActiveSessionExists
	}
	if s.Sessions[sess.CampaignID] == nil {
		s.Sessions[sess.CampaignID] = make(map[string]storage.SessionRecord)
	}
	s.Sessions[sess.CampaignID][sess.ID] = sess
	if sess.Status == session.StatusActive {
		s.ActiveSession[sess.CampaignID] = sess.ID
	}
	return nil
}

func (s *FakeSessionStore) EndSession(_ context.Context, campaignID, sessionID string, endedAt time.Time) (storage.SessionRecord, bool, error) {
	if s.EndErr != nil {
		return storage.SessionRecord{}, false, s.EndErr
	}
	byID, ok := s.Sessions[campaignID]
	if !ok {
		return storage.SessionRecord{}, false, storage.ErrNotFound
	}
	sess, ok := byID[sessionID]
	if !ok {
		return storage.SessionRecord{}, false, storage.ErrNotFound
	}
	if sess.Status == session.StatusEnded {
		return sess, false, nil
	}
	sess.Status = session.StatusEnded
	sess.EndedAt = &endedAt
	sess.UpdatedAt = endedAt
	s.Sessions[campaignID][sessionID] = sess
	if s.ActiveSession[campaignID] == sessionID {
		s.ActiveSession[campaignID] = ""
	}
	return sess, true, nil
}

func (s *FakeSessionStore) GetSession(_ context.Context, campaignID, sessionID string) (storage.SessionRecord, error) {
	if s.GetErr != nil {
		return storage.SessionRecord{}, s.GetErr
	}
	byID, ok := s.Sessions[campaignID]
	if !ok {
		return storage.SessionRecord{}, storage.ErrNotFound
	}
	sess, ok := byID[sessionID]
	if !ok {
		return storage.SessionRecord{}, storage.ErrNotFound
	}
	return sess, nil
}

func (s *FakeSessionStore) GetActiveSession(_ context.Context, campaignID string) (storage.SessionRecord, error) {
	if s.ActiveErr != nil {
		return storage.SessionRecord{}, s.ActiveErr
	}
	activeID, ok := s.ActiveSession[campaignID]
	if !ok || activeID == "" {
		return storage.SessionRecord{}, storage.ErrNotFound
	}
	byID := s.Sessions[campaignID]
	sess, ok := byID[activeID]
	if !ok {
		return storage.SessionRecord{}, storage.ErrNotFound
	}
	return sess, nil
}

func (s *FakeSessionStore) ListSessions(_ context.Context, campaignID string, pageSize int, pageToken string) (storage.SessionPage, error) {
	if s.ListErr != nil {
		return storage.SessionPage{}, s.ListErr
	}
	byID, ok := s.Sessions[campaignID]
	if !ok {
		return storage.SessionPage{}, nil
	}
	result := make([]storage.SessionRecord, 0, len(byID))
	for _, sess := range byID {
		result = append(result, sess)
	}
	return storage.SessionPage{
		Sessions:      result,
		NextPageToken: "",
	}, nil
}

// FakeSessionInteractionStore is a test double for storage.SessionInteractionStore.
type FakeSessionInteractionStore struct {
	Values map[string]storage.SessionInteraction // "campaignID:sessionID" -> interaction
}

// NewFakeSessionInteractionStore returns a ready-to-use FakeSessionInteractionStore.
func NewFakeSessionInteractionStore() *FakeSessionInteractionStore {
	return &FakeSessionInteractionStore{Values: make(map[string]storage.SessionInteraction)}
}

func (s *FakeSessionInteractionStore) GetSessionInteraction(_ context.Context, campaignID, sessionID string) (storage.SessionInteraction, error) {
	if s == nil || s.Values == nil {
		return storage.SessionInteraction{}, storage.ErrNotFound
	}
	value, ok := s.Values[campaignID+":"+sessionID]
	if !ok {
		return storage.SessionInteraction{}, storage.ErrNotFound
	}
	return value, nil
}

func (s *FakeSessionInteractionStore) PutSessionInteraction(_ context.Context, interaction storage.SessionInteraction) error {
	if s.Values == nil {
		s.Values = make(map[string]storage.SessionInteraction)
	}
	s.Values[interaction.CampaignID+":"+interaction.SessionID] = interaction
	return nil
}

// FakeEventStore is a test double for storage.EventStore.
type FakeEventStore struct {
	Events    map[string][]event.Event // campaignID -> Events
	ByHash    map[string]event.Event   // hash -> event
	AppendErr error
	ListErr   error
	GetErr    error
	NextSeq   map[string]uint64 // campaignID -> NextSeq
}

func NewFakeEventStore() *FakeEventStore {
	return &FakeEventStore{
		Events:  make(map[string][]event.Event),
		ByHash:  make(map[string]event.Event),
		NextSeq: make(map[string]uint64),
	}
}

func (s *FakeEventStore) AppendEvent(_ context.Context, evt event.Event) (event.Event, error) {
	if s.AppendErr != nil {
		return event.Event{}, s.AppendErr
	}
	cid := string(evt.CampaignID)
	seq := s.NextSeq[cid]
	if seq == 0 {
		seq = 1
	}
	evt.Seq = seq
	evt.Hash = "fakehash-" + cid + "-" + string(rune('0'+seq))
	s.NextSeq[cid] = seq + 1
	s.Events[cid] = append(s.Events[cid], evt)
	s.ByHash[evt.Hash] = evt
	return evt, nil
}

type FakeBatchEventStore struct {
	*FakeEventStore
}

func NewFakeBatchEventStore() *FakeBatchEventStore {
	return &FakeBatchEventStore{FakeEventStore: NewFakeEventStore()}
}

func (s *FakeBatchEventStore) BatchAppendEvents(ctx context.Context, events []event.Event) ([]event.Event, error) {
	if s.AppendErr != nil {
		return nil, s.AppendErr
	}
	stored := make([]event.Event, 0, len(events))
	for _, evt := range events {
		storedEvent, err := s.AppendEvent(ctx, evt)
		if err != nil {
			return nil, err
		}
		stored = append(stored, storedEvent)
	}
	return stored, nil
}

func (s *FakeEventStore) GetEventByHash(_ context.Context, hash string) (event.Event, error) {
	if s.GetErr != nil {
		return event.Event{}, s.GetErr
	}
	evt, ok := s.ByHash[hash]
	if !ok {
		return event.Event{}, storage.ErrNotFound
	}
	return evt, nil
}

func (s *FakeEventStore) GetEventBySeq(_ context.Context, campaignID string, seq uint64) (event.Event, error) {
	if s.GetErr != nil {
		return event.Event{}, s.GetErr
	}
	events, ok := s.Events[campaignID]
	if !ok {
		return event.Event{}, storage.ErrNotFound
	}
	for _, evt := range events {
		if evt.Seq == seq {
			return evt, nil
		}
	}
	return event.Event{}, storage.ErrNotFound
}

func (s *FakeEventStore) ListEvents(_ context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error) {
	if s.ListErr != nil {
		return nil, s.ListErr
	}
	events, ok := s.Events[campaignID]
	if !ok {
		return nil, nil
	}
	var result []event.Event
	for _, e := range events {
		if e.Seq > afterSeq {
			result = append(result, e)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (s *FakeEventStore) ListEventsBySession(_ context.Context, campaignID, sessionID string, afterSeq uint64, limit int) ([]event.Event, error) {
	if s.ListErr != nil {
		return nil, s.ListErr
	}
	events, ok := s.Events[campaignID]
	if !ok {
		return nil, nil
	}
	var result []event.Event
	for _, e := range events {
		if e.SessionID.String() == sessionID && e.Seq > afterSeq {
			result = append(result, e)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (s *FakeEventStore) GetLatestEventSeq(_ context.Context, campaignID string) (uint64, error) {
	if s.GetErr != nil {
		return 0, s.GetErr
	}
	seq := s.NextSeq[campaignID]
	if seq == 0 {
		return 0, nil
	}
	return seq - 1, nil
}

func (s *FakeEventStore) ListEventsPage(_ context.Context, req storage.ListEventsPageRequest) (storage.ListEventsPageResult, error) {
	if s.ListErr != nil {
		return storage.ListEventsPageResult{}, s.ListErr
	}
	events, ok := s.Events[req.CampaignID]
	if !ok {
		return storage.ListEventsPageResult{TotalCount: 0}, nil
	}

	// Copy Events for sorting
	sorted := make([]event.Event, len(events))
	copy(sorted, events)

	// Apply sort order (DESC reverses the base order)
	// For "previous page" navigation, we also temporarily reverse to grab from the near edge
	needsReverse := req.Descending
	if req.CursorReverse {
		needsReverse = !needsReverse
	}
	if needsReverse {
		for i, j := 0, len(sorted)-1; i < j; i, j = i+1, j-1 {
			sorted[i], sorted[j] = sorted[j], sorted[i]
		}
	}

	base := make([]event.Event, 0, len(sorted))
	for _, e := range sorted {
		if req.AfterSeq > 0 && e.Seq <= req.AfterSeq {
			continue
		}
		if filter := req.Filter; filter.EventType != "" && string(e.Type) != filter.EventType {
			continue
		}
		if filter := req.Filter; filter.EntityType != "" && e.EntityType != filter.EntityType {
			continue
		}
		if filter := req.Filter; filter.EntityID != "" && e.EntityID != filter.EntityID {
			continue
		}
		base = append(base, e)
	}

	// Apply cursor filter
	// The cursor direction directly determines the comparison:
	// - Forward (fwd): seq > cursor
	// - Backward (bwd): seq < cursor
	var filtered []event.Event
	for _, e := range base {
		if req.CursorSeq > 0 {
			if req.CursorDir == "bwd" {
				if e.Seq >= req.CursorSeq {
					continue
				}
			} else {
				if e.Seq <= req.CursorSeq {
					continue
				}
			}
		}
		filtered = append(filtered, e)
	}

	// Paginate
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}

	var result []event.Event
	hasMore := false
	if len(filtered) > pageSize {
		result = filtered[:pageSize]
		hasMore = true
	} else {
		result = filtered
	}

	// For "previous page" navigation, reverse to maintain consistent display order
	if req.CursorReverse {
		for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
			result[i], result[j] = result[j], result[i]
		}
	}

	// Determine hasPrev/hasNext based on pagination direction
	var hasNextPage, hasPrevPage bool
	if req.CursorReverse {
		hasNextPage = true // We came from next, so there is a next
		hasPrevPage = hasMore
	} else {
		hasNextPage = hasMore
		hasPrevPage = req.CursorSeq > 0
	}

	return storage.ListEventsPageResult{
		Events:      result,
		HasNextPage: hasNextPage,
		HasPrevPage: hasPrevPage,
		TotalCount:  len(base),
	}, nil
}

type FakeAuthClient struct {
	User               *authv1.User
	GetUserErr         error
	LastGetUserRequest *authv1.GetUserRequest
}

type FakeSocialClient struct {
	Profile               *socialv1.UserProfile
	GetUserProfileErr     error
	LastGetUserProfileReq *socialv1.GetUserProfileRequest
	GetUserProfileCalls   int
}

func (f *FakeAuthClient) IssueJoinGrant(ctx context.Context, req *authv1.IssueJoinGrantRequest, opts ...grpc.CallOption) (*authv1.IssueJoinGrantResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) BeginAccountRegistration(ctx context.Context, req *authv1.BeginAccountRegistrationRequest, opts ...grpc.CallOption) (*authv1.BeginAccountRegistrationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) FinishAccountRegistration(ctx context.Context, req *authv1.FinishAccountRegistrationRequest, opts ...grpc.CallOption) (*authv1.FinishAccountRegistrationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) AcknowledgeAccountRegistration(ctx context.Context, req *authv1.AcknowledgeAccountRegistrationRequest, opts ...grpc.CallOption) (*authv1.AcknowledgeAccountRegistrationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) CheckUsernameAvailability(ctx context.Context, req *authv1.CheckUsernameAvailabilityRequest, opts ...grpc.CallOption) (*authv1.CheckUsernameAvailabilityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) GetUser(ctx context.Context, req *authv1.GetUserRequest, opts ...grpc.CallOption) (*authv1.GetUserResponse, error) {
	f.LastGetUserRequest = req
	if f.GetUserErr != nil {
		return nil, f.GetUserErr
	}
	return &authv1.GetUserResponse{User: f.User}, nil
}

func (f *FakeAuthClient) ListUsers(ctx context.Context, req *authv1.ListUsersRequest, opts ...grpc.CallOption) (*authv1.ListUsersResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) LeaseIntegrationOutboxEvents(ctx context.Context, req *authv1.LeaseIntegrationOutboxEventsRequest, opts ...grpc.CallOption) (*authv1.LeaseIntegrationOutboxEventsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) AckIntegrationOutboxEvent(ctx context.Context, req *authv1.AckIntegrationOutboxEventRequest, opts ...grpc.CallOption) (*authv1.AckIntegrationOutboxEventResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) BeginPasskeyRegistration(ctx context.Context, req *authv1.BeginPasskeyRegistrationRequest, opts ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) FinishPasskeyRegistration(ctx context.Context, req *authv1.FinishPasskeyRegistrationRequest, opts ...grpc.CallOption) (*authv1.FinishPasskeyRegistrationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) BeginPasskeyLogin(ctx context.Context, req *authv1.BeginPasskeyLoginRequest, opts ...grpc.CallOption) (*authv1.BeginPasskeyLoginResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) FinishPasskeyLogin(ctx context.Context, req *authv1.FinishPasskeyLoginRequest, opts ...grpc.CallOption) (*authv1.FinishPasskeyLoginResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) BeginAccountRecovery(ctx context.Context, req *authv1.BeginAccountRecoveryRequest, opts ...grpc.CallOption) (*authv1.BeginAccountRecoveryResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) BeginRecoveryPasskeyRegistration(ctx context.Context, req *authv1.BeginRecoveryPasskeyRegistrationRequest, opts ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) FinishRecoveryPasskeyRegistration(ctx context.Context, req *authv1.FinishRecoveryPasskeyRegistrationRequest, opts ...grpc.CallOption) (*authv1.FinishAccountRegistrationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) CreateWebSession(ctx context.Context, req *authv1.CreateWebSessionRequest, opts ...grpc.CallOption) (*authv1.CreateWebSessionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) GetWebSession(ctx context.Context, req *authv1.GetWebSessionRequest, opts ...grpc.CallOption) (*authv1.GetWebSessionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) RevokeWebSession(ctx context.Context, req *authv1.RevokeWebSessionRequest, opts ...grpc.CallOption) (*authv1.RevokeWebSessionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) ListPasskeys(ctx context.Context, req *authv1.ListPasskeysRequest, opts ...grpc.CallOption) (*authv1.ListPasskeysResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeAuthClient) LookupUserByUsername(ctx context.Context, req *authv1.LookupUserByUsernameRequest, opts ...grpc.CallOption) (*authv1.LookupUserByUsernameResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *FakeSocialClient) AddContact(context.Context, *socialv1.AddContactRequest, ...grpc.CallOption) (*socialv1.AddContactResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake social client")
}

func (f *FakeSocialClient) RemoveContact(context.Context, *socialv1.RemoveContactRequest, ...grpc.CallOption) (*socialv1.RemoveContactResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake social client")
}

func (f *FakeSocialClient) ListContacts(context.Context, *socialv1.ListContactsRequest, ...grpc.CallOption) (*socialv1.ListContactsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake social client")
}

func (f *FakeSocialClient) SearchUsers(context.Context, *socialv1.SearchUsersRequest, ...grpc.CallOption) (*socialv1.SearchUsersResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake social client")
}

func (f *FakeSocialClient) SyncDirectoryUser(context.Context, *socialv1.SyncDirectoryUserRequest, ...grpc.CallOption) (*socialv1.SyncDirectoryUserResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake social client")
}

func (f *FakeSocialClient) SetUserProfile(context.Context, *socialv1.SetUserProfileRequest, ...grpc.CallOption) (*socialv1.SetUserProfileResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake social client")
}

func (f *FakeSocialClient) GetUserProfile(_ context.Context, req *socialv1.GetUserProfileRequest, _ ...grpc.CallOption) (*socialv1.GetUserProfileResponse, error) {
	f.GetUserProfileCalls++
	f.LastGetUserProfileReq = req
	if f.GetUserProfileErr != nil {
		return nil, f.GetUserProfileErr
	}
	if f.Profile == nil {
		return &socialv1.GetUserProfileResponse{}, nil
	}
	return &socialv1.GetUserProfileResponse{UserProfile: f.Profile}, nil
}

type FakeStatisticsStore struct {
	LastSince *time.Time
	Stats     storage.GameStatistics
	Err       error
}

func (f *FakeStatisticsStore) GetGameStatistics(_ context.Context, since *time.Time) (storage.GameStatistics, error) {
	f.LastSince = since
	return f.Stats, f.Err
}

type FakeCampaignForkStore struct {
	Metadata map[string]storage.ForkMetadata
	GetErr   error
	SetErr   error
}

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

// Test helper functions

func FixedClock(t time.Time) func() time.Time {
	return func() time.Time {
		return t
	}
}

func FixedIDGenerator(id string) func() (string, error) {
	return func() (string, error) {
		return id, nil
	}
}

func FixedSequenceIDGenerator(ids ...string) func() (string, error) {
	index := 0
	return func() (string, error) {
		if index >= len(ids) {
			return ids[len(ids)-1], nil
		}
		id := ids[index]
		index++
		return id, nil
	}
}

func SequentialIDGenerator(prefix string) func() (string, error) {
	counter := 0
	return func() (string, error) {
		counter++
		return prefix + "-" + string(rune('0'+counter)), nil
	}
}

func ContextWithParticipantID(participantID string) context.Context {
	if participantID == "" {
		return context.Background()
	}
	md := metadata.Pairs(grpcmeta.ParticipantIDHeader, participantID)
	return metadata.NewIncomingContext(context.Background(), md)
}

func ContextWithUserID(userID string) context.Context {
	if userID == "" {
		return context.Background()
	}
	md := metadata.Pairs(grpcmeta.UserIDHeader, userID)
	return metadata.NewIncomingContext(context.Background(), md)
}

func ContextWithAdminOverride(reason string) context.Context {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "test-override"
	}
	md := metadata.Pairs(
		grpcmeta.PlatformRoleHeader, grpcmeta.PlatformRoleAdmin,
		grpcmeta.AuthzOverrideReasonHeader, reason,
		grpcmeta.UserIDHeader, "user-admin-test",
	)
	return metadata.NewIncomingContext(context.Background(), md)
}

type JoinGrantSigner struct {
	Issuer   string
	Audience string
	Key      ed25519.PrivateKey
}

func NewJoinGrantSigner(t *testing.T) JoinGrantSigner {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate join grant key: %v", err)
	}
	issuer := "test-issuer"
	audience := "game-service"
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", issuer)
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", audience)
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY", base64.RawStdEncoding.EncodeToString(publicKey))
	return JoinGrantSigner{
		Issuer:   issuer,
		Audience: audience,
		Key:      privateKey,
	}
}

func (s JoinGrantSigner) Token(t *testing.T, campaignID, inviteID, userID, jti string, now time.Time) string {
	t.Helper()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if s.Key == nil {
		t.Fatal("join grant signer key is required")
	}
	if strings.TrimSpace(jti) == "" {
		jti = fmt.Sprintf("jti-%d", now.UnixNano())
	}
	headerJSON, err := json.Marshal(map[string]string{
		"alg": "EdDSA",
		"typ": "JWT",
	})
	if err != nil {
		t.Fatalf("encode join grant header: %v", err)
	}
	payloadJSON, err := json.Marshal(map[string]any{
		"iss":         s.Issuer,
		"aud":         s.Audience,
		"exp":         now.Add(5 * time.Minute).Unix(),
		"iat":         now.Unix(),
		"jti":         jti,
		"campaign_id": strings.TrimSpace(campaignID),
		"invite_id":   strings.TrimSpace(inviteID),
		"user_id":     strings.TrimSpace(userID),
	})
	if err != nil {
		t.Fatalf("encode join grant payload: %v", err)
	}
	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := encodedHeader + "." + encodedPayload
	signature := ed25519.Sign(s.Key, []byte(signingInput))
	encodedSig := base64.RawURLEncoding.EncodeToString(signature)
	return signingInput + "." + encodedSig
}
