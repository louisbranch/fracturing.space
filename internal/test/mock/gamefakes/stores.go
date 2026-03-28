package gamefakes

import (
	"context"
	"sort"
	"strings"
	"time"

	corefilter "github.com/louisbranch/fracturing.space/internal/services/game/core/filter"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// CampaignStore is a lightweight in-memory CampaignStore fake for tests.
type CampaignStore struct {
	Campaigns map[string]storage.CampaignRecord
	PutErr    error
	GetErr    error
	ListErr   error
}

// NewCampaignStore constructs a CampaignStore fake with initialized state maps.
func NewCampaignStore() *CampaignStore {
	return &CampaignStore{Campaigns: make(map[string]storage.CampaignRecord)}
}

func (s *CampaignStore) Put(_ context.Context, c storage.CampaignRecord) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	s.Campaigns[c.ID] = c
	return nil
}

func (s *CampaignStore) Get(_ context.Context, id string) (storage.CampaignRecord, error) {
	if s.GetErr != nil {
		return storage.CampaignRecord{}, s.GetErr
	}
	c, ok := s.Campaigns[id]
	if !ok {
		return storage.CampaignRecord{}, storage.ErrNotFound
	}
	return c, nil
}

func (s *CampaignStore) List(_ context.Context, _ int, _ string) (storage.CampaignPage, error) {
	if s.ListErr != nil {
		return storage.CampaignPage{}, s.ListErr
	}
	return storage.CampaignPage{}, nil
}

// DaggerheartStore is an in-memory DaggerheartStore fake for tests.
type DaggerheartStore struct {
	Profiles            map[string]projectionstore.DaggerheartCharacterProfile
	States              map[string]projectionstore.DaggerheartCharacterState
	Snapshots           map[string]projectionstore.DaggerheartSnapshot
	Countdowns          map[string]projectionstore.DaggerheartCountdown
	Adversaries         map[string]projectionstore.DaggerheartAdversary
	EnvironmentEntities map[string]projectionstore.DaggerheartEnvironmentEntity
	PutErr              error
	GetErr              error
}

// NewDaggerheartStore constructs a DaggerheartStore fake with initialized state maps.
func NewDaggerheartStore() *DaggerheartStore {
	return &DaggerheartStore{
		Profiles:            make(map[string]projectionstore.DaggerheartCharacterProfile),
		States:              make(map[string]projectionstore.DaggerheartCharacterState),
		Snapshots:           make(map[string]projectionstore.DaggerheartSnapshot),
		Countdowns:          make(map[string]projectionstore.DaggerheartCountdown),
		Adversaries:         make(map[string]projectionstore.DaggerheartAdversary),
		EnvironmentEntities: make(map[string]projectionstore.DaggerheartEnvironmentEntity),
	}
}

func (s *DaggerheartStore) PutDaggerheartCharacterProfile(_ context.Context, p projectionstore.DaggerheartCharacterProfile) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	s.Profiles[p.CampaignID+":"+p.CharacterID] = p
	return nil
}

func (s *DaggerheartStore) GetDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterProfile, error) {
	if s.GetErr != nil {
		return projectionstore.DaggerheartCharacterProfile{}, s.GetErr
	}
	p, ok := s.Profiles[campaignID+":"+characterID]
	if !ok {
		return projectionstore.DaggerheartCharacterProfile{}, storage.ErrNotFound
	}
	return p, nil
}

func (s *DaggerheartStore) ListDaggerheartCharacterProfiles(_ context.Context, campaignID string, _ int, _ string) (projectionstore.DaggerheartCharacterProfilePage, error) {
	if s.GetErr != nil {
		return projectionstore.DaggerheartCharacterProfilePage{}, s.GetErr
	}
	page := projectionstore.DaggerheartCharacterProfilePage{
		Profiles: make([]projectionstore.DaggerheartCharacterProfile, 0),
	}
	for key, profile := range s.Profiles {
		if len(key) > len(campaignID) && strings.HasPrefix(key, campaignID+":") {
			page.Profiles = append(page.Profiles, profile)
		}
	}
	return page, nil
}

