package game

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// fakeCampaignStore is a test double for storage.CampaignStore.
type fakeCampaignStore struct {
	campaigns map[string]storage.CampaignRecord
	putErr    error
	getErr    error
	listErr   error
}

func newFakeCampaignStore() *fakeCampaignStore {
	return &fakeCampaignStore{
		campaigns: make(map[string]storage.CampaignRecord),
	}
}

func (s *fakeCampaignStore) Put(_ context.Context, c storage.CampaignRecord) error {
	if s.putErr != nil {
		return s.putErr
	}
	s.campaigns[c.ID] = c
	return nil
}

func (s *fakeCampaignStore) Get(_ context.Context, id string) (storage.CampaignRecord, error) {
	if s.getErr != nil {
		return storage.CampaignRecord{}, s.getErr
	}
	c, ok := s.campaigns[id]
	if !ok {
		return storage.CampaignRecord{}, storage.ErrNotFound
	}
	return c, nil
}

func (s *fakeCampaignStore) List(_ context.Context, pageSize int, pageToken string) (storage.CampaignPage, error) {
	if s.listErr != nil {
		return storage.CampaignPage{}, s.listErr
	}
	campaigns := make([]storage.CampaignRecord, 0, len(s.campaigns))
	for _, c := range s.campaigns {
		campaigns = append(campaigns, c)
	}
	return storage.CampaignPage{
		Campaigns:     campaigns,
		NextPageToken: "",
	}, nil
}

// fakeParticipantStore is a test double for storage.ParticipantStore.
type fakeParticipantStore struct {
	participants                      map[string]map[string]storage.ParticipantRecord // campaignID -> participantID -> Participant
	putErr                            error
	getErr                            error
	deleteErr                         error
	listErr                           error
	listByCampaignCalls               int
	listCampaignIDsByUserErr          error
	listCampaignIDsByUserCalls        int
	listCampaignIDsByParticipantErr   error
	listCampaignIDsByParticipantCalls int
}

// fakeInviteStore is a test double for storage.InviteStore.
type fakeInviteStore struct {
	invites   map[string]storage.InviteRecord
	putErr    error
	getErr    error
	listErr   error
	updateErr error
}

func newFakeInviteStore() *fakeInviteStore {
	return &fakeInviteStore{invites: make(map[string]storage.InviteRecord)}
}

func (s *fakeInviteStore) PutInvite(_ context.Context, inv storage.InviteRecord) error {
	if s.putErr != nil {
		return s.putErr
	}
	s.invites[inv.ID] = inv
	return nil
}

func (s *fakeInviteStore) GetInvite(_ context.Context, inviteID string) (storage.InviteRecord, error) {
	if s.getErr != nil {
		return storage.InviteRecord{}, s.getErr
	}
	inv, ok := s.invites[inviteID]
	if !ok {
		return storage.InviteRecord{}, storage.ErrNotFound
	}
	return inv, nil
}

