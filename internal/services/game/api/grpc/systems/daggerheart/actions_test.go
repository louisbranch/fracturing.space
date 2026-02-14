package daggerheart

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

// --- Fake stores for daggerheart action tests ---

type fakeCampaignStore struct {
	campaigns map[string]campaign.Campaign
}

func newFakeCampaignStore() *fakeCampaignStore {
	return &fakeCampaignStore{campaigns: make(map[string]campaign.Campaign)}
}

func (s *fakeCampaignStore) Put(_ context.Context, c campaign.Campaign) error {
	s.campaigns[c.ID] = c
	return nil
}

func (s *fakeCampaignStore) Get(_ context.Context, id string) (campaign.Campaign, error) {
	c, ok := s.campaigns[id]
	if !ok {
		return campaign.Campaign{}, storage.ErrNotFound
	}
	return c, nil
}

func (s *fakeCampaignStore) List(_ context.Context, _ int, _ string) (storage.CampaignPage, error) {
	return storage.CampaignPage{}, nil
}

type fakeDaggerheartStore struct {
	profiles   map[string]storage.DaggerheartCharacterProfile
	states     map[string]storage.DaggerheartCharacterState
	snapshots  map[string]storage.DaggerheartSnapshot
	countdowns map[string]storage.DaggerheartCountdown
}

func newFakeDaggerheartStore() *fakeDaggerheartStore {
	return &fakeDaggerheartStore{
		profiles:   make(map[string]storage.DaggerheartCharacterProfile),
		states:     make(map[string]storage.DaggerheartCharacterState),
		snapshots:  make(map[string]storage.DaggerheartSnapshot),
		countdowns: make(map[string]storage.DaggerheartCountdown),
	}
}

func (s *fakeDaggerheartStore) PutDaggerheartCharacterProfile(_ context.Context, p storage.DaggerheartCharacterProfile) error {
	s.profiles[p.CampaignID+":"+p.CharacterID] = p
	return nil
}

func (s *fakeDaggerheartStore) GetDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) (storage.DaggerheartCharacterProfile, error) {
	p, ok := s.profiles[campaignID+":"+characterID]
	if !ok {
		return storage.DaggerheartCharacterProfile{}, storage.ErrNotFound
	}
	return p, nil
}

func (s *fakeDaggerheartStore) PutDaggerheartCharacterState(_ context.Context, st storage.DaggerheartCharacterState) error {
	s.states[st.CampaignID+":"+st.CharacterID] = st
	return nil
}

func (s *fakeDaggerheartStore) GetDaggerheartCharacterState(_ context.Context, campaignID, characterID string) (storage.DaggerheartCharacterState, error) {
	st, ok := s.states[campaignID+":"+characterID]
	if !ok {
		return storage.DaggerheartCharacterState{}, storage.ErrNotFound
	}
	return st, nil
}

func (s *fakeDaggerheartStore) PutDaggerheartSnapshot(_ context.Context, snap storage.DaggerheartSnapshot) error {
	s.snapshots[snap.CampaignID] = snap
	return nil
}

func (s *fakeDaggerheartStore) GetDaggerheartSnapshot(_ context.Context, campaignID string) (storage.DaggerheartSnapshot, error) {
	snap, ok := s.snapshots[campaignID]
	if !ok {
		return storage.DaggerheartSnapshot{}, storage.ErrNotFound
	}
	return snap, nil
}

func (s *fakeDaggerheartStore) PutDaggerheartCountdown(_ context.Context, cd storage.DaggerheartCountdown) error {
	s.countdowns[cd.CampaignID+":"+cd.CountdownID] = cd
	return nil
}

func (s *fakeDaggerheartStore) GetDaggerheartCountdown(_ context.Context, campaignID, countdownID string) (storage.DaggerheartCountdown, error) {
	cd, ok := s.countdowns[campaignID+":"+countdownID]
	if !ok {
		return storage.DaggerheartCountdown{}, storage.ErrNotFound
	}
	return cd, nil
}

func (s *fakeDaggerheartStore) ListDaggerheartCountdowns(_ context.Context, campaignID string) ([]storage.DaggerheartCountdown, error) {
	var result []storage.DaggerheartCountdown
	for key, cd := range s.countdowns {
		if len(key) > len(campaignID) && key[:len(campaignID)] == campaignID {
			result = append(result, cd)
		}
	}
	return result, nil
}

func (s *fakeDaggerheartStore) DeleteDaggerheartCountdown(_ context.Context, campaignID, countdownID string) error {
	delete(s.countdowns, campaignID+":"+countdownID)
	return nil
}

func (s *fakeDaggerheartStore) PutDaggerheartAdversary(_ context.Context, _ storage.DaggerheartAdversary) error {
	return nil
}

func (s *fakeDaggerheartStore) GetDaggerheartAdversary(_ context.Context, _, _ string) (storage.DaggerheartAdversary, error) {
	return storage.DaggerheartAdversary{}, storage.ErrNotFound
}

func (s *fakeDaggerheartStore) ListDaggerheartAdversaries(_ context.Context, _, _ string) ([]storage.DaggerheartAdversary, error) {
	return nil, nil
}

func (s *fakeDaggerheartStore) DeleteDaggerheartAdversary(_ context.Context, _, _ string) error {
	return nil
}

type fakeEventStore struct {
	events  map[string][]event.Event
	byHash  map[string]event.Event
	nextSeq map[string]uint64
}

func newFakeActionEventStore() *fakeEventStore {
	return &fakeEventStore{
		events:  make(map[string][]event.Event),
		byHash:  make(map[string]event.Event),
		nextSeq: make(map[string]uint64),
	}
}

func (s *fakeEventStore) AppendEvent(_ context.Context, evt event.Event) (event.Event, error) {
	seq := s.nextSeq[evt.CampaignID]
	if seq == 0 {
		seq = 1
	}
	evt.Seq = seq
	evt.Hash = "fakehash"
	s.nextSeq[evt.CampaignID] = seq + 1
	s.events[evt.CampaignID] = append(s.events[evt.CampaignID], evt)
	s.byHash[evt.Hash] = evt
	return evt, nil
}

func (s *fakeEventStore) GetEventByHash(_ context.Context, hash string) (event.Event, error) {
	evt, ok := s.byHash[hash]
	if !ok {
		return event.Event{}, storage.ErrNotFound
	}
	return evt, nil
}