func (s *DaggerheartStore) DeleteDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	delete(s.Profiles, campaignID+":"+characterID)
	return nil
}

func (s *DaggerheartStore) PutDaggerheartCharacterState(_ context.Context, st projectionstore.DaggerheartCharacterState) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	s.States[st.CampaignID+":"+st.CharacterID] = st
	return nil
}

func (s *DaggerheartStore) GetDaggerheartCharacterState(_ context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error) {
	if s.GetErr != nil {
		return projectionstore.DaggerheartCharacterState{}, s.GetErr
	}
	st, ok := s.States[campaignID+":"+characterID]
	if !ok {
		return projectionstore.DaggerheartCharacterState{}, storage.ErrNotFound
	}
	return st, nil
}

func (s *DaggerheartStore) PutDaggerheartSnapshot(_ context.Context, snap projectionstore.DaggerheartSnapshot) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	s.Snapshots[snap.CampaignID] = snap
	return nil
}

func (s *DaggerheartStore) GetDaggerheartSnapshot(_ context.Context, campaignID string) (projectionstore.DaggerheartSnapshot, error) {
	if s.GetErr != nil {
		return projectionstore.DaggerheartSnapshot{}, s.GetErr
	}
	snap, ok := s.Snapshots[campaignID]
	if !ok {
		return projectionstore.DaggerheartSnapshot{}, storage.ErrNotFound
	}
	return snap, nil
}

func (s *DaggerheartStore) PutDaggerheartCountdown(_ context.Context, cd projectionstore.DaggerheartCountdown) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	s.Countdowns[cd.CampaignID+":"+cd.CountdownID] = cd
	return nil
}

func (s *DaggerheartStore) GetDaggerheartCountdown(_ context.Context, campaignID, countdownID string) (projectionstore.DaggerheartCountdown, error) {
	if s.GetErr != nil {
		return projectionstore.DaggerheartCountdown{}, s.GetErr
	}
	cd, ok := s.Countdowns[campaignID+":"+countdownID]
	if !ok {
		return projectionstore.DaggerheartCountdown{}, storage.ErrNotFound
	}
	return cd, nil
}

func (s *DaggerheartStore) ListDaggerheartCountdowns(_ context.Context, campaignID string) ([]projectionstore.DaggerheartCountdown, error) {
	if s.GetErr != nil {
		return nil, s.GetErr
	}
	result := make([]projectionstore.DaggerheartCountdown, 0)
	for key, cd := range s.Countdowns {
		if len(key) > len(campaignID) && strings.HasPrefix(key, campaignID+":") {
			result = append(result, cd)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}
		return result[i].CountdownID < result[j].CountdownID
	})
	return result, nil
}

func (s *DaggerheartStore) DeleteDaggerheartCountdown(_ context.Context, campaignID, countdownID string) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	delete(s.Countdowns, campaignID+":"+countdownID)
	return nil
}

func (s *DaggerheartStore) PutDaggerheartAdversary(_ context.Context, adv projectionstore.DaggerheartAdversary) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	s.Adversaries[adv.CampaignID+":"+adv.AdversaryID] = adv
	return nil
}

func (s *DaggerheartStore) GetDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
	if s.GetErr != nil {
		return projectionstore.DaggerheartAdversary{}, s.GetErr
	}
	adv, ok := s.Adversaries[campaignID+":"+adversaryID]
	if !ok {
		return projectionstore.DaggerheartAdversary{}, storage.ErrNotFound
	}
	return adv, nil
}

func (s *DaggerheartStore) ListDaggerheartAdversaries(_ context.Context, campaignID, sessionID string) ([]projectionstore.DaggerheartAdversary, error) {
	if s.GetErr != nil {
		return nil, s.GetErr
	}
	result := make([]projectionstore.DaggerheartAdversary, 0)
	prefix := campaignID + ":"
	for key, adv := range s.Adversaries {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if strings.TrimSpace(sessionID) != "" && adv.SessionID != sessionID {
			continue
		}
		result = append(result, adv)
	}
	return result, nil
}

