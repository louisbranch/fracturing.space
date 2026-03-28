package gametest

import (
	"context"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

var (
	_ storage.SessionGateStore        = (*FakeSessionGateStore)(nil)
	_ storage.SessionStore            = (*FakeSessionStore)(nil)
	_ storage.SessionSpotlightStore   = (*FakeSessionSpotlightStore)(nil)
	_ storage.SessionInteractionStore = (*FakeSessionInteractionStore)(nil)
)

// FakeSessionGateStore is a test double for storage.SessionGateStore.
type FakeSessionGateStore struct {
	Gates  map[string]storage.SessionGate
	PutErr error
	GetErr error
}

// NewFakeSessionGateStore returns a ready-to-use session gate store fake.
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
	CountErr      error
	ListErr       error
}

// FakeSessionSpotlightStore is a test double for storage.SessionSpotlightStore.
type FakeSessionSpotlightStore struct {
	Spotlights map[string]map[string]storage.SessionSpotlight
	PutErr     error
	GetErr     error
	ClearErr   error
}

// NewFakeSessionSpotlightStore returns a ready-to-use spotlight store fake.
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

// NewFakeSessionStore returns a ready-to-use session store fake.
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

func (s *FakeSessionStore) CountSessions(_ context.Context, campaignID string) (int, error) {
	if s.CountErr != nil {
		return 0, s.CountErr
	}
	return len(s.Sessions[campaignID]), nil
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

// NewFakeSessionInteractionStore returns a ready-to-use session interaction fake.
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