func (s *fakeEventStore) GetEventBySeq(_ context.Context, campaignID string, seq uint64) (event.Event, error) {
	for _, evt := range s.events[campaignID] {
		if evt.Seq == seq {
			return evt, nil
		}
	}
	return event.Event{}, storage.ErrNotFound
}

func (s *fakeEventStore) ListEvents(_ context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error) {
	var result []event.Event
	for _, e := range s.events[campaignID] {
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
	var result []event.Event
	for _, e := range s.events[campaignID] {
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
	seq := s.nextSeq[campaignID]
	if seq == 0 {
		return 0, nil
	}
	return seq - 1, nil
}

func (s *fakeEventStore) ListEventsPage(_ context.Context, _ storage.ListEventsPageRequest) (storage.ListEventsPageResult, error) {
	return storage.ListEventsPageResult{}, nil
}

type fakeCharacterStore struct {
	characters map[string]character.Character
}

func newFakeCharacterStore() *fakeCharacterStore {
	return &fakeCharacterStore{characters: make(map[string]character.Character)}
}

func (s *fakeCharacterStore) PutCharacter(_ context.Context, c character.Character) error {
	s.characters[c.CampaignID+":"+c.ID] = c
	return nil
}

func (s *fakeCharacterStore) GetCharacter(_ context.Context, campaignID, characterID string) (character.Character, error) {
	c, ok := s.characters[campaignID+":"+characterID]
	if !ok {
		return character.Character{}, storage.ErrNotFound
	}
	return c, nil
}

func (s *fakeCharacterStore) DeleteCharacter(_ context.Context, _, _ string) error {
	return nil
}

func (s *fakeCharacterStore) ListCharacters(_ context.Context, _ string, _ int, _ string) (storage.CharacterPage, error) {
	return storage.CharacterPage{}, nil
}

type fakeSessionGateStore struct{}

func (s *fakeSessionGateStore) PutSessionGate(_ context.Context, _ storage.SessionGate) error {
	return nil
}

func (s *fakeSessionGateStore) GetSessionGate(_ context.Context, _, _, _ string) (storage.SessionGate, error) {
	return storage.SessionGate{}, storage.ErrNotFound
}

func (s *fakeSessionGateStore) GetOpenSessionGate(_ context.Context, _, _ string) (storage.SessionGate, error) {
	return storage.SessionGate{}, storage.ErrNotFound
}

type fakeSessionSpotlightStore struct{}

func (s *fakeSessionSpotlightStore) PutSessionSpotlight(_ context.Context, _ storage.SessionSpotlight) error {
	return nil
}

func (s *fakeSessionSpotlightStore) GetSessionSpotlight(_ context.Context, _, _ string) (storage.SessionSpotlight, error) {
	return storage.SessionSpotlight{}, storage.ErrNotFound
}

func (s *fakeSessionSpotlightStore) ClearSessionSpotlight(_ context.Context, _, _ string) error {
	return nil
}

type fakeSessionStore struct {
	sessions map[string]session.Session // campaignID:sessionID -> session
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{sessions: make(map[string]session.Session)}
}

func (s *fakeSessionStore) PutSession(_ context.Context, sess session.Session) error {
	s.sessions[sess.CampaignID+":"+sess.ID] = sess
	return nil
}

func (s *fakeSessionStore) EndSession(_ context.Context, _, _ string, _ time.Time) (session.Session, bool, error) {
	return session.Session{}, false, nil
}

func (s *fakeSessionStore) GetSession(_ context.Context, campaignID, sessionID string) (session.Session, error) {
	sess, ok := s.sessions[campaignID+":"+sessionID]
	if !ok {
		return session.Session{}, storage.ErrNotFound
	}
	return sess, nil
}

func (s *fakeSessionStore) GetActiveSession(_ context.Context, _ string) (session.Session, error) {
	return session.Session{}, storage.ErrNotFound
}

func (s *fakeSessionStore) ListSessions(_ context.Context, _ string, _ int, _ string) (storage.SessionPage, error) {
	return storage.SessionPage{}, nil
}

func contextWithSessionID(sessionID string) context.Context {
	md := metadata.Pairs(grpcmeta.SessionIDHeader, sessionID)
	return metadata.NewIncomingContext(context.Background(), md)
}

func newActionTestService() *DaggerheartService {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["camp-1"] = campaign.Campaign{
		ID:     "camp-1",
		Status: campaign.CampaignStatusActive,
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
	}

	dhStore := newFakeDaggerheartStore()
	dhStore.profiles["camp-1:char-1"] = storage.DaggerheartCharacterProfile{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HpMax:       6,
		StressMax:   6,
		ArmorMax:    2,
	}
	dhStore.states["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          6,
		Hope:        2,
		HopeMax:     daggerheart.HopeMax,
		Stress:      3,
		Armor:       0,
		LifeState:   daggerheart.LifeStateAlive,
	}
	dhStore.profiles["camp-1:char-2"] = storage.DaggerheartCharacterProfile{
		CampaignID:  "camp-1",
		CharacterID: "char-2",
		HpMax:       8,
		StressMax:   6,
		ArmorMax:    1,
	}
	dhStore.states["camp-1:char-2"] = storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-2",
		Hp:          8,
		Hope:        3,
		HopeMax:     daggerheart.HopeMax,
		Stress:      1,
		Armor:       0,
		LifeState:   daggerheart.LifeStateAlive,
	}

	sessStore := newFakeSessionStore()
	sessStore.sessions["camp-1:sess-1"] = session.Session{
		ID:         "sess-1",
		CampaignID: "camp-1",
		Status:     session.SessionStatusActive,
	}

	return &DaggerheartService{
		stores: Stores{
			Campaign:         campaignStore,
			Daggerheart:      dhStore,
			Event:            newFakeActionEventStore(),
			SessionGate:      &fakeSessionGateStore{},
			SessionSpotlight: &fakeSessionSpotlightStore{},
			Character:        newFakeCharacterStore(),
			Session:          sessStore,
		},
		seedFunc: func() (int64, error) { return 42, nil },
	}
}

// --- ApplyDowntimeMove tests ---

func TestApplyDowntimeMove_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyDowntimeMove(context.Background(), &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId: "c1", CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyDowntimeMove_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDowntimeMove(ctx, &pb.DaggerheartApplyDowntimeMoveRequest{
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDowntimeMove_MissingCharacterId(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDowntimeMove(ctx, &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDowntimeMove_CampaignNotFound(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDowntimeMove(ctx, &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId: "nonexistent", CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyDowntimeMove_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyDowntimeMove(context.Background(), &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDowntimeMove_MissingMove(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDowntimeMove(ctx, &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDowntimeMove_UnspecifiedMove(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDowntimeMove(ctx, &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move: &pb.DaggerheartDowntimeRequest{
			Move: pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_UNSPECIFIED,
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDowntimeMove_Success(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyDowntimeMove(ctx, &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move: &pb.DaggerheartDowntimeRequest{
			Move: pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_CLEAR_ALL_STRESS,
		},
	})
	if err != nil {
		t.Fatalf("ApplyDowntimeMove returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character_id = %q, want char-1", resp.CharacterId)
	}
	if resp.State == nil {
		t.Fatal("expected state in response")
	}
	if resp.State.Stress != 0 {
		t.Fatalf("stress = %d, want 0", resp.State.Stress)
	}
}

// --- SwapLoadout tests ---

func TestSwapLoadout_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SwapLoadout(context.Background(), &pb.DaggerheartSwapLoadoutRequest{
		CampaignId: "c1", CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSwapLoadout_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSwapLoadout_MissingCharacterId(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSwapLoadout_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SwapLoadout(context.Background(), &pb.DaggerheartSwapLoadoutRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSwapLoadout_MissingSwap(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSwapLoadout_MissingCardId(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Swap:        &pb.DaggerheartLoadoutSwapRequest{},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSwapLoadout_NegativeRecallCost(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Swap: &pb.DaggerheartLoadoutSwapRequest{
			CardId:     "card-1",
			RecallCost: -1,
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSwapLoadout_Success(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Swap: &pb.DaggerheartLoadoutSwapRequest{
			CardId:     "card-1",
			RecallCost: 0,
		},
	})
	if err != nil {
		t.Fatalf("SwapLoadout returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character_id = %q, want char-1", resp.CharacterId)
	}
	if resp.State == nil {
		t.Fatal("expected state in response")
	}
}

func TestSwapLoadout_WithRecallCost(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Swap: &pb.DaggerheartLoadoutSwapRequest{
			CardId:     "card-1",
			RecallCost: 1,
		},
	})
	if err != nil {
		t.Fatalf("SwapLoadout returned error: %v", err)
	}
	if resp.State.Stress != 2 {
		t.Fatalf("stress = %d, want 2 (3 - 1 recall cost)", resp.State.Stress)
	}
}

func TestSwapLoadout_InRestSkipsRecallCost(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.SwapLoadout(ctx, &pb.DaggerheartSwapLoadoutRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Swap: &pb.DaggerheartLoadoutSwapRequest{
			CardId:     "card-1",
			RecallCost: 2,
			InRest:     true,
		},
	})
	if err != nil {
		t.Fatalf("SwapLoadout returned error: %v", err)
	}
	if resp.State.Stress != 3 {
		t.Fatalf("stress = %d, want 3 (in-rest should skip recall cost)", resp.State.Stress)
	}
}

// --- ApplyDeathMove tests ---

func TestApplyDeathMove_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyDeathMove(context.Background(), &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId: "c1", CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyDeathMove_MissingSeedFunc(t *testing.T) {
	svc := &DaggerheartService{
		stores: Stores{
			Campaign:    newFakeCampaignStore(),
			Daggerheart: newFakeDaggerheartStore(),
			Event:       newFakeActionEventStore(),
		},
	}
	_, err := svc.ApplyDeathMove(context.Background(), &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId: "c1", CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyDeathMove_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDeathMove(ctx, &pb.DaggerheartApplyDeathMoveRequest{
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDeathMove_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyDeathMove(context.Background(), &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDeathMove_UnspecifiedMove(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDeathMove(ctx, &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_UNSPECIFIED,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDeathMove_HpClearOnNonRiskItAll(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	hpClear := int32(1)
	_, err := svc.ApplyDeathMove(ctx, &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH,
		HpClear:     &hpClear,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDeathMove_HpNotZero(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDeathMove(ctx, &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestApplyDeathMove_AlreadyDead(t *testing.T) {
	svc := newActionTestService()
	// Set up state with hp=0 and life_state=dead
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.states["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          0,
		LifeState:   daggerheart.LifeStateDead,
	}
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDeathMove(ctx, &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestApplyDeathMove_AvoidDeath_Success(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.states["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          0,
		Hope:        2,
		HopeMax:     daggerheart.HopeMax,
		Stress:      1,
		LifeState:   daggerheart.LifeStateAlive,
	}
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyDeathMove(ctx, &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH,
	})
	if err != nil {
		t.Fatalf("ApplyDeathMove returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character_id = %q, want char-1", resp.CharacterId)
	}
	if resp.Result == nil {
		t.Fatal("expected result in response")
	}
}

// --- ResolveBlazeOfGlory tests ---

func TestResolveBlazeOfGlory_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ResolveBlazeOfGlory(context.Background(), &pb.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId: "c1", CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestResolveBlazeOfGlory_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ResolveBlazeOfGlory(ctx, &pb.DaggerheartResolveBlazeOfGloryRequest{
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestResolveBlazeOfGlory_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ResolveBlazeOfGlory(context.Background(), &pb.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestResolveBlazeOfGlory_CharacterAlreadyDead(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.states["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		LifeState:   daggerheart.LifeStateDead,
	}
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ResolveBlazeOfGlory(ctx, &pb.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestResolveBlazeOfGlory_NotInBlazeState(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ResolveBlazeOfGlory(ctx, &pb.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestResolveBlazeOfGlory_Success(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.states["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          0,
		LifeState:   daggerheart.LifeStateBlazeOfGlory,
	}
	charStore := svc.stores.Character.(*fakeCharacterStore)
	charStore.characters["camp-1:char-1"] = character.Character{
		ID:         "char-1",
		CampaignID: "camp-1",
		Name:       "Hero",
		Kind:       character.CharacterKindPC,
	}

	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ResolveBlazeOfGlory(ctx, &pb.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	if err != nil {
		t.Fatalf("ResolveBlazeOfGlory returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character_id = %q, want char-1", resp.CharacterId)
	}
	if resp.Result == nil {
		t.Fatal("expected result in response")
	}
	if resp.Result.LifeState != pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD {
		t.Fatalf("life_state = %v, want DEAD", resp.Result.LifeState)
	}
}

// --- ApplyDamage tests ---

func TestApplyDamage_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyDamage(context.Background(), &pb.DaggerheartApplyDamageRequest{
		CampaignId: "c1", CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyDamage_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{CharacterId: "ch1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDamage_MissingCharacterId(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{CampaignId: "camp-1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDamage_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyDamage(context.Background(), &pb.DaggerheartApplyDamageRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDamage_MissingDamage(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDamage_NegativeAmount(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     -1,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDamage_UnspecifiedType(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     2,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED,
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDamage_Success(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     3,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if err != nil {
		t.Fatalf("ApplyDamage returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character_id = %q, want char-1", resp.CharacterId)
	}
	if resp.State == nil {
		t.Fatal("expected state in response")
	}
	if resp.State.Hp >= 6 {
		t.Fatalf("hp = %d, expected less than 6 after 3 damage", resp.State.Hp)
	}
}

func TestApplyDamage_WithArmorMitigation(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")

	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	profile := dhStore.profiles["camp-1:char-1"]
	profile.MajorThreshold = 3
	profile.SevereThreshold = 6
	profile.ArmorMax = 1
	dhStore.profiles["camp-1:char-1"] = profile

	state := dhStore.states["camp-1:char-1"]
	state.Hp = 6
	state.Armor = 1
	dhStore.states["camp-1:char-1"] = state

	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     4,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if err != nil {
		t.Fatalf("ApplyDamage returned error: %v", err)
	}

	eventStore := svc.stores.Event.(*fakeEventStore)
	events := eventStore.events["camp-1"]
	if len(events) == 0 {
		t.Fatal("expected damage event")
	}
	last := events[len(events)-1]
	if last.Type != daggerheart.EventTypeDamageApplied {
		t.Fatalf("last event type = %s, want %s", last.Type, daggerheart.EventTypeDamageApplied)
	}
	var payload daggerheart.DamageAppliedPayload
	if err := json.Unmarshal(last.PayloadJSON, &payload); err != nil {
		t.Fatalf("decode damage payload: %v", err)
	}
	if payload.ArmorSpent != 1 {
		t.Fatalf("armor_spent = %d, want 1", payload.ArmorSpent)
	}
	if payload.Marks != 1 {
		t.Fatalf("marks = %d, want 1", payload.Marks)
	}
	if payload.Severity != "minor" {
		t.Fatalf("severity = %s, want minor", payload.Severity)
	}
}

func TestApplyDamage_RequireDamageRollWithoutSeq(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     3,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
		RequireDamageRoll: true,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyConditions_LifeStateOnly(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		LifeState:   pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS,
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if resp.State == nil {
		t.Fatal("expected state in response")
	}
	if resp.State.LifeState != pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS {
		t.Fatalf("life_state = %v, want UNCONSCIOUS", resp.State.LifeState)
	}

	eventStore := svc.stores.Event.(*fakeEventStore)
	events := eventStore.events["camp-1"]
	if len(events) == 0 {
		t.Fatal("expected events")
	}
	last := events[len(events)-1]
	if last.Type != daggerheart.EventTypeCharacterStatePatched {
		t.Fatalf("last event type = %s, want %s", last.Type, daggerheart.EventTypeCharacterStatePatched)
	}
}

func TestApplyConditions_LifeStateNoChange(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		LifeState:   pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestApplyConditions_InvalidStoredLifeState(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.states["camp-1:char-1"]
	state.LifeState = "not-a-life-state"
	dhStore.states["camp-1:char-1"] = state

	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		LifeState:   pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyConditions_NoConditionChanges(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.states["camp-1:char-1"]
	state.Conditions = []string{"vulnerable"}
	dhStore.states["camp-1:char-1"] = state

	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

// --- ApplyRest tests ---

func TestApplyRest_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyRest(context.Background(), &pb.DaggerheartApplyRestRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyRest_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRest_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyRest(context.Background(), &pb.DaggerheartApplyRestRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRest_MissingRest(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRest_UnspecifiedRestType(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId: "camp-1",
		Rest: &pb.DaggerheartRestRequest{
			RestType: pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_UNSPECIFIED,
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRest_ShortRest_Success(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId:   "camp-1",
		CharacterIds: []string{"char-1"},
		Rest: &pb.DaggerheartRestRequest{
			RestType:  pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_SHORT,
			PartySize: 3,
		},
	})
	if err != nil {
		t.Fatalf("ApplyRest returned error: %v", err)
	}
	if resp.Snapshot == nil {
		t.Fatal("expected snapshot in response")
	}
}

func TestApplyRest_LongRest_Success(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyRest(ctx, &pb.DaggerheartApplyRestRequest{
		CampaignId:   "camp-1",
		CharacterIds: []string{"char-1"},
		Rest: &pb.DaggerheartRestRequest{
			RestType:  pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG,
			PartySize: 3,
		},
	})
	if err != nil {
		t.Fatalf("ApplyRest returned error: %v", err)
	}
	if resp.Snapshot == nil {
		t.Fatal("expected snapshot in response")
	}
}

// --- ApplyGmMove tests ---

func TestApplyGmMove_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyGmMove_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		SessionId: "sess-1", Move: "test_move",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyGmMove_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1", Move: "test_move",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyGmMove_MissingMove(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyGmMove_NegativeFearSpent(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Move: "test_move", FearSpent: -1,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyGmMove_Success(t *testing.T) {
	svc := newActionTestService()
	resp, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Move: "change_environment",
	})
	if err != nil {
		t.Fatalf("ApplyGmMove returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response")
	}
}

func TestApplyGmMove_WithFearSpent(t *testing.T) {
	svc := newActionTestService()
	// Pre-populate GM fear in snapshot
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.snapshots["camp-1"] = storage.DaggerheartSnapshot{CampaignID: "camp-1", GMFear: 3}
	resp, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Move: "change_environment", FearSpent: 1,
	})
	if err != nil {
		t.Fatalf("ApplyGmMove returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response")
	}
}

// --- CreateCountdown tests ---

func TestCreateCountdown_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.CreateCountdown(context.Background(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestCreateCountdown_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.CreateCountdown(context.Background(), &pb.DaggerheartCreateCountdownRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCountdown_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.CreateCountdown(context.Background(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCountdown_MissingName(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.CreateCountdown(context.Background(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCountdown_InvalidMax(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.CreateCountdown(context.Background(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		Name:       "Test Countdown",
		Kind:       pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Direction:  pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:        0,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCountdown_CurrentOutOfRange(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.CreateCountdown(context.Background(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		Name:       "Test Countdown",
		Kind:       pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Direction:  pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:        4,
		Current:    5,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateCountdown_Success(t *testing.T) {
	svc := newActionTestService()
	resp, err := svc.CreateCountdown(context.Background(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		Name:       "Test Countdown",
		Kind:       pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Direction:  pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:        4,
		Current:    0,
	})
	if err != nil {
		t.Fatalf("CreateCountdown returned error: %v", err)
	}
	if resp.Countdown == nil {
		t.Fatal("expected countdown in response")
	}
	if resp.Countdown.Name != "Test Countdown" {
		t.Fatalf("name = %q, want Test Countdown", resp.Countdown.Name)
	}
}

// --- UpdateCountdown tests ---

func TestUpdateCountdown_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.UpdateCountdown(context.Background(), &pb.DaggerheartUpdateCountdownRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestUpdateCountdown_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.UpdateCountdown(context.Background(), &pb.DaggerheartUpdateCountdownRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateCountdown_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.UpdateCountdown(context.Background(), &pb.DaggerheartUpdateCountdownRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateCountdown_MissingCountdownId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.UpdateCountdown(context.Background(), &pb.DaggerheartUpdateCountdownRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateCountdown_NoDeltaOrCurrent(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.UpdateCountdown(context.Background(), &pb.DaggerheartUpdateCountdownRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CountdownId: "cd-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateCountdown_Success(t *testing.T) {
	svc := newActionTestService()
	// First create a countdown
	createResp, err := svc.CreateCountdown(context.Background(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		Name:        "Update Test",
		Kind:        pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Direction:   pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:         4,
		Current:     0,
		CountdownId: "cd-update",
	})
	if err != nil {
		t.Fatalf("CreateCountdown returned error: %v", err)
	}
	cdID := createResp.Countdown.CountdownId

	resp, err := svc.UpdateCountdown(context.Background(), &pb.DaggerheartUpdateCountdownRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CountdownId: cdID, Delta: 1,
	})
	if err != nil {
		t.Fatalf("UpdateCountdown returned error: %v", err)
	}
	if resp.Countdown == nil {
		t.Fatal("expected countdown in response")
	}
	if resp.After != 1 {
		t.Fatalf("after = %d, want 1", resp.After)
	}
}

// --- DeleteCountdown tests ---

func TestDeleteCountdown_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.DeleteCountdown(context.Background(), &pb.DaggerheartDeleteCountdownRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestDeleteCountdown_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.DeleteCountdown(context.Background(), &pb.DaggerheartDeleteCountdownRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDeleteCountdown_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.DeleteCountdown(context.Background(), &pb.DaggerheartDeleteCountdownRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDeleteCountdown_MissingCountdownId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.DeleteCountdown(context.Background(), &pb.DaggerheartDeleteCountdownRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDeleteCountdown_Success(t *testing.T) {
	svc := newActionTestService()
	// First create a countdown
	createResp, err := svc.CreateCountdown(context.Background(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		Name:        "Delete Test",
		Kind:        pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_CONSEQUENCE,
		Direction:   pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:         4,
		Current:     0,
		CountdownId: "cd-delete",
	})
	if err != nil {
		t.Fatalf("CreateCountdown returned error: %v", err)
	}
	cdID := createResp.Countdown.CountdownId

	resp, err := svc.DeleteCountdown(context.Background(), &pb.DaggerheartDeleteCountdownRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CountdownId: cdID,
	})
	if err != nil {
		t.Fatalf("DeleteCountdown returned error: %v", err)
	}
	if resp.CountdownId != cdID {
		t.Fatalf("countdown_id = %q, want %q", resp.CountdownId, cdID)
	}
}

// --- ApplyAdversaryDamage tests ---

func newAdversaryDamageTestService() *DaggerheartService {
	svc := newAdversaryTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartAdversaryStore)
	dhStore.adversaries["camp-1:adv-1"] = storage.DaggerheartAdversary{
		AdversaryID: "adv-1",
		CampaignID:  "camp-1",
		SessionID:   "sess-1",
		Name:        "Goblin",
		HP:          8,
		HPMax:       8,
		Armor:       1,
		Major:       4,
		Severe:      7,
	}
	return svc
}

func TestApplyAdversaryDamage_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyAdversaryDamage(context.Background(), &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId: "c1", AdversaryId: "a1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyAdversaryDamage_MissingCampaignId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryDamage_MissingAdversaryId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryDamage_MissingSessionId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.ApplyAdversaryDamage(context.Background(), &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId: "camp-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryDamage_MissingDamage(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId: "camp-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryDamage_NegativeAmount(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     -1,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryDamage_UnspecifiedType(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     2,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED,
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryDamage_Success(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     5, // major damage (>=4, <7)
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryDamage returned error: %v", err)
	}
	if resp.AdversaryId != "adv-1" {
		t.Fatalf("adversary_id = %q, want adv-1", resp.AdversaryId)
	}
	if resp.Adversary == nil {
		t.Fatal("expected adversary in response")
	}
}

func TestApplyAdversaryDamage_DirectDamage(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     3,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
			Direct:     true,
		},
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryDamage returned error: %v", err)
	}
	if resp.Adversary == nil {
		t.Fatal("expected adversary in response")
	}
}

// --- ApplyAdversaryConditions tests ---

func TestApplyAdversaryConditions_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyAdversaryConditions(context.Background(), &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId: "c1", AdversaryId: "a1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyAdversaryConditions_MissingCampaignId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		AdversaryId: "adv-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryConditions_MissingAdversaryId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId: "camp-1",
		Add:        []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryConditions_MissingSessionId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.ApplyAdversaryConditions(context.Background(), &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryConditions_NoConditions(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryConditions_ConflictAddRemoveSame(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
		Remove:      []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryConditions_AddCondition_Success(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryConditions returned error: %v", err)
	}
	if resp.AdversaryId != "adv-1" {
		t.Fatalf("adversary_id = %q, want adv-1", resp.AdversaryId)
	}
	if len(resp.Added) == 0 {
		t.Fatal("expected added conditions")
	}
}

func TestApplyAdversaryConditions_RemoveCondition_Success(t *testing.T) {
	svc := newAdversaryDamageTestService()
	// Pre-populate a condition on the adversary.
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartAdversaryStore)
	adv := dhStore.adversaries["camp-1:adv-1"]
	adv.Conditions = []string{"vulnerable"}
	dhStore.adversaries["camp-1:adv-1"] = adv

	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Remove:      []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryConditions returned error: %v", err)
	}
	if len(resp.Removed) == 0 {
		t.Fatal("expected removed conditions")
	}
}

func TestApplyAdversaryConditions_NoChanges(t *testing.T) {
	svc := newAdversaryDamageTestService()
	// Pre-populate a condition that we try to re-add.
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartAdversaryStore)
	adv := dhStore.adversaries["camp-1:adv-1"]
	adv.Conditions = []string{"vulnerable"}
	dhStore.adversaries["camp-1:adv-1"] = adv

	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

// --- ApplyConditions gap fills ---

func TestApplyConditions_AddCondition_Success(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if len(resp.Added) == 0 {
		t.Fatal("expected added conditions")
	}
}

func TestApplyConditions_RemoveCondition_Success(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.states["camp-1:char-1"]
	state.Conditions = []string{"hidden", "vulnerable"}
	dhStore.states["camp-1:char-1"] = state

	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Remove:      []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if len(resp.Removed) == 0 {
		t.Fatal("expected removed conditions")
	}
}

func TestApplyConditions_AddAndRemove(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.states["camp-1:char-1"]
	state.Conditions = []string{"hidden"}
	dhStore.states["camp-1:char-1"] = state

	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
		Remove:      []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if len(resp.Added) == 0 {
		t.Fatal("expected added conditions")
	}
	if len(resp.Removed) == 0 {
		t.Fatal("expected removed conditions")
	}
}

func TestApplyConditions_ConflictAddRemoveSame(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
		Remove:      []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

// --- ApplyGmMove gap fills ---

func TestApplyGmMove_FearSpentExceedsAvailable(t *testing.T) {
	svc := newActionTestService()
	// Snapshot has 0 fear.
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Move: "test_move", FearSpent: 10,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyGmMove_CampaignNotFound(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "nonexistent", SessionId: "sess-1", Move: "test",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyGmMove_SessionNotFound(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1", SessionId: "nonexistent", Move: "test",
	})
	assertStatusCode(t, err, codes.Internal)
}

// --- SessionActionRoll tests ---

func TestSessionActionRoll_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionActionRoll_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
		SessionId: "sess-1", CharacterId: "char-1", Trait: "agility",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionActionRoll_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
		CampaignId: "camp-1", CharacterId: "char-1", Trait: "agility",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionActionRoll_MissingCharacterId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Trait: "agility",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionActionRoll_MissingTrait(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionActionRoll_Success(t *testing.T) {
	svc := newActionTestService()
	resp, err := svc.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Trait:       "agility",
		Difficulty:  10,
	})
	if err != nil {
		t.Fatalf("SessionActionRoll returned error: %v", err)
	}
	if resp.RollSeq == 0 {
		t.Fatal("expected non-zero roll seq")
	}
	if resp.Rng == nil {
		t.Fatal("expected rng in response")
	}
}

func TestSessionActionRoll_WithModifiers(t *testing.T) {
	svc := newActionTestService()
	resp, err := svc.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Trait:       "agility",
		Difficulty:  10,
		Modifiers: []*pb.ActionRollModifier{
			{Value: 2, Source: "experience"},
		},
	})
	if err != nil {
		t.Fatalf("SessionActionRoll returned error: %v", err)
	}
	if resp.RollSeq == 0 {
		t.Fatal("expected non-zero roll seq")
	}
}

// --- SessionDamageRoll tests ---

func TestSessionDamageRoll_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionDamageRoll(context.Background(), &pb.SessionDamageRollRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionDamageRoll_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionDamageRoll(context.Background(), &pb.SessionDamageRollRequest{
		SessionId: "sess-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionDamageRoll_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionDamageRoll(context.Background(), &pb.SessionDamageRollRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionDamageRoll_MissingCharacterId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionDamageRoll(context.Background(), &pb.SessionDamageRollRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionDamageRoll_MissingDice(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionDamageRoll(context.Background(), &pb.SessionDamageRollRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionDamageRoll_Success(t *testing.T) {
	svc := newActionTestService()
	resp, err := svc.SessionDamageRoll(context.Background(), &pb.SessionDamageRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Dice:        []*pb.DiceSpec{{Sides: 6, Count: 2}},
	})
	if err != nil {
		t.Fatalf("SessionDamageRoll returned error: %v", err)
	}
	if resp.RollSeq == 0 {
		t.Fatal("expected non-zero roll seq")
	}
	if resp.Total == 0 {
		t.Fatal("expected non-zero total")
	}
}

// --- SessionAttackFlow tests ---

func TestSessionAttackFlow_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionAttackFlow_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAttackFlow_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAttackFlow_MissingCharacterId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAttackFlow_MissingTrait(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAttackFlow_MissingTargetId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CharacterId: "char-1", Trait: "agility",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAttackFlow_MissingDamage(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CharacterId: "char-1", Trait: "agility", TargetId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAttackFlow_MissingDamageType(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CharacterId: "char-1", Trait: "agility", TargetId: "adv-1",
		Damage: &pb.DaggerheartAttackDamageSpec{},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

// --- SessionReactionFlow tests ---

func TestSessionReactionFlow_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionReactionFlow(context.Background(), &pb.SessionReactionFlowRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionReactionFlow_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionReactionFlow(context.Background(), &pb.SessionReactionFlowRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionReactionFlow_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionReactionFlow(context.Background(), &pb.SessionReactionFlowRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionReactionFlow_MissingCharacterId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionReactionFlow(context.Background(), &pb.SessionReactionFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionReactionFlow_MissingTrait(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionReactionFlow(context.Background(), &pb.SessionReactionFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionReactionFlow_Success(t *testing.T) {
	svc := newActionTestService()
	ctx := grpcmeta.WithRequestID(context.Background(), "req-reaction-1")
	resp, err := svc.SessionReactionFlow(ctx, &pb.SessionReactionFlowRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Trait:       "agility",
		Difficulty:  10,
	})
	if err != nil {
		t.Fatalf("SessionReactionFlow returned error: %v", err)
	}
	if resp.ActionRoll == nil {
		t.Fatal("expected action roll in response")
	}
	if resp.RollOutcome == nil {
		t.Fatal("expected roll outcome in response")
	}
	if resp.ReactionOutcome == nil {
		t.Fatal("expected reaction outcome in response")
	}
}

// --- SessionAdversaryAttackRoll tests ---

func TestSessionAdversaryAttackRoll_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionAdversaryAttackRoll(context.Background(), &pb.SessionAdversaryAttackRollRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionAdversaryAttackRoll_MissingCampaignId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackRoll(context.Background(), &pb.SessionAdversaryAttackRollRequest{
		SessionId: "sess-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackRoll_MissingSessionId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackRoll(context.Background(), &pb.SessionAdversaryAttackRollRequest{
		CampaignId: "camp-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackRoll_MissingAdversaryId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackRoll(context.Background(), &pb.SessionAdversaryAttackRollRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackRoll_Success(t *testing.T) {
	svc := newAdversaryDamageTestService()
	resp, err := svc.SessionAdversaryAttackRoll(context.Background(), &pb.SessionAdversaryAttackRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
	})
	if err != nil {
		t.Fatalf("SessionAdversaryAttackRoll returned error: %v", err)
	}
	if resp.RollSeq == 0 {
		t.Fatal("expected non-zero roll seq")
	}
}

// --- SessionAdversaryActionCheck tests ---

func TestSessionAdversaryActionCheck_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionAdversaryActionCheck_MissingCampaignId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
		SessionId: "sess-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryActionCheck_MissingSessionId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
		CampaignId: "camp-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryActionCheck_MissingAdversaryId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryActionCheck_Success(t *testing.T) {
	svc := newAdversaryDamageTestService()
	resp, err := svc.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
		Difficulty:  10,
	})
	if err != nil {
		t.Fatalf("SessionAdversaryActionCheck returned error: %v", err)
	}
	if resp.RollSeq == 0 {
		t.Fatal("expected non-zero roll seq")
	}
}

// --- SessionAdversaryAttackFlow tests ---

func TestSessionAdversaryAttackFlow_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionAdversaryAttackFlow_MissingCampaignId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackFlow_MissingSessionId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackFlow_MissingAdversaryId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackFlow_MissingTargetId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackFlow_MissingDamage(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", AdversaryId: "adv-1", TargetId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackFlow_MissingDamageType(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", AdversaryId: "adv-1", TargetId: "char-1",
		Damage: &pb.DaggerheartAttackDamageSpec{},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

// --- SessionGroupActionFlow tests ---

func TestSessionGroupActionFlow_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionGroupActionFlow_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionGroupActionFlow_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionGroupActionFlow_MissingLeader(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionGroupActionFlow_MissingLeaderTrait(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", LeaderCharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionGroupActionFlow_MissingDifficulty(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", LeaderCharacterId: "char-1", LeaderTrait: "agility",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionGroupActionFlow_MissingSupporters(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionGroupActionFlow(context.Background(), &pb.SessionGroupActionFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", LeaderCharacterId: "char-1", LeaderTrait: "agility", Difficulty: 10,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

// --- SessionTagTeamFlow tests ---

func TestSessionTagTeamFlow_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionTagTeamFlow_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionTagTeamFlow_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionTagTeamFlow_MissingDifficulty(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionTagTeamFlow_MissingFirst(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Difficulty: 10,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionTagTeamFlow_MissingSecond(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Difficulty: 10,
		First: &pb.TagTeamParticipant{CharacterId: "char-1", Trait: "agility"},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionTagTeamFlow_SameParticipant(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.SessionTagTeamFlow(context.Background(), &pb.SessionTagTeamFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", Difficulty: 10,
		First:               &pb.TagTeamParticipant{CharacterId: "char-1", Trait: "agility"},
		Second:              &pb.TagTeamParticipant{CharacterId: "char-1", Trait: "strength"},
		SelectedCharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

// --- ApplyRollOutcome tests ---

func TestApplyRollOutcome_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyRollOutcome(context.Background(), &pb.ApplyRollOutcomeRequest{
		SessionId: "s1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyRollOutcome_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyRollOutcome(context.Background(), &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRollOutcome_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	ctx := withCampaignSessionMetadata(context.Background(), "camp-1", "")
	_, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		RollSeq: 1,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRollOutcome_MissingRollSeq(t *testing.T) {
	svc := newActionTestService()
	ctx := withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRollOutcome_Success(t *testing.T) {
	svc := newActionTestService()
	// First create a roll event via SessionActionRoll.
	// Must include request ID as ApplyRollOutcome requires it.
	rollCtx := grpcmeta.WithRequestID(context.Background(), "req-roll-1")
	rollResp, err := svc.SessionActionRoll(rollCtx, &pb.SessionActionRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Trait:       "agility",
		Difficulty:  10,
	})
	if err != nil {
		t.Fatalf("SessionActionRoll returned error: %v", err)
	}

	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-1")
	resp, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollResp.RollSeq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if resp.RollSeq == 0 {
		t.Fatal("expected roll seq in response")
	}
}

// --- ApplyAttackOutcome tests ---

func TestApplyAttackOutcome_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyAttackOutcome(context.Background(), &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "s1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyAttackOutcome_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyAttackOutcome(context.Background(), &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1, Targets: []string{"adv-1"},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAttackOutcome_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyAttackOutcome(context.Background(), &pb.DaggerheartApplyAttackOutcomeRequest{
		RollSeq: 1, Targets: []string{"adv-1"},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAttackOutcome_MissingRollSeq(t *testing.T) {
	svc := newActionTestService()
	ctx := withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyAttackOutcome(ctx, &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "sess-1", Targets: []string{"adv-1"},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAttackOutcome_MissingTargets(t *testing.T) {
	svc := newActionTestService()
	ctx := withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyAttackOutcome(ctx, &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

// --- ApplyAdversaryAttackOutcome tests ---

func TestApplyAdversaryAttackOutcome_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyAdversaryAttackOutcome(context.Background(), &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId: "s1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyAdversaryAttackOutcome_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyAdversaryAttackOutcome(context.Background(), &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1, Targets: []string{"char-1"},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryAttackOutcome_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyAdversaryAttackOutcome(context.Background(), &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		RollSeq: 1, Targets: []string{"char-1"},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryAttackOutcome_MissingRollSeq(t *testing.T) {
	svc := newActionTestService()
	ctx := withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyAdversaryAttackOutcome(ctx, &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId: "sess-1", Targets: []string{"char-1"},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryAttackOutcome_MissingTargets(t *testing.T) {
	svc := newActionTestService()
	ctx := withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyAdversaryAttackOutcome(ctx, &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

// --- ApplyReactionOutcome tests ---

func TestApplyReactionOutcome_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyReactionOutcome(context.Background(), &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: "s1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyReactionOutcome_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyReactionOutcome(context.Background(), &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyReactionOutcome_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyReactionOutcome(context.Background(), &pb.DaggerheartApplyReactionOutcomeRequest{
		RollSeq: 1,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyReactionOutcome_MissingRollSeq(t *testing.T) {
	svc := newActionTestService()
	ctx := withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyReactionOutcome(ctx, &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

// --- Success path tests for flow handlers ---

func TestSessionAttackFlow_Success(t *testing.T) {
	svc := newActionTestService()
	ctx := grpcmeta.WithRequestID(context.Background(), "req-attack-1")
	resp, err := svc.SessionAttackFlow(ctx, &pb.SessionAttackFlowRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Trait:       "agility",
		Difficulty:  10,
		TargetId:    "char-2",
		Damage: &pb.DaggerheartAttackDamageSpec{
			DamageType:         pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
			SourceCharacterIds: []string{"char-1"},
		},
		DamageDice: []*pb.DiceSpec{{Sides: 6, Count: 1}},
	})
	if err != nil {
		t.Fatalf("SessionAttackFlow returned error: %v", err)
	}
	if resp.ActionRoll == nil {
		t.Fatal("expected action roll in response")
	}
	if resp.AttackOutcome == nil {
		t.Fatal("expected attack outcome in response")
	}
}

func TestSessionAdversaryAttackFlow_Success(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := grpcmeta.WithRequestID(context.Background(), "req-adv-attack-1")
	resp, err := svc.SessionAdversaryAttackFlow(ctx, &pb.SessionAdversaryAttackFlowRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
		TargetId:    "char-1",
		Difficulty:  10,
		Damage: &pb.DaggerheartAttackDamageSpec{
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
		DamageDice: []*pb.DiceSpec{{Sides: 6, Count: 1}},
	})
	if err != nil {
		t.Fatalf("SessionAdversaryAttackFlow returned error: %v", err)
	}
	if resp.AttackRoll == nil {
		t.Fatal("expected attack roll in response")
	}
	if resp.AttackOutcome == nil {
		t.Fatal("expected attack outcome in response")
	}
}

func TestSessionGroupActionFlow_Success(t *testing.T) {
	svc := newActionTestService()
	ctx := grpcmeta.WithRequestID(context.Background(), "req-group-1")
	resp, err := svc.SessionGroupActionFlow(ctx, &pb.SessionGroupActionFlowRequest{
		CampaignId:        "camp-1",
		SessionId:         "sess-1",
		LeaderCharacterId: "char-1",
		LeaderTrait:       "agility",
		Difficulty:        10,
		Supporters: []*pb.GroupActionSupporter{
			{CharacterId: "char-2", Trait: "strength"},
		},
	})
	if err != nil {
		t.Fatalf("SessionGroupActionFlow returned error: %v", err)
	}
	if resp.LeaderRoll == nil {
		t.Fatal("expected leader roll in response")
	}
	if len(resp.SupporterRolls) == 0 {
		t.Fatal("expected supporter rolls in response")
	}
}

func TestSessionTagTeamFlow_Success(t *testing.T) {
	svc := newActionTestService()
	ctx := grpcmeta.WithRequestID(context.Background(), "req-tagteam-1")
	resp, err := svc.SessionTagTeamFlow(ctx, &pb.SessionTagTeamFlowRequest{
		CampaignId:          "camp-1",
		SessionId:           "sess-1",
		Difficulty:          10,
		First:               &pb.TagTeamParticipant{CharacterId: "char-1", Trait: "agility"},
		Second:              &pb.TagTeamParticipant{CharacterId: "char-2", Trait: "strength"},
		SelectedCharacterId: "char-1",
	})
	if err != nil {
		t.Fatalf("SessionTagTeamFlow returned error: %v", err)
	}
	if resp.FirstRoll == nil {
		t.Fatal("expected first roll in response")
	}
	if resp.SecondRoll == nil {
		t.Fatal("expected second roll in response")
	}
}

func TestApplyAttackOutcome_Success(t *testing.T) {
	svc := newActionTestService()
	// Create an action roll event first.
	rollCtx := grpcmeta.WithRequestID(context.Background(), "req-atk-outcome-1")
	rollResp, err := svc.SessionActionRoll(rollCtx, &pb.SessionActionRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Trait:       "agility",
		Difficulty:  10,
	})
	if err != nil {
		t.Fatalf("SessionActionRoll returned error: %v", err)
	}

	ctx := grpcmeta.WithRequestID(
		withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"),
		"req-atk-outcome-1",
	)
	resp, err := svc.ApplyAttackOutcome(ctx, &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollResp.RollSeq,
		Targets:   []string{"char-2"},
	})
	if err != nil {
		t.Fatalf("ApplyAttackOutcome returned error: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("expected result in response")
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("expected attacker char-1, got %s", resp.CharacterId)
	}
}

func TestApplyAdversaryAttackOutcome_Success(t *testing.T) {
	svc := newAdversaryDamageTestService()
	// Create an adversary attack roll event first.
	rollCtx := grpcmeta.WithRequestID(context.Background(), "req-adv-atk-outcome-1")
	rollResp, err := svc.SessionAdversaryAttackRoll(rollCtx, &pb.SessionAdversaryAttackRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
	})
	if err != nil {
		t.Fatalf("SessionAdversaryAttackRoll returned error: %v", err)
	}

	ctx := grpcmeta.WithRequestID(
		withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"),
		"req-adv-atk-outcome-1",
	)
	resp, err := svc.ApplyAdversaryAttackOutcome(ctx, &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId:  "sess-1",
		RollSeq:    rollResp.RollSeq,
		Targets:    []string{"char-1"},
		Difficulty: 10,
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryAttackOutcome returned error: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("expected result in response")
	}
	if resp.AdversaryId != "adv-1" {
		t.Fatalf("expected adversary adv-1, got %s", resp.AdversaryId)
	}
}