func (s *DaggerheartStore) DeleteDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	delete(s.Adversaries, campaignID+":"+adversaryID)
	return nil
}

func (s *DaggerheartStore) PutDaggerheartEnvironmentEntity(_ context.Context, environmentEntity projectionstore.DaggerheartEnvironmentEntity) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	s.EnvironmentEntities[environmentEntity.CampaignID+":"+environmentEntity.EnvironmentEntityID] = environmentEntity
	return nil
}

func (s *DaggerheartStore) GetDaggerheartEnvironmentEntity(_ context.Context, campaignID, environmentEntityID string) (projectionstore.DaggerheartEnvironmentEntity, error) {
	if s.GetErr != nil {
		return projectionstore.DaggerheartEnvironmentEntity{}, s.GetErr
	}
	entity, ok := s.EnvironmentEntities[campaignID+":"+environmentEntityID]
	if !ok {
		return projectionstore.DaggerheartEnvironmentEntity{}, storage.ErrNotFound
	}
	return entity, nil
}

func (s *DaggerheartStore) ListDaggerheartEnvironmentEntities(_ context.Context, campaignID, sessionID, sceneID string) ([]projectionstore.DaggerheartEnvironmentEntity, error) {
	if s.GetErr != nil {
		return nil, s.GetErr
	}
	result := make([]projectionstore.DaggerheartEnvironmentEntity, 0)
	prefix := campaignID + ":"
	for key, entity := range s.EnvironmentEntities {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if strings.TrimSpace(sessionID) != "" && entity.SessionID != sessionID {
			continue
		}
		if strings.TrimSpace(sceneID) != "" && entity.SceneID != sceneID {
			continue
		}
		result = append(result, entity)
	}
	return result, nil
}

func (s *DaggerheartStore) DeleteDaggerheartEnvironmentEntity(_ context.Context, campaignID, environmentEntityID string) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	delete(s.EnvironmentEntities, campaignID+":"+environmentEntityID)
	return nil
}

// EventStore is an in-memory EventStore fake with simple list/filter behavior.
type EventStore struct {
	Events    map[string][]event.Event
	ByHash    map[string]event.Event
	NextSeq   map[string]uint64
	AppendErr error
	GetErr    error
	ListErr   error
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
	if s.AppendErr != nil {
		return event.Event{}, s.AppendErr
	}
	cid := string(evt.CampaignID)
	seq := s.NextSeq[cid]
	if seq == 0 {
		seq = 1
	}
	evt.Seq = seq
	evt.Hash = "fakehash"
	s.NextSeq[cid] = seq + 1
	s.Events[cid] = append(s.Events[cid], evt)
	s.ByHash[evt.Hash] = evt
	return evt, nil
}

func (s *EventStore) GetEventByHash(_ context.Context, hash string) (event.Event, error) {
	if s.GetErr != nil {
		return event.Event{}, s.GetErr
	}
	evt, ok := s.ByHash[hash]
	if !ok {
		return event.Event{}, storage.ErrNotFound
	}
	return evt, nil
}

func (s *EventStore) GetEventBySeq(_ context.Context, campaignID string, seq uint64) (event.Event, error) {
	if s.GetErr != nil {
		return event.Event{}, s.GetErr
	}
	for _, evt := range s.Events[campaignID] {
		if evt.Seq == seq {
			return evt, nil
		}
	}
	return event.Event{}, storage.ErrNotFound
}

