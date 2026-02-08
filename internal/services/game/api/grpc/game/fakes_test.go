package game

import (
	"context"
	"time"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/metadata"
)

// fakeCampaignStore is a test double for storage.CampaignStore.
type fakeCampaignStore struct {
	campaigns map[string]campaign.Campaign
	putErr    error
	getErr    error
	listErr   error
}

func newFakeCampaignStore() *fakeCampaignStore {
	return &fakeCampaignStore{
		campaigns: make(map[string]campaign.Campaign),
	}
}

func (s *fakeCampaignStore) Put(_ context.Context, c campaign.Campaign) error {
	if s.putErr != nil {
		return s.putErr
	}
	s.campaigns[c.ID] = c
	return nil
}

func (s *fakeCampaignStore) Get(_ context.Context, id string) (campaign.Campaign, error) {
	if s.getErr != nil {
		return campaign.Campaign{}, s.getErr
	}
	c, ok := s.campaigns[id]
	if !ok {
		return campaign.Campaign{}, storage.ErrNotFound
	}
	return c, nil
}

func (s *fakeCampaignStore) List(_ context.Context, pageSize int, pageToken string) (storage.CampaignPage, error) {
	if s.listErr != nil {
		return storage.CampaignPage{}, s.listErr
	}
	campaigns := make([]campaign.Campaign, 0, len(s.campaigns))
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
	participants map[string]map[string]participant.Participant // campaignID -> participantID -> Participant
	putErr       error
	getErr       error
	deleteErr    error
	listErr      error
}

// fakeInviteStore is a test double for storage.InviteStore.
type fakeInviteStore struct {
	invites   map[string]invite.Invite
	putErr    error
	getErr    error
	listErr   error
	updateErr error
}

func newFakeInviteStore() *fakeInviteStore {
	return &fakeInviteStore{invites: make(map[string]invite.Invite)}
}

func (s *fakeInviteStore) PutInvite(_ context.Context, inv invite.Invite) error {
	if s.putErr != nil {
		return s.putErr
	}
	s.invites[inv.ID] = inv
	return nil
}

func (s *fakeInviteStore) GetInvite(_ context.Context, inviteID string) (invite.Invite, error) {
	if s.getErr != nil {
		return invite.Invite{}, s.getErr
	}
	inv, ok := s.invites[inviteID]
	if !ok {
		return invite.Invite{}, storage.ErrNotFound
	}
	return inv, nil
}

