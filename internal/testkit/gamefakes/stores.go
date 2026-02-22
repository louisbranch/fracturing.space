package gamefakes

import (
	"context"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// CampaignStore is a lightweight in-memory CampaignStore fake for tests.
type CampaignStore struct {
	Campaigns map[string]storage.CampaignRecord
}

// NewCampaignStore constructs a CampaignStore fake with initialized state maps.
func NewCampaignStore() *CampaignStore {
	return &CampaignStore{Campaigns: make(map[string]storage.CampaignRecord)}
}

func (s *CampaignStore) Put(_ context.Context, c storage.CampaignRecord) error {
	s.Campaigns[c.ID] = c
	return nil
}

func (s *CampaignStore) Get(_ context.Context, id string) (storage.CampaignRecord, error) {
	c, ok := s.Campaigns[id]
	if !ok {
		return storage.CampaignRecord{}, storage.ErrNotFound
	}
	return c, nil
}

func (s *CampaignStore) List(_ context.Context, _ int, _ string) (storage.CampaignPage, error) {
	return storage.CampaignPage{}, nil
}

// DaggerheartStore is an in-memory DaggerheartStore fake for tests.
type DaggerheartStore struct {
	Profiles   map[string]storage.DaggerheartCharacterProfile
	States     map[string]storage.DaggerheartCharacterState
	Snapshots  map[string]storage.DaggerheartSnapshot
	Countdowns map[string]storage.DaggerheartCountdown
}

// NewDaggerheartStore constructs a DaggerheartStore fake with initialized state maps.
func NewDaggerheartStore() *DaggerheartStore {
	return &DaggerheartStore{
		Profiles:   make(map[string]storage.DaggerheartCharacterProfile),
		States:     make(map[string]storage.DaggerheartCharacterState),
		Snapshots:  make(map[string]storage.DaggerheartSnapshot),
		Countdowns: make(map[string]storage.DaggerheartCountdown),
	}
}

func (s *DaggerheartStore) PutDaggerheartCharacterProfile(_ context.Context, p storage.DaggerheartCharacterProfile) error {
	s.Profiles[p.CampaignID+":"+p.CharacterID] = p
	return nil
}

func (s *DaggerheartStore) GetDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) (storage.DaggerheartCharacterProfile, error) {
	p, ok := s.Profiles[campaignID+":"+characterID]
	if !ok {
		return storage.DaggerheartCharacterProfile{}, storage.ErrNotFound
	}
	return p, nil
}

func (s *DaggerheartStore) PutDaggerheartCharacterState(_ context.Context, st storage.DaggerheartCharacterState) error {
	s.States[st.CampaignID+":"+st.CharacterID] = st
	return nil
}

func (s *DaggerheartStore) GetDaggerheartCharacterState(_ context.Context, campaignID, characterID string) (storage.DaggerheartCharacterState, error) {
	st, ok := s.States[campaignID+":"+characterID]
	if !ok {
		return storage.DaggerheartCharacterState{}, storage.ErrNotFound
	}
	return st, nil
}

func (s *DaggerheartStore) PutDaggerheartSnapshot(_ context.Context, snap storage.DaggerheartSnapshot) error {
	s.Snapshots[snap.CampaignID] = snap
	return nil
}

func (s *DaggerheartStore) GetDaggerheartSnapshot(_ context.Context, campaignID string) (storage.DaggerheartSnapshot, error) {
	snap, ok := s.Snapshots[campaignID]
	if !ok {
		return storage.DaggerheartSnapshot{}, storage.ErrNotFound
	}
	return snap, nil
}

func (s *DaggerheartStore) PutDaggerheartCountdown(_ context.Context, cd storage.DaggerheartCountdown) error {
	s.Countdowns[cd.CampaignID+":"+cd.CountdownID] = cd
	return nil
}

func (s *DaggerheartStore) GetDaggerheartCountdown(_ context.Context, campaignID, countdownID string) (storage.DaggerheartCountdown, error) {
	cd, ok := s.Countdowns[campaignID+":"+countdownID]
	if !ok {
		return storage.DaggerheartCountdown{}, storage.ErrNotFound
	}
	return cd, nil
}

