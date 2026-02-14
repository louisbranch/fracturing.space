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

	sessStore := newFakeSessionStore()
	sessStore.sessions["camp-1:sess-1"] = session.Session{
		ID:         "sess-1",
		CampaignID: "camp-1",
		Status:     session.SessionStatusActive,
	}

	return &DaggerheartService{
		stores: Stores{
			Campaign:    campaignStore,
			Daggerheart: dhStore,
			Event:       newFakeActionEventStore(),
			SessionGate: &fakeSessionGateStore{},
			Character:   newFakeCharacterStore(),
			Session:     sessStore,
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