func (s *fakeInviteStore) ListInvites(_ context.Context, campaignID string, pageSize int, pageToken string) (storage.InvitePage, error) {
	if s.listErr != nil {
		return storage.InvitePage{}, s.listErr
	}
	result := make([]invite.Invite, 0)
	for _, inv := range s.invites {
		if inv.CampaignID == campaignID {
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
		participants: make(map[string]map[string]participant.Participant),
	}
}

func (s *fakeParticipantStore) PutParticipant(_ context.Context, p participant.Participant) error {
	if s.putErr != nil {
		return s.putErr
	}
	if s.participants[p.CampaignID] == nil {
		s.participants[p.CampaignID] = make(map[string]participant.Participant)
	}
	s.participants[p.CampaignID][p.ID] = p
	return nil
}

func (s *fakeParticipantStore) GetParticipant(_ context.Context, campaignID, participantID string) (participant.Participant, error) {
	if s.getErr != nil {
		return participant.Participant{}, s.getErr
	}
	byID, ok := s.participants[campaignID]
	if !ok {
		return participant.Participant{}, storage.ErrNotFound
	}
	p, ok := byID[participantID]
	if !ok {
		return participant.Participant{}, storage.ErrNotFound
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

func (s *fakeParticipantStore) ListParticipantsByCampaign(_ context.Context, campaignID string) ([]participant.Participant, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	byID, ok := s.participants[campaignID]
	if !ok {
		return nil, nil
	}
	result := make([]participant.Participant, 0, len(byID))
	for _, p := range byID {
		result = append(result, p)
	}
	return result, nil
}

func (s *fakeParticipantStore) ListParticipants(_ context.Context, campaignID string, pageSize int, pageToken string) (storage.ParticipantPage, error) {
	if s.listErr != nil {
		return storage.ParticipantPage{}, s.listErr
	}
	byID, ok := s.participants[campaignID]
	if !ok {
		return storage.ParticipantPage{}, nil
	}
	result := make([]participant.Participant, 0, len(byID))
	for _, p := range byID {
		result = append(result, p)
	}
	return storage.ParticipantPage{
		Participants:  result,
		NextPageToken: "",
	}, nil
}

// fakeCharacterStore is a test double for storage.CharacterStore.
type fakeCharacterStore struct {
	characters map[string]map[string]character.Character // campaignID -> characterID -> Character
	putErr     error
	getErr     error
	deleteErr  error
	listErr    error
}

func newFakeCharacterStore() *fakeCharacterStore {
	return &fakeCharacterStore{
		characters: make(map[string]map[string]character.Character),
	}
}

func (s *fakeCharacterStore) PutCharacter(_ context.Context, c character.Character) error {
	if s.putErr != nil {
		return s.putErr
	}
	if s.characters[c.CampaignID] == nil {
		s.characters[c.CampaignID] = make(map[string]character.Character)
	}
	s.characters[c.CampaignID][c.ID] = c
	return nil
}

func (s *fakeCharacterStore) GetCharacter(_ context.Context, campaignID, characterID string) (character.Character, error) {
	if s.getErr != nil {
		return character.Character{}, s.getErr
	}
	byID, ok := s.characters[campaignID]
	if !ok {
		return character.Character{}, storage.ErrNotFound
	}
	c, ok := byID[characterID]
	if !ok {
		return character.Character{}, storage.ErrNotFound
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
	result := make([]character.Character, 0, len(byID))
	for _, c := range byID {
		result = append(result, c)
	}
	return storage.CharacterPage{
		Characters:    result,
		NextPageToken: "",
	}, nil
}

// fakeControlDefaultStore is a test double for storage.ControlDefaultStore.
type fakeControlDefaultStore struct {
	defaults map[string]map[string]character.CharacterController // campaignID -> characterID -> Controller
	putErr   error
	getErr   error
}

func newFakeControlDefaultStore() *fakeControlDefaultStore {
	return &fakeControlDefaultStore{
		defaults: make(map[string]map[string]character.CharacterController),
	}
}

func (s *fakeControlDefaultStore) PutControlDefault(_ context.Context, campaignID, characterID string, controller character.CharacterController) error {
	if s.putErr != nil {
		return s.putErr
	}
	if s.defaults[campaignID] == nil {
		s.defaults[campaignID] = make(map[string]character.CharacterController)
	}
	s.defaults[campaignID][characterID] = controller
	return nil
}

func (s *fakeControlDefaultStore) GetControlDefault(_ context.Context, campaignID, characterID string) (character.CharacterController, error) {
	if s.getErr != nil {
		return character.CharacterController{}, s.getErr
	}
	byID, ok := s.defaults[campaignID]
	if !ok {
		return character.CharacterController{}, storage.ErrNotFound
	}
	ctrl, ok := byID[characterID]
	if !ok {
		return character.CharacterController{}, storage.ErrNotFound
	}
	return ctrl, nil
}

// fakeDaggerheartStore is a test double for storage.DaggerheartStore.
type fakeDaggerheartStore struct {
	profiles  map[string]map[string]storage.DaggerheartCharacterProfile // campaignID -> characterID -> profile
	states    map[string]map[string]storage.DaggerheartCharacterState   // campaignID -> characterID -> state
	snapshots map[string]storage.DaggerheartSnapshot                    // campaignID -> snapshot
	statePuts map[string]int
	snapPuts  map[string]int
	putErr    error
	getErr    error
}

func newFakeDaggerheartStore() *fakeDaggerheartStore {
	return &fakeDaggerheartStore{
		profiles:  make(map[string]map[string]storage.DaggerheartCharacterProfile),
		states:    make(map[string]map[string]storage.DaggerheartCharacterState),
		snapshots: make(map[string]storage.DaggerheartSnapshot),
		statePuts: make(map[string]int),
		snapPuts:  make(map[string]int),
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

// fakeSessionStore is a test double for storage.SessionStore.
type fakeSessionStore struct {
	sessions      map[string]map[string]session.Session // campaignID -> sessionID -> Session
	activeSession map[string]string                     // campaignID -> sessionID (active session ID)
	putErr        error
	getErr        error
	endErr        error
	activeErr     error
	listErr       error
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{
		sessions:      make(map[string]map[string]session.Session),
		activeSession: make(map[string]string),
	}
}

func (s *fakeSessionStore) PutSession(_ context.Context, sess session.Session) error {
	if s.putErr != nil {
		return s.putErr
	}
	// Check for active session
	if activeID, ok := s.activeSession[sess.CampaignID]; ok && activeID != "" {
		return storage.ErrActiveSessionExists
	}
	if s.sessions[sess.CampaignID] == nil {
		s.sessions[sess.CampaignID] = make(map[string]session.Session)
	}
	s.sessions[sess.CampaignID][sess.ID] = sess
	if sess.Status == session.SessionStatusActive {
		s.activeSession[sess.CampaignID] = sess.ID
	}
	return nil
}

func (s *fakeSessionStore) EndSession(_ context.Context, campaignID, sessionID string, endedAt time.Time) (session.Session, bool, error) {
	if s.endErr != nil {
		return session.Session{}, false, s.endErr
	}
	byID, ok := s.sessions[campaignID]
	if !ok {
		return session.Session{}, false, storage.ErrNotFound
	}
	sess, ok := byID[sessionID]
	if !ok {
		return session.Session{}, false, storage.ErrNotFound
	}
	if sess.Status == session.SessionStatusEnded {
		return sess, false, nil
	}
	sess.Status = session.SessionStatusEnded
	sess.EndedAt = &endedAt
	sess.UpdatedAt = endedAt
	s.sessions[campaignID][sessionID] = sess
	if s.activeSession[campaignID] == sessionID {
		s.activeSession[campaignID] = ""
	}
	return sess, true, nil
}

func (s *fakeSessionStore) GetSession(_ context.Context, campaignID, sessionID string) (session.Session, error) {
	if s.getErr != nil {
		return session.Session{}, s.getErr
	}
	byID, ok := s.sessions[campaignID]
	if !ok {
		return session.Session{}, storage.ErrNotFound
	}
	sess, ok := byID[sessionID]
	if !ok {
		return session.Session{}, storage.ErrNotFound
	}
	return sess, nil
}

func (s *fakeSessionStore) GetActiveSession(_ context.Context, campaignID string) (session.Session, error) {
	if s.activeErr != nil {
		return session.Session{}, s.activeErr
	}
	activeID, ok := s.activeSession[campaignID]
	if !ok || activeID == "" {
		return session.Session{}, storage.ErrNotFound
	}
	byID := s.sessions[campaignID]
	sess, ok := byID[activeID]
	if !ok {
		return session.Session{}, storage.ErrNotFound
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
	result := make([]session.Session, 0, len(byID))
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

	// Apply cursor filter
	// The cursor direction directly determines the comparison:
	// - Forward (fwd): seq > cursor
	// - Backward (bwd): seq < cursor
	var filtered []event.Event
	for _, e := range sorted {
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
		TotalCount:  len(events),
	}, nil
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
