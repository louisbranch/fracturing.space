package outcometransport

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/test/mock/gamefakes"
	"google.golang.org/grpc/metadata"
)

var testTimestamp = time.Date(2026, time.February, 14, 0, 0, 0, 0, time.UTC)

type fakeSessionGateStore struct{}

func (s *fakeSessionGateStore) GetOpenSessionGate(context.Context, string, string) (storage.SessionGate, error) {
	return storage.SessionGate{}, storage.ErrNotFound
}

type fakeSessionSpotlightStore struct{}

func (s *fakeSessionSpotlightStore) GetSessionSpotlight(context.Context, string, string) (storage.SessionSpotlight, error) {
	return storage.SessionSpotlight{}, storage.ErrNotFound
}

type callbackRecorder struct {
	systemCommands []SystemCommandInput
	coreCommands   []CoreCommandInput
	stressCalls    []ApplyStressVulnerableConditionInput
}

func TestHandlerApplyAttackOutcomeSuccess(t *testing.T) {
	handler, events, _ := newTestHandler()
	roll := appendRollEvent(t, events, rollEventConfig{
		outcome: pb.Outcome_SUCCESS_WITH_HOPE.String(),
		metadata: workflowtransport.RollSystemMetadata{
			CharacterID: "char-1",
			RollKind:    pb.RollKind_ROLL_KIND_ACTION.String(),
			HopeFear:    workflowtransport.BoolPtr(true),
		},
	})

	resp, err := handler.ApplyAttackOutcome(testSessionContext("camp-1", "sess-1"), &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   roll.Seq,
		Targets:   []string{"char-2"},
	})
	if err != nil {
		t.Fatalf("ApplyAttackOutcome returned error: %v", err)
	}
	if got := resp.GetCharacterId(); got != "char-1" {
		t.Fatalf("character_id = %q, want char-1", got)
	}
	if got := resp.GetResult().GetFlavor(); got != "HOPE" {
		t.Fatalf("flavor = %q, want HOPE", got)
	}
	if !resp.GetResult().GetSuccess() {
		t.Fatal("expected success result")
	}
}

func TestHandlerApplyReactionOutcomeCritNegatesEffects(t *testing.T) {
	handler, events, _ := newTestHandler()
	roll := appendRollEvent(t, events, rollEventConfig{
		outcome: pb.Outcome_CRITICAL_SUCCESS.String(),
		metadata: workflowtransport.RollSystemMetadata{
			CharacterID: "char-1",
			RollKind:    pb.RollKind_ROLL_KIND_REACTION.String(),
			Crit:        workflowtransport.BoolPtr(true),
		},
	})

	resp, err := handler.ApplyReactionOutcome(testSessionContext("camp-1", "sess-1"), &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   roll.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyReactionOutcome returned error: %v", err)
	}
	if !resp.GetResult().GetCrit() {
		t.Fatal("expected crit result")
	}
	if !resp.GetResult().GetEffectsNegated() {
		t.Fatal("expected crit to negate effects")
	}
}

func TestHandlerBuildApplyRollOutcomeIdempotentResponseIncludesGMFear(t *testing.T) {
	handler, _, _ := newTestHandler()

	resp, err := handler.buildApplyRollOutcomeIdempotentResponse(context.Background(), "camp-1", 7, []string{"char-1"}, true, true)
	if err != nil {
		t.Fatalf("buildApplyRollOutcomeIdempotentResponse returned error: %v", err)
	}
	if got := len(resp.GetUpdated().GetCharacterStates()); got != 1 {
		t.Fatalf("character_states len = %d, want 1", got)
	}
	if got := resp.GetUpdated().GetGmFear(); got != 2 {
		t.Fatalf("gm_fear = %d, want 2", got)
	}
}

func TestHandlerBuildGMConsequenceOutcomeEffectsAddsGateAndSpotlight(t *testing.T) {
	handler, _, _ := newTestHandler()

	effects, err := handler.buildGMConsequenceOutcomeEffects(context.Background(), "camp-1", "sess-1", 9, "req-1")
	if err != nil {
		t.Fatalf("buildGMConsequenceOutcomeEffects returned error: %v", err)
	}
	if got := len(effects); got != 2 {
		t.Fatalf("effects len = %d, want 2", got)
	}
	if got := effects[0].Type; got != "session.gate_opened" {
		t.Fatalf("first effect type = %q", got)
	}
	if got := effects[1].Type; got != "session.spotlight_set" {
		t.Fatalf("second effect type = %q", got)
	}
}