func (s *EventStore) ListEvents(_ context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error) {
	if s.ListErr != nil {
		return nil, s.ListErr
	}
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
	if s.ListErr != nil {
		return nil, s.ListErr
	}
	result := make([]event.Event, 0)
	for _, e := range s.Events[campaignID] {
		if e.SessionID.String() == sessionID && e.Seq > afterSeq {
			result = append(result, e)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (s *EventStore) GetLatestEventSeq(_ context.Context, campaignID string) (uint64, error) {
	if s.GetErr != nil {
		return 0, s.GetErr
	}
	seq := s.NextSeq[campaignID]
	if seq == 0 {
		return 0, nil
	}
	return seq - 1, nil
}

func (s *EventStore) ListEventsPage(_ context.Context, req storage.ListEventsPageRequest) (storage.ListEventsPageResult, error) {
	if s.ListErr != nil {
		return storage.ListEventsPageResult{}, s.ListErr
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}

	filtered := make([]event.Event, 0)
	for _, evt := range s.Events[req.CampaignID] {
		if evt.Seq <= req.AfterSeq {
			continue
		}
		if !eventMatchesPageFilter(evt, req.Filter) {
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

func eventMatchesPageFilter(evt event.Event, filter storage.EventQueryFilter) bool {
	matchExact := func(value, current string) bool {
		value = strings.TrimSpace(value)
		if value == "" {
			return true
		}
		return current == value
	}

	if !matchExact(filter.EventType, string(evt.Type)) {
		return false
	}
	if !matchExact(filter.SessionID, evt.SessionID.String()) {
		return false
	}
	if !matchExact(filter.SceneID, evt.SceneID.String()) {
		return false
	}
	if !matchExact(filter.RequestID, evt.RequestID) {
		return false
	}
	if !matchExact(filter.InvocationID, evt.InvocationID) {
		return false
	}
	if !matchExact(filter.ActorType, string(evt.ActorType)) {
		return false
	}
	if !matchExact(filter.ActorID, evt.ActorID) {
		return false
	}
	if !matchExact(filter.SystemID, evt.SystemID) {
		return false
	}
	if !matchExact(filter.SystemVersion, evt.SystemVersion) {
		return false
	}
	if !matchExact(filter.EntityType, evt.EntityType) {
		return false
	}
	if !matchExact(filter.EntityID, evt.EntityID) {
		return false
	}

	expression := strings.TrimSpace(filter.Expression)
	if expression == "" {
		return true
	}
	cond, err := corefilter.ParseEventFilter(expression)
	if err != nil {
		return false
	}
	return eventMatchesPageFilterClause(evt, cond.Clause, cond.Params)
}

func eventMatchesPageFilterClause(evt event.Event, clause string, params []any) bool {
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
		if !ok || evt.SessionID.String() != value {
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
	PutErr     error
	GetErr     error
	DeleteErr  error
	ListErr    error
}

// NewCharacterStore constructs a CharacterStore fake with initialized state maps.
func NewCharacterStore() *CharacterStore {
	return &CharacterStore{Characters: make(map[string]storage.CharacterRecord)}
}

func (s *CharacterStore) PutCharacter(_ context.Context, c storage.CharacterRecord) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	s.Characters[c.CampaignID+":"+c.ID] = c
	return nil
}

func (s *CharacterStore) GetCharacter(_ context.Context, campaignID, characterID string) (storage.CharacterRecord, error) {
	if s.GetErr != nil {
		return storage.CharacterRecord{}, s.GetErr
	}
	record, ok := s.Characters[campaignID+":"+characterID]
	if !ok {
		return storage.CharacterRecord{}, storage.ErrNotFound
	}
	return record, nil
}

func (s *CharacterStore) DeleteCharacter(_ context.Context, campaignID, characterID string) error {
	if s.DeleteErr != nil {
		return s.DeleteErr
	}
	delete(s.Characters, campaignID+":"+characterID)
	return nil
}

func (s *CharacterStore) CountCharacters(_ context.Context, campaignID string) (int, error) {
	if s.ListErr != nil {
		return 0, s.ListErr
	}
	count := 0
	for key := range s.Characters {
		if strings.HasPrefix(key, campaignID+":") {
			count++
		}
	}
	return count, nil
}

func (s *CharacterStore) ListCharactersByOwnerParticipant(_ context.Context, campaignID, participantID string) ([]storage.CharacterRecord, error) {
	if s.ListErr != nil {
		return nil, s.ListErr
	}
	result := make([]storage.CharacterRecord, 0)
	for key, record := range s.Characters {
		if !strings.HasPrefix(key, campaignID+":") {
			continue
		}
		if record.OwnerParticipantID == participantID {
			result = append(result, record)
		}
	}
	return result, nil
}

func (s *CharacterStore) ListCharactersByControllerParticipant(_ context.Context, campaignID, participantID string) ([]storage.CharacterRecord, error) {
	if s.ListErr != nil {
		return nil, s.ListErr
	}
	result := make([]storage.CharacterRecord, 0)
	for key, record := range s.Characters {
		if !strings.HasPrefix(key, campaignID+":") {
			continue
		}
		if record.OwnerParticipantID == participantID {
			result = append(result, record)
		}
	}
	return result, nil
}

func (s *CharacterStore) ListCharacters(_ context.Context, _ string, _ int, _ string) (storage.CharacterPage, error) {
	if s.ListErr != nil {
		return storage.CharacterPage{}, s.ListErr
	}
	return storage.CharacterPage{}, nil
}

// SessionStore is a lightweight in-memory SessionStore fake for tests.
type SessionStore struct {
	Sessions map[string]storage.SessionRecord
	PutErr   error
	GetErr   error
	EndErr   error
	ListErr  error
}

// NewSessionStore constructs a SessionStore fake with initialized state maps.
func NewSessionStore() *SessionStore {
	return &SessionStore{Sessions: make(map[string]storage.SessionRecord)}
}

func (s *SessionStore) PutSession(_ context.Context, sess storage.SessionRecord) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	s.Sessions[sess.CampaignID+":"+sess.ID] = sess
	return nil
}

func (s *SessionStore) EndSession(_ context.Context, campaignID, sessionID string, endedAt time.Time) (storage.SessionRecord, bool, error) {
	if s.EndErr != nil {
		return storage.SessionRecord{}, false, s.EndErr
	}
	key := campaignID + ":" + sessionID
	sess, ok := s.Sessions[key]
	if !ok {
		return storage.SessionRecord{}, false, storage.ErrNotFound
	}
	sess.EndedAt = &endedAt
	s.Sessions[key] = sess
	return sess, true, nil
}

func (s *SessionStore) GetSession(_ context.Context, campaignID, sessionID string) (storage.SessionRecord, error) {
	if s.GetErr != nil {
		return storage.SessionRecord{}, s.GetErr
	}
	sess, ok := s.Sessions[campaignID+":"+sessionID]
	if !ok {
		return storage.SessionRecord{}, storage.ErrNotFound
	}
	return sess, nil
}

func (s *SessionStore) GetActiveSession(_ context.Context, campaignID string) (storage.SessionRecord, error) {
	if s.GetErr != nil {
		return storage.SessionRecord{}, s.GetErr
	}
	for key, sess := range s.Sessions {
		if strings.HasPrefix(key, campaignID+":") && sess.EndedAt == nil {
			return sess, nil
		}
	}
	return storage.SessionRecord{}, storage.ErrNotFound
}

func (s *SessionStore) CountSessions(_ context.Context, campaignID string) (int, error) {
	if s.ListErr != nil {
		return 0, s.ListErr
	}
	count := 0
	for key := range s.Sessions {
		if strings.HasPrefix(key, campaignID+":") {
			count++
		}
	}
	return count, nil
}

func (s *SessionStore) ListSessions(_ context.Context, campaignID string, _ int, _ string) (storage.SessionPage, error) {
	if s.ListErr != nil {
		return storage.SessionPage{}, s.ListErr
	}
	result := make([]storage.SessionRecord, 0)
	for key, sess := range s.Sessions {
		if strings.HasPrefix(key, campaignID+":") {
			result = append(result, sess)
		}
	}
	return storage.SessionPage{Sessions: result}, nil
}