func (s *DaggerheartStore) ListDaggerheartCountdowns(_ context.Context, campaignID string) ([]storage.DaggerheartCountdown, error) {
	result := make([]storage.DaggerheartCountdown, 0)
	for key, cd := range s.Countdowns {
		if len(key) > len(campaignID) && strings.HasPrefix(key, campaignID) {
			result = append(result, cd)
		}
	}
	return result, nil
}

func (s *DaggerheartStore) DeleteDaggerheartCountdown(_ context.Context, campaignID, countdownID string) error {
	delete(s.Countdowns, campaignID+":"+countdownID)
	return nil
}

func (s *DaggerheartStore) PutDaggerheartAdversary(_ context.Context, _ storage.DaggerheartAdversary) error {
	return nil
}

func (s *DaggerheartStore) GetDaggerheartAdversary(_ context.Context, _, _ string) (storage.DaggerheartAdversary, error) {
	return storage.DaggerheartAdversary{}, storage.ErrNotFound
}

func (s *DaggerheartStore) ListDaggerheartAdversaries(_ context.Context, _, _ string) ([]storage.DaggerheartAdversary, error) {
	return nil, nil
}

func (s *DaggerheartStore) DeleteDaggerheartAdversary(_ context.Context, _, _ string) error {
	return nil
}

// EventStore is an in-memory EventStore fake with simple list/filter behavior.
type EventStore struct {
	Events  map[string][]event.Event
	ByHash  map[string]event.Event
	NextSeq map[string]uint64
}

// NewEventStore constructs an EventStore fake with initialized state maps.
func NewEventStore() *EventStore {
	return &EventStore{
		Events:  make(map[string][]event.Event),
		ByHash:  make(map[string]event.Event),
		NextSeq: make(map[string]uint64),
	}
}

func (s *EventStore) AppendEvent(_ context.Context, evt event.Event) (event.Event, error) {
	seq := s.NextSeq[evt.CampaignID]
	if seq == 0 {
		seq = 1
	}
	evt.Seq = seq
	evt.Hash = "fakehash"
	s.NextSeq[evt.CampaignID] = seq + 1
	s.Events[evt.CampaignID] = append(s.Events[evt.CampaignID], evt)
	s.ByHash[evt.Hash] = evt
	return evt, nil
}

func (s *EventStore) GetEventByHash(_ context.Context, hash string) (event.Event, error) {
	evt, ok := s.ByHash[hash]
	if !ok {
		return event.Event{}, storage.ErrNotFound
	}
	return evt, nil
}

func (s *EventStore) GetEventBySeq(_ context.Context, campaignID string, seq uint64) (event.Event, error) {
	for _, evt := range s.Events[campaignID] {
		if evt.Seq == seq {
			return evt, nil
		}
	}
	return event.Event{}, storage.ErrNotFound
}