func TestHandlerSessionRequestEventExistsMatchesEvent(t *testing.T) {
	handler, events, _ := newTestHandler()
	roll := appendRollEvent(t, events, rollEventConfig{})
	if _, err := events.AppendEvent(context.Background(), event.Event{
		CampaignID: "camp-1",
		Timestamp:  testTimestamp,
		Type:       eventTypeActionOutcomeApplied,
		SessionID:  "sess-1",
		RequestID:  "req-1",
		EntityType: "outcome",
		EntityID:   "req-1",
	}); err != nil {
		t.Fatalf("append outcome event: %v", err)
	}

	exists, err := handler.sessionRequestEventExists(context.Background(), "camp-1", "sess-1", roll.Seq, "req-1", eventTypeActionOutcomeApplied, "req-1")
	if err != nil {
		t.Fatalf("sessionRequestEventExists returned error: %v", err)
	}
	if !exists {
		t.Fatal("expected matching session event to exist")
	}
}

func TestHandlerApplyRollOutcomeSuccess(t *testing.T) {
	handler, events, recorder := newTestHandler()
	roll := appendRollEvent(t, events, rollEventConfig{
		requestID: "roll-hope-1",
		outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		metadata: workflowtransport.RollSystemMetadata{
			CharacterID: "char-1",
			RollKind:    pb.RollKind_ROLL_KIND_ACTION.String(),
			HopeFear:    workflowtransport.BoolPtr(true),
			GMMove:      workflowtransport.BoolPtr(false),
		},
	})

	resp, err := handler.ApplyRollOutcome(testSessionContext("camp-1", "sess-1"), &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   roll.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if got := resp.GetUpdated().GetCharacterStates()[0].GetHope(); got != 3 {
		t.Fatalf("updated hope = %d, want 3", got)
	}
	if got := len(recorder.systemCommands); got != 1 {
		t.Fatalf("system command count = %d, want 1", got)
	}
	if got := recorder.systemCommands[0].CommandType; got != commandTypeDaggerheartCharacterStatePatch {
		t.Fatalf("system command type = %q", got)
	}
	if got := len(recorder.coreCommands); got != 1 {
		t.Fatalf("core command count = %d, want 1", got)
	}
	if got := recorder.coreCommands[0].CommandType; got != commandTypeActionOutcomeApply {
		t.Fatalf("core command type = %q", got)
	}
	if got := len(recorder.stressCalls); got != 1 {
		t.Fatalf("stress call count = %d, want 1", got)
	}
}

func TestHandlerApplyRollOutcomeIdempotentFearRepairsGMConsequence(t *testing.T) {
	handler, events, recorder := newTestHandler()
	roll := appendRollEvent(t, events, rollEventConfig{
		requestID: "roll-fear-1",
		outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		metadata: workflowtransport.RollSystemMetadata{
			CharacterID: "char-1",
			RollKind:    pb.RollKind_ROLL_KIND_ACTION.String(),
			HopeFear:    workflowtransport.BoolPtr(true),
			GMMove:      workflowtransport.BoolPtr(true),
		},
	})
	if _, err := events.AppendEvent(context.Background(), event.Event{
		CampaignID: "camp-1",
		Timestamp:  testTimestamp,
		Type:       eventTypeActionOutcomeApplied,
		SessionID:  "sess-1",
		RequestID:  "roll-fear-1",
		EntityType: "outcome",
		EntityID:   "roll-fear-1",
	}); err != nil {
		t.Fatalf("append outcome event: %v", err)
	}

	resp, err := handler.ApplyRollOutcome(testSessionContext("camp-1", "sess-1"), &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		SceneId:   "scene-1",
		RollSeq:   roll.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if !resp.GetRequiresComplication() {
		t.Fatal("expected complication on fear retry")
	}
	if got := len(recorder.coreCommands); got != 2 {
		t.Fatalf("core command count = %d, want 2", got)
	}
	if recorder.coreCommands[0].CommandType != commandTypeSessionGateOpen {
		t.Fatalf("first core command type = %q", recorder.coreCommands[0].CommandType)
	}
	if recorder.coreCommands[1].CommandType != commandTypeSessionSpotlightSet {
		t.Fatalf("second core command type = %q", recorder.coreCommands[1].CommandType)
	}
}

func TestHandlerApplyAdversaryAttackOutcomeSuccess(t *testing.T) {
	handler, events, _ := newTestHandler()
	roll := appendRollEvent(t, events, rollEventConfig{
		outcome: pb.Outcome_SUCCESS_WITH_FEAR.String(),
		metadata: workflowtransport.RollSystemMetadata{
			CharacterID: "adv-1",
			AdversaryID: "adv-1",
			RollKind:    "adversary_roll",
			Roll:        workflowtransport.IntPtr(20),
			Modifier:    workflowtransport.IntPtr(2),
			Total:       workflowtransport.IntPtr(22),
		},
	})

	resp, err := handler.ApplyAdversaryAttackOutcome(testSessionContext("camp-1", "sess-1"), &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId:  "sess-1",
		RollSeq:    roll.Seq,
		Difficulty: 18,
		Targets:    []string{"char-1"},
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryAttackOutcome returned error: %v", err)
	}
	if !resp.GetResult().GetSuccess() {
		t.Fatal("expected successful adversary attack")
	}
	if !resp.GetResult().GetCrit() {
		t.Fatal("expected crit from natural 20")
	}
}

type rollEventConfig struct {
	campaignID string
	sessionID  string
	requestID  string
	outcome    string
	metadata   workflowtransport.RollSystemMetadata
}

func appendRollEvent(t *testing.T, store *gamefakes.EventStore, config rollEventConfig) event.Event {
	t.Helper()
	campaignID := config.campaignID
	if campaignID == "" {
		campaignID = "camp-1"
	}
	sessionID := config.sessionID
	if sessionID == "" {
		sessionID = "sess-1"
	}
	requestID := config.requestID
	if requestID == "" {
		requestID = "req-1"
	}
	metadata := config.metadata
	if metadata.CharacterID == "" {
		metadata.CharacterID = "char-1"
	}
	if metadata.RollKind == "" {
		metadata.RollKind = pb.RollKind_ROLL_KIND_ACTION.String()
	}
	if metadata.HopeFear == nil {
		metadata.HopeFear = workflowtransport.BoolPtr(true)
	}
	outcome := config.outcome
	if outcome == "" {
		outcome = pb.Outcome_SUCCESS_WITH_HOPE.String()
	}

	payloadJSON, err := json.Marshal(eventPayload{
		RequestID:  requestID,
		RollSeq:    1,
		Outcome:    outcome,
		SystemData: metadata.MapValue(),
	})
	if err != nil {
		t.Fatalf("marshal roll payload: %v", err)
	}
	evt, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  ids.CampaignID(campaignID),
		Timestamp:   testTimestamp,
		Type:        eventTypeActionRollResolved,
		SessionID:   ids.SessionID(sessionID),
		RequestID:   requestID,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    requestID,
		PayloadJSON: payloadJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}
	return evt
}

type eventPayload struct {
	RequestID  string         `json:"request_id,omitempty"`
	RollSeq    uint64         `json:"roll_seq,omitempty"`
	Outcome    string         `json:"outcome,omitempty"`
	SystemData map[string]any `json:"system_data,omitempty"`
}

func newTestHandler() (*Handler, *gamefakes.EventStore, *callbackRecorder) {
	recorder := &callbackRecorder{}
	campaigns := gamefakes.NewCampaignStore()
	campaigns.Campaigns["camp-1"] = storage.CampaignRecord{
		ID:     "camp-1",
		Status: campaign.StatusActive,
		System: systembridge.SystemIDDaggerheart,
	}

	sessions := gamefakes.NewSessionStore()
	sessions.Sessions["camp-1:sess-1"] = storage.SessionRecord{
		ID:         "sess-1",
		CampaignID: "camp-1",
		Status:     session.StatusActive,
	}

	daggerheartStore := gamefakes.NewDaggerheartStore()
	daggerheartStore.Profiles["camp-1:char-1"] = projectionstore.DaggerheartCharacterProfile{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		StressMax:   6,
	}
	daggerheartStore.States["camp-1:char-1"] = projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hope:        2,
		HopeMax:     bridge.HopeMax,
		Stress:      3,
		Hp:          6,
	}
	daggerheartStore.Snapshots["camp-1"] = projectionstore.DaggerheartSnapshot{
		CampaignID: "camp-1",
		GMFear:     2,
	}

	events := gamefakes.NewEventStore()

	return NewHandler(Dependencies{
		Campaign:         campaigns,
		Session:          sessions,
		SessionGate:      &fakeSessionGateStore{},
		SessionSpotlight: &fakeSessionSpotlightStore{},
		Daggerheart:      daggerheartStore,
		Event:            events,
		ExecuteSystemCommand: func(_ context.Context, in SystemCommandInput) error {
			recorder.systemCommands = append(recorder.systemCommands, in)
			return nil
		},
		ExecuteCoreCommand: func(_ context.Context, in CoreCommandInput) error {
			recorder.coreCommands = append(recorder.coreCommands, in)
			return nil
		},
		ApplyStressVulnerableCondition: func(_ context.Context, in ApplyStressVulnerableConditionInput) error {
			recorder.stressCalls = append(recorder.stressCalls, in)
			return nil
		},
	}), events, recorder
}

func testSessionContext(campaignID, sessionID string) context.Context {
	md := metadata.Pairs(grpcmeta.CampaignIDHeader, campaignID, grpcmeta.SessionIDHeader, sessionID)
	return metadata.NewIncomingContext(context.Background(), md)
}