func (s *fakeInviteStore) ListInvites(_ context.Context, campaignID string, recipientUserID string, status invite.Status, pageSize int, pageToken string) (storage.InvitePage, error) {
	if s.listErr != nil {
		return storage.InvitePage{}, s.listErr
	}
	result := make([]storage.InviteRecord, 0)
	for _, inv := range s.invites {
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

func (s *fakeInviteStore) ListPendingInvites(_ context.Context, campaignID string, pageSize int, pageToken string) (storage.InvitePage, error) {
	if s.listErr != nil {
		return storage.InvitePage{}, s.listErr
	}
	result := make([]storage.InviteRecord, 0)
	for _, inv := range s.invites {
		if inv.CampaignID == campaignID && inv.Status == invite.StatusPending {
			result = append(result, inv)
		}
	}
	return storage.InvitePage{Invites: result, NextPageToken: ""}, nil
}

func (s *fakeInviteStore) ListPendingInvitesForRecipient(_ context.Context, userID string, pageSize int, pageToken string) (storage.InvitePage, error) {
	if s.listErr != nil {
		return storage.InvitePage{}, s.listErr
	}
	result := make([]storage.InviteRecord, 0)
	for _, inv := range s.invites {
		if inv.RecipientUserID == userID && inv.Status == invite.StatusPending {
			result = append(result, inv)
		}
	}
	return storage.InvitePage{Invites: result, NextPageToken: ""}, nil
}

func (s *fakeInviteStore) UpdateInviteStatus(_ context.Context, inviteID string, status invite.Status, updatedAt time.Time) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	inv, ok := s.invites[inviteID]
	if !ok {
		return storage.ErrNotFound
	}
	inv.Status = status
	inv.UpdatedAt = updatedAt
	s.invites[inviteID] = inv
	return nil
}

func newFakeParticipantStore() *fakeParticipantStore {
	return &fakeParticipantStore{
		participants: make(map[string]map[string]storage.ParticipantRecord),
	}
}

func (s *fakeParticipantStore) PutParticipant(_ context.Context, p storage.ParticipantRecord) error {
	if s.putErr != nil {
		return s.putErr
	}
	if s.participants[p.CampaignID] == nil {
		s.participants[p.CampaignID] = make(map[string]storage.ParticipantRecord)
	}
	if strings.TrimSpace(p.UserID) != "" {
		for id, existing := range s.participants[p.CampaignID] {
			if id == p.ID {
				continue
			}
			if strings.TrimSpace(existing.UserID) == p.UserID {
				return apperrors.WithMetadata(
					apperrors.CodeParticipantUserAlreadyClaimed,
					"participant user already claimed",
					map[string]string{
						"CampaignID": p.CampaignID,
						"UserID":     p.UserID,
					},
				)
			}
		}
	}
	s.participants[p.CampaignID][p.ID] = p
	return nil
}

func (s *fakeParticipantStore) GetParticipant(_ context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
	if s.getErr != nil {
		return storage.ParticipantRecord{}, s.getErr
	}
	byID, ok := s.participants[campaignID]
	if !ok {
		return storage.ParticipantRecord{}, storage.ErrNotFound
	}
	p, ok := byID[participantID]
	if !ok {
		return storage.ParticipantRecord{}, storage.ErrNotFound
	}
	return p, nil
}

func (s *fakeParticipantStore) DeleteParticipant(_ context.Context, campaignID, participantID string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	byID, ok := s.participants[campaignID]
	if !ok {
		return storage.ErrNotFound
	}
	if _, ok := byID[participantID]; !ok {
		return storage.ErrNotFound
	}
	delete(byID, participantID)
	return nil
}

func (s *fakeParticipantStore) ListParticipantsByCampaign(_ context.Context, campaignID string) ([]storage.ParticipantRecord, error) {
	s.listByCampaignCalls++
	if s.listErr != nil {
		return nil, s.listErr
	}
	byID, ok := s.participants[campaignID]
	if !ok {
		return nil, nil
	}
	result := make([]storage.ParticipantRecord, 0, len(byID))
	for _, p := range byID {
		result = append(result, p)
	}
	return result, nil
}

func (s *fakeParticipantStore) ListCampaignIDsByUser(_ context.Context, userID string) ([]string, error) {
	s.listCampaignIDsByUserCalls++
	if s.listCampaignIDsByUserErr != nil {
		return nil, s.listCampaignIDsByUserErr
	}
	userID = strings.TrimSpace(userID)
	ids := make([]string, 0)
	seen := make(map[string]struct{})
	if userID == "" {
		return ids, nil
	}
	for _, byID := range s.participants {
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
	return ids, nil
}

func (s *fakeParticipantStore) ListCampaignIDsByParticipant(_ context.Context, participantID string) ([]string, error) {
	s.listCampaignIDsByParticipantCalls++
	if s.listCampaignIDsByParticipantErr != nil {
		return nil, s.listCampaignIDsByParticipantErr
	}
	participantID = strings.TrimSpace(participantID)
	ids := make([]string, 0)
	seen := make(map[string]struct{})
	if participantID == "" {
		return ids, nil
	}
	for _, byID := range s.participants {
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
	return ids, nil
}

func (s *fakeParticipantStore) ListParticipants(_ context.Context, campaignID string, pageSize int, pageToken string) (storage.ParticipantPage, error) {
	if s.listErr != nil {
		return storage.ParticipantPage{}, s.listErr
	}
	byID, ok := s.participants[campaignID]
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

func (s *fakeParticipantStore) CountParticipants(_ context.Context, campaignID string) (int, error) {
	byID, ok := s.participants[campaignID]
	if !ok {
		return 0, nil
	}
	return len(byID), nil
}

// fakeCharacterStore is a test double for storage.CharacterStore.
type fakeCharacterStore struct {
	characters map[string]map[string]storage.CharacterRecord // campaignID -> characterID -> Character
	putErr     error
	getErr     error
	deleteErr  error
	listErr    error
}

func newFakeCharacterStore() *fakeCharacterStore {
	return &fakeCharacterStore{
		characters: make(map[string]map[string]storage.CharacterRecord),
	}
}

func (s *fakeCharacterStore) PutCharacter(_ context.Context, c storage.CharacterRecord) error {
	if s.putErr != nil {
		return s.putErr
	}
	if s.characters[c.CampaignID] == nil {
		s.characters[c.CampaignID] = make(map[string]storage.CharacterRecord)
	}
	s.characters[c.CampaignID][c.ID] = c
	return nil
}

func (s *fakeCharacterStore) GetCharacter(_ context.Context, campaignID, characterID string) (storage.CharacterRecord, error) {
	if s.getErr != nil {
		return storage.CharacterRecord{}, s.getErr
	}
	byID, ok := s.characters[campaignID]
	if !ok {
		return storage.CharacterRecord{}, storage.ErrNotFound
	}
	c, ok := byID[characterID]
	if !ok {
		return storage.CharacterRecord{}, storage.ErrNotFound
	}
	return c, nil
}

func (s *fakeCharacterStore) DeleteCharacter(_ context.Context, campaignID, characterID string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	byID, ok := s.characters[campaignID]
	if !ok {
		return storage.ErrNotFound
	}
	if _, ok := byID[characterID]; !ok {
		return storage.ErrNotFound
	}
	delete(byID, characterID)
	return nil
}

func (s *fakeCharacterStore) ListCharacters(_ context.Context, campaignID string, pageSize int, pageToken string) (storage.CharacterPage, error) {
	if s.listErr != nil {
		return storage.CharacterPage{}, s.listErr
	}
	byID, ok := s.characters[campaignID]
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

func (s *fakeCharacterStore) CountCharacters(_ context.Context, campaignID string) (int, error) {
	byID, ok := s.characters[campaignID]
	if !ok {
		return 0, nil
	}
	return len(byID), nil
}

// fakeDaggerheartStore is a test double for storage.DaggerheartStore.
type fakeDaggerheartStore struct {
	profiles    map[string]map[string]storage.DaggerheartCharacterProfile // campaignID -> characterID -> profile
	states      map[string]map[string]storage.DaggerheartCharacterState   // campaignID -> characterID -> state
	snapshots   map[string]storage.DaggerheartSnapshot                    // campaignID -> snapshot
	countdowns  map[string]map[string]storage.DaggerheartCountdown        // campaignID -> countdownID -> countdown
	adversaries map[string]map[string]storage.DaggerheartAdversary        // campaignID -> adversaryID -> adversary
	statePuts   map[string]int
	snapPuts    map[string]int
	putErr      error
	getErr      error
}

func newFakeDaggerheartStore() *fakeDaggerheartStore {
	return &fakeDaggerheartStore{
		profiles:    make(map[string]map[string]storage.DaggerheartCharacterProfile),
		states:      make(map[string]map[string]storage.DaggerheartCharacterState),
		snapshots:   make(map[string]storage.DaggerheartSnapshot),
		countdowns:  make(map[string]map[string]storage.DaggerheartCountdown),
		adversaries: make(map[string]map[string]storage.DaggerheartAdversary),
		statePuts:   make(map[string]int),
		snapPuts:    make(map[string]int),
	}
}

func (s *fakeDaggerheartStore) PutDaggerheartCharacterProfile(_ context.Context, p storage.DaggerheartCharacterProfile) error {
	if s.putErr != nil {
		return s.putErr
	}
	if s.profiles[p.CampaignID] == nil {
		s.profiles[p.CampaignID] = make(map[string]storage.DaggerheartCharacterProfile)
	}
	s.profiles[p.CampaignID][p.CharacterID] = p
	return nil
}

func (s *fakeDaggerheartStore) GetDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) (storage.DaggerheartCharacterProfile, error) {
	if s.getErr != nil {
		return storage.DaggerheartCharacterProfile{}, s.getErr
	}
	byID, ok := s.profiles[campaignID]
	if !ok {
		return storage.DaggerheartCharacterProfile{}, storage.ErrNotFound
	}
	p, ok := byID[characterID]
	if !ok {
		return storage.DaggerheartCharacterProfile{}, storage.ErrNotFound
	}
	return p, nil
}

func (s *fakeDaggerheartStore) PutDaggerheartCharacterState(_ context.Context, st storage.DaggerheartCharacterState) error {
	if s.putErr != nil {
		return s.putErr
	}
	if s.states[st.CampaignID] == nil {
		s.states[st.CampaignID] = make(map[string]storage.DaggerheartCharacterState)
	}
	s.states[st.CampaignID][st.CharacterID] = st
	s.statePuts[st.CampaignID]++
	return nil
}

func (s *fakeDaggerheartStore) GetDaggerheartCharacterState(_ context.Context, campaignID, characterID string) (storage.DaggerheartCharacterState, error) {
	if s.getErr != nil {
		return storage.DaggerheartCharacterState{}, s.getErr
	}
	byID, ok := s.states[campaignID]
	if !ok {
		return storage.DaggerheartCharacterState{}, storage.ErrNotFound
	}
	st, ok := byID[characterID]
	if !ok {
		return storage.DaggerheartCharacterState{}, storage.ErrNotFound
	}
	return st, nil
}

func (s *fakeDaggerheartStore) PutDaggerheartSnapshot(_ context.Context, snap storage.DaggerheartSnapshot) error {
	if s.putErr != nil {
		return s.putErr
	}
	s.snapshots[snap.CampaignID] = snap
	s.snapPuts[snap.CampaignID]++
	return nil
}

func (s *fakeDaggerheartStore) GetDaggerheartSnapshot(_ context.Context, campaignID string) (storage.DaggerheartSnapshot, error) {
	if s.getErr != nil {
		return storage.DaggerheartSnapshot{}, s.getErr
	}
	snap, ok := s.snapshots[campaignID]
	if !ok {
		return storage.DaggerheartSnapshot{}, storage.ErrNotFound
	}
	return snap, nil
}

func (s *fakeDaggerheartStore) PutDaggerheartCountdown(_ context.Context, countdown storage.DaggerheartCountdown) error {
	if s.putErr != nil {
		return s.putErr
	}
	if s.countdowns[countdown.CampaignID] == nil {
		s.countdowns[countdown.CampaignID] = make(map[string]storage.DaggerheartCountdown)
	}
	s.countdowns[countdown.CampaignID][countdown.CountdownID] = countdown
	return nil
}

func (s *fakeDaggerheartStore) GetDaggerheartCountdown(_ context.Context, campaignID, countdownID string) (storage.DaggerheartCountdown, error) {
	if s.getErr != nil {
		return storage.DaggerheartCountdown{}, s.getErr
	}
	byID, ok := s.countdowns[campaignID]
	if !ok {
		return storage.DaggerheartCountdown{}, storage.ErrNotFound
	}
	countdown, ok := byID[countdownID]
	if !ok {
		return storage.DaggerheartCountdown{}, storage.ErrNotFound
	}
	return countdown, nil
}

func (s *fakeDaggerheartStore) ListDaggerheartCountdowns(_ context.Context, campaignID string) ([]storage.DaggerheartCountdown, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	byID, ok := s.countdowns[campaignID]
	if !ok {
		return nil, nil
	}
	result := make([]storage.DaggerheartCountdown, 0, len(byID))
	for _, countdown := range byID {
		result = append(result, countdown)
	}
	return result, nil
}

func (s *fakeDaggerheartStore) DeleteDaggerheartCountdown(_ context.Context, campaignID, countdownID string) error {
	if s.putErr != nil {
		return s.putErr
	}
	byID, ok := s.countdowns[campaignID]
	if !ok {
		return storage.ErrNotFound
	}
	if _, ok := byID[countdownID]; !ok {
		return storage.ErrNotFound
	}
	delete(byID, countdownID)
	return nil
}

func (s *fakeDaggerheartStore) PutDaggerheartAdversary(_ context.Context, adversary storage.DaggerheartAdversary) error {
	if s.putErr != nil {
		return s.putErr
	}
	if s.adversaries[adversary.CampaignID] == nil {
		s.adversaries[adversary.CampaignID] = make(map[string]storage.DaggerheartAdversary)
	}
	s.adversaries[adversary.CampaignID][adversary.AdversaryID] = adversary
	return nil
}

func (s *fakeDaggerheartStore) GetDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) (storage.DaggerheartAdversary, error) {
	if s.getErr != nil {
		return storage.DaggerheartAdversary{}, s.getErr
	}
	byID, ok := s.adversaries[campaignID]
	if !ok {
		return storage.DaggerheartAdversary{}, storage.ErrNotFound
	}
	adversary, ok := byID[adversaryID]
	if !ok {
		return storage.DaggerheartAdversary{}, storage.ErrNotFound
	}
	return adversary, nil
}

func (s *fakeDaggerheartStore) ListDaggerheartAdversaries(_ context.Context, campaignID, sessionID string) ([]storage.DaggerheartAdversary, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	byID, ok := s.adversaries[campaignID]
	if !ok {
		return nil, nil
	}
	result := make([]storage.DaggerheartAdversary, 0, len(byID))
	for _, adversary := range byID {
		if strings.TrimSpace(sessionID) != "" && adversary.SessionID != sessionID {
			continue
		}
		result = append(result, adversary)
	}
	return result, nil
}

func (s *fakeDaggerheartStore) DeleteDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) error {
	if s.putErr != nil {
		return s.putErr
	}
	byID, ok := s.adversaries[campaignID]
	if !ok {
		return storage.ErrNotFound
	}
	if _, ok := byID[adversaryID]; !ok {
		return storage.ErrNotFound
	}
	delete(byID, adversaryID)
	return nil
}

// fakeSessionGateStore is a test double for storage.SessionGateStore.
type fakeSessionGateStore struct {
	gates  map[string]storage.SessionGate
	putErr error
	getErr error
}

func newFakeSessionGateStore() *fakeSessionGateStore {
	return &fakeSessionGateStore{gates: make(map[string]storage.SessionGate)}
}

func (s *fakeSessionGateStore) PutSessionGate(_ context.Context, gate storage.SessionGate) error {
	if s.putErr != nil {
		return s.putErr
	}
	key := gate.CampaignID + ":" + gate.SessionID + ":" + gate.GateID
	s.gates[key] = gate
	return nil
}

func (s *fakeSessionGateStore) GetSessionGate(_ context.Context, campaignID, sessionID, gateID string) (storage.SessionGate, error) {
	if s.getErr != nil {
		return storage.SessionGate{}, s.getErr
	}
	key := campaignID + ":" + sessionID + ":" + gateID
	gate, ok := s.gates[key]
	if !ok {
		return storage.SessionGate{}, storage.ErrNotFound
	}
	return gate, nil
}

func (s *fakeSessionGateStore) GetOpenSessionGate(_ context.Context, campaignID, sessionID string) (storage.SessionGate, error) {
	if s.getErr != nil {
		return storage.SessionGate{}, s.getErr
	}
	for _, gate := range s.gates {
		if gate.CampaignID == campaignID && gate.SessionID == sessionID && gate.Status == session.GateStatusOpen {
			return gate, nil
		}
	}
	return storage.SessionGate{}, storage.ErrNotFound
}

// fakeSessionStore is a test double for storage.SessionStore.
type fakeSessionStore struct {
	sessions      map[string]map[string]storage.SessionRecord // campaignID -> sessionID -> Session
	activeSession map[string]string                           // campaignID -> sessionID (active session ID)
	putErr        error
	getErr        error
	endErr        error
	activeErr     error
	listErr       error
}

// fakeSessionSpotlightStore is a test double for storage.SessionSpotlightStore.
type fakeSessionSpotlightStore struct {
	spotlights map[string]map[string]storage.SessionSpotlight
	putErr     error
	getErr     error
	clearErr   error
}

func newFakeSessionSpotlightStore() *fakeSessionSpotlightStore {
	return &fakeSessionSpotlightStore{
		spotlights: make(map[string]map[string]storage.SessionSpotlight),
	}
}

func (s *fakeSessionSpotlightStore) PutSessionSpotlight(_ context.Context, spotlight storage.SessionSpotlight) error {
	if s.putErr != nil {
		return s.putErr
	}
	if s.spotlights[spotlight.CampaignID] == nil {
		s.spotlights[spotlight.CampaignID] = make(map[string]storage.SessionSpotlight)
	}
	s.spotlights[spotlight.CampaignID][spotlight.SessionID] = spotlight
	return nil
}

func (s *fakeSessionSpotlightStore) GetSessionSpotlight(_ context.Context, campaignID, sessionID string) (storage.SessionSpotlight, error) {
	if s.getErr != nil {
		return storage.SessionSpotlight{}, s.getErr
	}
	bySession, ok := s.spotlights[campaignID]
	if !ok {
		return storage.SessionSpotlight{}, storage.ErrNotFound
	}
	spotlight, ok := bySession[sessionID]
	if !ok {
		return storage.SessionSpotlight{}, storage.ErrNotFound
	}
	return spotlight, nil
}

func (s *fakeSessionSpotlightStore) ClearSessionSpotlight(_ context.Context, campaignID, sessionID string) error {
	if s.clearErr != nil {
		return s.clearErr
	}
	bySession, ok := s.spotlights[campaignID]
	if !ok {
		return storage.ErrNotFound
	}
	if _, ok := bySession[sessionID]; !ok {
		return storage.ErrNotFound
	}
	delete(bySession, sessionID)
	return nil
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{
		sessions:      make(map[string]map[string]storage.SessionRecord),
		activeSession: make(map[string]string),
	}
}

func (s *fakeSessionStore) PutSession(_ context.Context, sess storage.SessionRecord) error {
	if s.putErr != nil {
		return s.putErr
	}
	// Check for active session
	if activeID, ok := s.activeSession[sess.CampaignID]; ok && activeID != "" {
		return storage.ErrActiveSessionExists
	}
	if s.sessions[sess.CampaignID] == nil {
		s.sessions[sess.CampaignID] = make(map[string]storage.SessionRecord)
	}
	s.sessions[sess.CampaignID][sess.ID] = sess
	if sess.Status == session.StatusActive {
		s.activeSession[sess.CampaignID] = sess.ID
	}
	return nil
}

func (s *fakeSessionStore) EndSession(_ context.Context, campaignID, sessionID string, endedAt time.Time) (storage.SessionRecord, bool, error) {
	if s.endErr != nil {
		return storage.SessionRecord{}, false, s.endErr
	}
	byID, ok := s.sessions[campaignID]
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
	s.sessions[campaignID][sessionID] = sess
	if s.activeSession[campaignID] == sessionID {
		s.activeSession[campaignID] = ""
	}
	return sess, true, nil
}

func (s *fakeSessionStore) GetSession(_ context.Context, campaignID, sessionID string) (storage.SessionRecord, error) {
	if s.getErr != nil {
		return storage.SessionRecord{}, s.getErr
	}
	byID, ok := s.sessions[campaignID]
	if !ok {
		return storage.SessionRecord{}, storage.ErrNotFound
	}
	sess, ok := byID[sessionID]
	if !ok {
		return storage.SessionRecord{}, storage.ErrNotFound
	}
	return sess, nil
}

func (s *fakeSessionStore) GetActiveSession(_ context.Context, campaignID string) (storage.SessionRecord, error) {
	if s.activeErr != nil {
		return storage.SessionRecord{}, s.activeErr
	}
	activeID, ok := s.activeSession[campaignID]
	if !ok || activeID == "" {
		return storage.SessionRecord{}, storage.ErrNotFound
	}
	byID := s.sessions[campaignID]
	sess, ok := byID[activeID]
	if !ok {
		return storage.SessionRecord{}, storage.ErrNotFound
	}
	return sess, nil
}

func (s *fakeSessionStore) ListSessions(_ context.Context, campaignID string, pageSize int, pageToken string) (storage.SessionPage, error) {
	if s.listErr != nil {
		return storage.SessionPage{}, s.listErr
	}
	byID, ok := s.sessions[campaignID]
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

// fakeEventStore is a test double for storage.EventStore.
type fakeEventStore struct {
	events    map[string][]event.Event // campaignID -> events
	byHash    map[string]event.Event   // hash -> event
	appendErr error
	listErr   error
	getErr    error
	nextSeq   map[string]uint64 // campaignID -> nextSeq
}

func newFakeEventStore() *fakeEventStore {
	return &fakeEventStore{
		events:  make(map[string][]event.Event),
		byHash:  make(map[string]event.Event),
		nextSeq: make(map[string]uint64),
	}
}

func (s *fakeEventStore) AppendEvent(_ context.Context, evt event.Event) (event.Event, error) {
	if s.appendErr != nil {
		return event.Event{}, s.appendErr
	}
	seq := s.nextSeq[evt.CampaignID]
	if seq == 0 {
		seq = 1
	}
	evt.Seq = seq
	evt.Hash = "fakehash-" + evt.CampaignID + "-" + string(rune('0'+seq))
	s.nextSeq[evt.CampaignID] = seq + 1
	s.events[evt.CampaignID] = append(s.events[evt.CampaignID], evt)
	s.byHash[evt.Hash] = evt
	return evt, nil
}

func (s *fakeEventStore) GetEventByHash(_ context.Context, hash string) (event.Event, error) {
	if s.getErr != nil {
		return event.Event{}, s.getErr
	}
	evt, ok := s.byHash[hash]
	if !ok {
		return event.Event{}, storage.ErrNotFound
	}
	return evt, nil
}

func (s *fakeEventStore) GetEventBySeq(_ context.Context, campaignID string, seq uint64) (event.Event, error) {
	if s.getErr != nil {
		return event.Event{}, s.getErr
	}
	events, ok := s.events[campaignID]
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

func (s *fakeEventStore) ListEvents(_ context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	events, ok := s.events[campaignID]
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

func (s *fakeEventStore) ListEventsBySession(_ context.Context, campaignID, sessionID string, afterSeq uint64, limit int) ([]event.Event, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	events, ok := s.events[campaignID]
	if !ok {
		return nil, nil
	}
	var result []event.Event
	for _, e := range events {
		if e.SessionID == sessionID && e.Seq > afterSeq {
			result = append(result, e)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (s *fakeEventStore) GetLatestEventSeq(_ context.Context, campaignID string) (uint64, error) {
	if s.getErr != nil {
		return 0, s.getErr
	}
	seq := s.nextSeq[campaignID]
	if seq == 0 {
		return 0, nil
	}
	return seq - 1, nil
}

func (s *fakeEventStore) ListEventsPage(_ context.Context, req storage.ListEventsPageRequest) (storage.ListEventsPageResult, error) {
	if s.listErr != nil {
		return storage.ListEventsPageResult{}, s.listErr
	}
	events, ok := s.events[req.CampaignID]
	if !ok {
		return storage.ListEventsPageResult{TotalCount: 0}, nil
	}

	// Copy events for sorting
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

type fakeAuthClient struct {
	user               *authv1.User
	getUserErr         error
	lastGetUserRequest *authv1.GetUserRequest
}

func (f *fakeAuthClient) CreateUser(ctx context.Context, req *authv1.CreateUserRequest, opts ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *fakeAuthClient) IssueJoinGrant(ctx context.Context, req *authv1.IssueJoinGrantRequest, opts ...grpc.CallOption) (*authv1.IssueJoinGrantResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *fakeAuthClient) GetUser(ctx context.Context, req *authv1.GetUserRequest, opts ...grpc.CallOption) (*authv1.GetUserResponse, error) {
	f.lastGetUserRequest = req
	if f.getUserErr != nil {
		return nil, f.getUserErr
	}
	return &authv1.GetUserResponse{User: f.user}, nil
}

func (f *fakeAuthClient) ListUsers(ctx context.Context, req *authv1.ListUsersRequest, opts ...grpc.CallOption) (*authv1.ListUsersResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *fakeAuthClient) AddContact(ctx context.Context, req *authv1.AddContactRequest, opts ...grpc.CallOption) (*authv1.AddContactResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *fakeAuthClient) RemoveContact(ctx context.Context, req *authv1.RemoveContactRequest, opts ...grpc.CallOption) (*authv1.RemoveContactResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *fakeAuthClient) ListContacts(ctx context.Context, req *authv1.ListContactsRequest, opts ...grpc.CallOption) (*authv1.ListContactsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *fakeAuthClient) BeginPasskeyRegistration(ctx context.Context, req *authv1.BeginPasskeyRegistrationRequest, opts ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *fakeAuthClient) FinishPasskeyRegistration(ctx context.Context, req *authv1.FinishPasskeyRegistrationRequest, opts ...grpc.CallOption) (*authv1.FinishPasskeyRegistrationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *fakeAuthClient) BeginPasskeyLogin(ctx context.Context, req *authv1.BeginPasskeyLoginRequest, opts ...grpc.CallOption) (*authv1.BeginPasskeyLoginResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *fakeAuthClient) FinishPasskeyLogin(ctx context.Context, req *authv1.FinishPasskeyLoginRequest, opts ...grpc.CallOption) (*authv1.FinishPasskeyLoginResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *fakeAuthClient) GenerateMagicLink(ctx context.Context, req *authv1.GenerateMagicLinkRequest, opts ...grpc.CallOption) (*authv1.GenerateMagicLinkResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *fakeAuthClient) ConsumeMagicLink(ctx context.Context, req *authv1.ConsumeMagicLinkRequest, opts ...grpc.CallOption) (*authv1.ConsumeMagicLinkResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

func (f *fakeAuthClient) ListUserEmails(ctx context.Context, req *authv1.ListUserEmailsRequest, opts ...grpc.CallOption) (*authv1.ListUserEmailsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in fake auth client")
}

type fakeStatisticsStore struct {
	lastSince *time.Time
	stats     storage.GameStatistics
	err       error
}

func (f *fakeStatisticsStore) GetGameStatistics(_ context.Context, since *time.Time) (storage.GameStatistics, error) {
	f.lastSince = since
	return f.stats, f.err
}

type fakeCampaignForkStore struct {
	metadata map[string]storage.ForkMetadata
	getErr   error
	setErr   error
}

func newFakeCampaignForkStore() *fakeCampaignForkStore {
	return &fakeCampaignForkStore{metadata: make(map[string]storage.ForkMetadata)}
}

func (s *fakeCampaignForkStore) GetCampaignForkMetadata(_ context.Context, campaignID string) (storage.ForkMetadata, error) {
	if s.getErr != nil {
		return storage.ForkMetadata{}, s.getErr
	}
	metadata, ok := s.metadata[campaignID]
	if !ok {
		return storage.ForkMetadata{}, storage.ErrNotFound
	}
	return metadata, nil
}

func (s *fakeCampaignForkStore) SetCampaignForkMetadata(_ context.Context, campaignID string, metadata storage.ForkMetadata) error {
	if s.setErr != nil {
		return s.setErr
	}
	s.metadata[campaignID] = metadata
	return nil
}

// Test helper functions

func fixedClock(t time.Time) func() time.Time {
	return func() time.Time {
		return t
	}
}

func fixedIDGenerator(id string) func() (string, error) {
	return func() (string, error) {
		return id, nil
	}
}

func fixedSequenceIDGenerator(ids ...string) func() (string, error) {
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

func sequentialIDGenerator(prefix string) func() (string, error) {
	counter := 0
	return func() (string, error) {
		counter++
		return prefix + "-" + string(rune('0'+counter)), nil
	}
}

func contextWithParticipantID(participantID string) context.Context {
	if participantID == "" {
		return context.Background()
	}
	md := metadata.Pairs(grpcmeta.ParticipantIDHeader, participantID)
	return metadata.NewIncomingContext(context.Background(), md)
}

func contextWithUserID(userID string) context.Context {
	if userID == "" {
		return context.Background()
	}
	md := metadata.Pairs(grpcmeta.UserIDHeader, userID)
	return metadata.NewIncomingContext(context.Background(), md)
}

type joinGrantSigner struct {
	issuer   string
	audience string
	key      ed25519.PrivateKey
}

func newJoinGrantSigner(t *testing.T) joinGrantSigner {
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
	return joinGrantSigner{
		issuer:   issuer,
		audience: audience,
		key:      privateKey,
	}
}

func (s joinGrantSigner) Token(t *testing.T, campaignID, inviteID, userID, jti string, now time.Time) string {
	t.Helper()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if s.key == nil {
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
		"iss":         s.issuer,
		"aud":         s.audience,
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
	signature := ed25519.Sign(s.key, []byte(signingInput))
	encodedSig := base64.RawURLEncoding.EncodeToString(signature)
	return signingInput + "." + encodedSig
}