func (s *EventStore) ListEvents(_ context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error) {
	result := make([]event.Event, 0)
	for _, e := range s.Events[campaignID] {
		if e.Seq > afterSeq {
			result = append(result, e)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (s *EventStore) ListEventsBySession(_ context.Context, campaignID, sessionID string, afterSeq uint64, limit int) ([]event.Event, error) {
	result := make([]event.Event, 0)
	for _, e := range s.Events[campaignID] {
		if e.SessionID == sessionID && e.Seq > afterSeq {
			result = append(result, e)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (s *EventStore) GetLatestEventSeq(_ context.Context, campaignID string) (uint64, error) {
	seq := s.NextSeq[campaignID]
	if seq == 0 {
		return 0, nil
	}
	return seq - 1, nil
}

func (s *EventStore) ListEventsPage(_ context.Context, req storage.ListEventsPageRequest) (storage.ListEventsPageResult, error) {
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}

	filtered := make([]event.Event, 0)
	for _, evt := range s.Events[req.CampaignID] {
		if evt.Seq <= req.AfterSeq {
			continue
		}
		if !eventMatchesPageFilter(evt, req.FilterClause, req.FilterParams) {
			continue
		}
		filtered = append(filtered, evt)
	}

	if req.Descending {
		for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
			filtered[i], filtered[j] = filtered[j], filtered[i]
		}
	}

	totalCount := len(filtered)
	hasNextPage := len(filtered) > pageSize
	if hasNextPage {
		filtered = filtered[:pageSize]
	}

	return storage.ListEventsPageResult{
		Events:      filtered,
		HasNextPage: hasNextPage,
		TotalCount:  totalCount,
	}, nil
}

func eventMatchesPageFilter(evt event.Event, clause string, params []any) bool {
	if strings.TrimSpace(clause) == "" {
		return true
	}

	paramIndex := 0
	nextString := func() (string, bool) {
		if paramIndex >= len(params) {
			return "", false
		}
		value, ok := params[paramIndex].(string)
		if !ok {
			return "", false
		}
		paramIndex++
		return value, true
	}

	if strings.Contains(clause, "session_id = ?") {
		value, ok := nextString()
		if !ok || evt.SessionID != value {
			return false
		}
	}
	if strings.Contains(clause, "request_id = ?") {
		value, ok := nextString()
		if !ok || evt.RequestID != value {
			return false
		}
	}
	if strings.Contains(clause, "event_type = ?") {
		value, ok := nextString()
		if !ok || string(evt.Type) != value {
			return false
		}
	}
	if strings.Contains(clause, "entity_id = ?") {
		value, ok := nextString()
		if !ok || evt.EntityID != value {
			return false
		}
	}

	return true
}

// CharacterStore is a lightweight in-memory CharacterStore fake for tests.
type CharacterStore struct {
	Characters map[string]storage.CharacterRecord
}

// NewCharacterStore constructs a CharacterStore fake with initialized state maps.
func NewCharacterStore() *CharacterStore {
	return &CharacterStore{Characters: make(map[string]storage.CharacterRecord)}
}

func (s *CharacterStore) PutCharacter(_ context.Context, c storage.CharacterRecord) error {
	s.Characters[c.CampaignID+":"+c.ID] = c
	return nil
}

func (s *CharacterStore) GetCharacter(_ context.Context, campaignID, characterID string) (storage.CharacterRecord, error) {
	record, ok := s.Characters[campaignID+":"+characterID]
	if !ok {
		return storage.CharacterRecord{}, storage.ErrNotFound
	}
	return record, nil
}

func (s *CharacterStore) DeleteCharacter(_ context.Context, _, _ string) error {
	return nil
}

func (s *CharacterStore) CountCharacters(_ context.Context, campaignID string) (int, error) {
	count := 0
	for key := range s.Characters {
		if strings.HasPrefix(key, campaignID+":") {
			count++
		}
	}
	return count, nil
}

func (s *CharacterStore) ListCharacters(_ context.Context, _ string, _ int, _ string) (storage.CharacterPage, error) {
	return storage.CharacterPage{}, nil
}

// SessionStore is a lightweight in-memory SessionStore fake for tests.
type SessionStore struct {
	Sessions map[string]storage.SessionRecord
}

// NewSessionStore constructs a SessionStore fake with initialized state maps.
func NewSessionStore() *SessionStore {
	return &SessionStore{Sessions: make(map[string]storage.SessionRecord)}
}

func (s *SessionStore) PutSession(_ context.Context, sess storage.SessionRecord) error {
	s.Sessions[sess.CampaignID+":"+sess.ID] = sess
	return nil
}

func (s *SessionStore) EndSession(_ context.Context, _, _ string, _ time.Time) (storage.SessionRecord, bool, error) {
	return storage.SessionRecord{}, false, nil
}

func (s *SessionStore) GetSession(_ context.Context, campaignID, sessionID string) (storage.SessionRecord, error) {
	sess, ok := s.Sessions[campaignID+":"+sessionID]
	if !ok {
		return storage.SessionRecord{}, storage.ErrNotFound
	}
	return sess, nil
}

func (s *SessionStore) GetActiveSession(_ context.Context, _ string) (storage.SessionRecord, error) {
	return storage.SessionRecord{}, storage.ErrNotFound
}

func (s *SessionStore) ListSessions(_ context.Context, _ string, _ int, _ string) (storage.SessionPage, error) {
	return storage.SessionPage{}, nil
}
