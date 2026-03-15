package engine

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/replay"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

type trackingSnapshotStore struct {
	calls      int
	campaignID string
	lastSeq    uint64
	state      any
	err        error
}

func (s *trackingSnapshotStore) GetState(context.Context, string) (any, uint64, error) {
	return nil, 0, replay.ErrCheckpointNotFound
}

func (s *trackingSnapshotStore) SaveState(_ context.Context, campaignID string, lastSeq uint64, state any) error {
	s.calls++
	s.campaignID = campaignID
	s.lastSeq = lastSeq
	s.state = state
	return s.err
}

type orderedSnapshotStore struct {
	order *[]string
}

func (s *orderedSnapshotStore) GetState(context.Context, string) (any, uint64, error) {
	return nil, 0, replay.ErrCheckpointNotFound
}

func (s *orderedSnapshotStore) SaveState(context.Context, string, uint64, any) error {
	*s.order = append(*s.order, "snapshot")
	return nil
}

type orderedCheckpointStore struct {
	order *[]string
}

func (s *orderedCheckpointStore) Get(context.Context, string) (replay.Checkpoint, error) {
	return replay.Checkpoint{}, replay.ErrCheckpointNotFound
}

func (s *orderedCheckpointStore) Save(context.Context, replay.Checkpoint) error {
	*s.order = append(*s.order, "checkpoint")
	return nil
}

func TestExecute_RequiresGateStateLoaderForSessionScopedCommand(t *testing.T) {
	registry := command.NewRegistry()
	if err := registry.Register(command.Definition{
		Type:  command.Type("session.start"),
		Owner: command.OwnerCore,
		Gate:  command.GatePolicy{Scope: command.GateScopeSession},
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}

	handler := Handler{
		Commands: registry,
		Gate:     DecisionGate{Registry: registry},
		Decider:  fixedDecider{},
	}

	_, err := handler.Execute(context.Background(), command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("session.start"),
		ActorType:  command.ActorTypeSystem,
		SessionID:  "sess-1",
	})
	if !errors.Is(err, ErrGateStateLoaderRequired) {
		t.Fatalf("expected ErrGateStateLoaderRequired, got %v", err)
	}
}

func TestExecute_PropagatesGateLoaderError(t *testing.T) {
	registry := command.NewRegistry()
	if err := registry.Register(command.Definition{
		Type:  command.Type("session.start"),
		Owner: command.OwnerCore,
		Gate:  command.GatePolicy{Scope: command.GateScopeSession},
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}
	loaderErr := errors.New("gate loader boom")
	handler := Handler{
		Commands:        registry,
		Gate:            DecisionGate{Registry: registry},
		GateStateLoader: fakeGateLoader{err: loaderErr},
		Decider:         fixedDecider{},
	}

	_, err := handler.Execute(context.Background(), command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("session.start"),
		ActorType:  command.ActorTypeSystem,
		SessionID:  "sess-1",
	})
	if !errors.Is(err, loaderErr) {
		t.Fatalf("expected loader error, got %v", err)
	}
}

func TestExecute_RetriesRejectedDecisionWithFreshState(t *testing.T) {
	cmdRegistry := command.NewRegistry()
	if err := cmdRegistry.Register(command.Definition{
		Type:  command.Type("session.ai_turn.clear"),
		Owner: command.OwnerCore,
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}
	loader := &staleThenFreshStateLoader{
		loadState: aggregate.State{
			Session: session.State{},
		},
		freshState: aggregate.State{
			Session: session.State{
				Started:      true,
				SessionID:    "sess-1",
				AITurnStatus: session.AITurnStatusRunning,
				AITurnToken:  "turn-1",
			},
		},
	}
	journal := &fakeJournal{}
	handler := Handler{
		Commands:    cmdRegistry,
		Journal:     journal,
		StateLoader: loader,
		Decider:     aiTurnStateDecider{},
	}

	result, err := handler.Execute(context.Background(), command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("session.ai_turn.clear"),
		ActorType:  command.ActorTypeSystem,
		SessionID:  "sess-1",
		EntityType: "session",
		EntityID:   "sess-1",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if loader.loadCalls != 1 {
		t.Fatalf("load calls = %d, want 1", loader.loadCalls)
	}
	if loader.freshCalls != 1 {
		t.Fatalf("fresh load calls = %d, want 1", loader.freshCalls)
	}
	if len(result.Decision.Rejections) != 0 {
		t.Fatalf("rejections = %d, want 0", len(result.Decision.Rejections))
	}
	if len(result.Decision.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(result.Decision.Events))
	}
	if journal.nextSeq != 1 {
		t.Fatalf("journal next seq = %d, want 1", journal.nextSeq)
	}
}

type staleFreshErrorStateLoader struct {
	loadState  any
	freshErr   error
	loadCalls  int
	freshCalls int
}

func (l *staleFreshErrorStateLoader) Load(_ context.Context, _ command.Command) (any, error) {
	l.loadCalls++
	return l.loadState, nil
}

func (l *staleFreshErrorStateLoader) LoadFresh(_ context.Context, _ command.Command) (any, error) {
	l.freshCalls++
	return nil, l.freshErr
}

func TestExecute_DoesNotFreshReplayAcceptedDecision(t *testing.T) {
	cmdRegistry := command.NewRegistry()
	if err := cmdRegistry.Register(command.Definition{
		Type:  command.Type("session.ai_turn.clear"),
		Owner: command.OwnerCore,
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}
	loader := &staleThenFreshStateLoader{
		loadState: aggregate.State{
			Session: session.State{
				Started:      true,
				SessionID:    "sess-1",
				AITurnStatus: session.AITurnStatusRunning,
				AITurnToken:  "turn-1",
			},
		},
		freshState: aggregate.State{
			Session: session.State{},
		},
	}
	handler := Handler{
		Commands:    cmdRegistry,
		Journal:     &fakeJournal{},
		StateLoader: loader,
		Decider:     aiTurnStateDecider{},
	}

	_, err := handler.Execute(context.Background(), command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("session.ai_turn.clear"),
		ActorType:  command.ActorTypeSystem,
		SessionID:  "sess-1",
		EntityType: "session",
		EntityID:   "sess-1",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if loader.freshCalls != 0 {
		t.Fatalf("fresh load calls = %d, want 0", loader.freshCalls)
	}
}

func TestExecute_DoesNotRetryRejectedDecisionWithoutFreshLoader(t *testing.T) {
	cmdRegistry := command.NewRegistry()
	if err := cmdRegistry.Register(command.Definition{
		Type:  command.Type("session.ai_turn.clear"),
		Owner: command.OwnerCore,
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}
	loader := &trackingStateLoader{}
	handler := Handler{
		Commands:    cmdRegistry,
		Journal:     &fakeJournal{},
		StateLoader: loader,
		Decider:     aiTurnStateDecider{},
	}

	result, err := handler.Execute(context.Background(), command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("session.ai_turn.clear"),
		ActorType:  command.ActorTypeSystem,
		SessionID:  "sess-1",
		EntityType: "session",
		EntityID:   "sess-1",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(result.Decision.Rejections) != 1 {
		t.Fatalf("rejections = %d, want 1", len(result.Decision.Rejections))
	}
	if loader.campaignIDs == nil || len(loader.campaignIDs) != 1 {
		t.Fatalf("load calls = %d, want 1", len(loader.campaignIDs))
	}
}

func TestExecute_PropagatesFreshReplayError(t *testing.T) {
	cmdRegistry := command.NewRegistry()
	if err := cmdRegistry.Register(command.Definition{
		Type:  command.Type("session.ai_turn.clear"),
		Owner: command.OwnerCore,
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}
	loaderErr := errors.New("fresh replay failed")
	loader := &staleFreshErrorStateLoader{
		loadState: aggregate.State{
			Session: session.State{},
		},
		freshErr: loaderErr,
	}
	handler := Handler{
		Commands:    cmdRegistry,
		Journal:     &fakeJournal{},
		StateLoader: loader,
		Decider:     aiTurnStateDecider{},
	}

	_, err := handler.Execute(context.Background(), command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("session.ai_turn.clear"),
		ActorType:  command.ActorTypeSystem,
		SessionID:  "sess-1",
		EntityType: "session",
		EntityID:   "sess-1",
	})
	if !errors.Is(err, loaderErr) {
		t.Fatalf("expected loader error, got %v", err)
	}
	if loader.loadCalls != 1 || loader.freshCalls != 1 {
		t.Fatalf("load/fresh calls = %d/%d, want 1/1", loader.loadCalls, loader.freshCalls)
	}
}

func TestShouldRetryRejectedDecisionWithFreshState(t *testing.T) {
	if shouldRetryRejectedDecisionWithFreshState(command.Command{Type: command.Type("other.command")}, command.Reject(command.Rejection{
		Code: rejectionCodeSessionAITurnNotActive,
	})) {
		t.Fatal("expected non-ai-turn command not to retry")
	}
	if shouldRetryRejectedDecisionWithFreshState(command.Command{Type: command.Type("session.ai_turn.clear")}, command.Decision{}) {
		t.Fatal("expected empty rejection list not to retry")
	}
	if shouldRetryRejectedDecisionWithFreshState(command.Command{Type: command.Type("session.ai_turn.clear")}, command.Reject(command.Rejection{
		Code: "OTHER",
	})) {
		t.Fatal("expected mixed rejection code not to retry")
	}
}

func TestExecute_SavesSnapshotAfterAppend(t *testing.T) {
	cmdRegistry := command.NewRegistry()
	if err := cmdRegistry.Register(command.Definition{Type: command.Type("action.test"), Owner: command.OwnerCore}); err != nil {
		t.Fatalf("register command: %v", err)
	}
	journal := &fakeJournal{}
	snapshots := &trackingSnapshotStore{}
	handler := Handler{
		Commands:  cmdRegistry,
		Decider:   fixedDecider{decision: command.Accept(event.Event{CampaignID: "camp-1", Type: event.Type("action.tested"), Timestamp: time.Unix(0, 0).UTC(), ActorType: event.ActorTypeSystem, PayloadJSON: []byte(`{}`)})},
		Journal:   journal,
		Snapshots: snapshots,
	}

	result, err := handler.Execute(context.Background(), command.Command{CampaignID: "camp-1", Type: command.Type("action.test"), ActorType: command.ActorTypeSystem})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if snapshots.calls != 1 {
		t.Fatalf("snapshot save calls = %d, want 1", snapshots.calls)
	}
	if snapshots.campaignID != "camp-1" {
		t.Fatalf("snapshot campaign id = %q, want camp-1", snapshots.campaignID)
	}
	if snapshots.lastSeq != 1 {
		t.Fatalf("snapshot seq = %d, want 1", snapshots.lastSeq)
	}
	if snapshots.state != result.State {
		t.Fatal("expected saved snapshot state to match execution result state")
	}
}

func TestExecute_DoesNotSaveCheckpointOrSnapshotForUnsequencedEvents(t *testing.T) {
	cmdRegistry := command.NewRegistry()
	if err := cmdRegistry.Register(command.Definition{Type: command.Type("action.test"), Owner: command.OwnerCore}); err != nil {
		t.Fatalf("register command: %v", err)
	}
	checkpoints := &fakeCheckpointStore{}
	snapshots := &trackingSnapshotStore{}
	handler := Handler{
		Commands:    cmdRegistry,
		Decider:     fixedDecider{decision: command.Accept(event.Event{CampaignID: "camp-1", Type: event.Type("action.tested"), Timestamp: time.Unix(0, 0).UTC(), ActorType: event.ActorTypeSystem, PayloadJSON: []byte(`{}`)})},
		Checkpoints: checkpoints,
		Snapshots:   snapshots,
	}

	_, err := handler.Execute(context.Background(), command.Command{CampaignID: "camp-1", Type: command.Type("action.test"), ActorType: command.ActorTypeSystem})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if checkpoints.calls != 0 {
		t.Fatalf("checkpoint calls = %d, want 0", checkpoints.calls)
	}
	if snapshots.calls != 0 {
		t.Fatalf("snapshot calls = %d, want 0", snapshots.calls)
	}
}

func TestExecute_SavesSnapshotBeforeCheckpoint(t *testing.T) {
	cmdRegistry := command.NewRegistry()
	if err := cmdRegistry.Register(command.Definition{Type: command.Type("action.test"), Owner: command.OwnerCore}); err != nil {
		t.Fatalf("register command: %v", err)
	}
	journal := &fakeJournal{}
	order := make([]string, 0, 2)
	handler := Handler{
		Commands:    cmdRegistry,
		Decider:     fixedDecider{decision: command.Accept(event.Event{CampaignID: "camp-1", Type: event.Type("action.tested"), Timestamp: time.Unix(0, 0).UTC(), ActorType: event.ActorTypeSystem, PayloadJSON: []byte(`{}`)})},
		Journal:     journal,
		Checkpoints: &orderedCheckpointStore{order: &order},
		Snapshots:   &orderedSnapshotStore{order: &order},
	}

	_, err := handler.Execute(context.Background(), command.Command{CampaignID: "camp-1", Type: command.Type("action.test"), ActorType: command.ActorTypeSystem})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got := strings.Join(order, ","); got != "snapshot,checkpoint" {
		t.Fatalf("post-persist order = %q, want %q", got, "snapshot,checkpoint")
	}
}

func TestExecute_PropagatesSnapshotSaveError(t *testing.T) {
	cmdRegistry := command.NewRegistry()
	if err := cmdRegistry.Register(command.Definition{Type: command.Type("action.test"), Owner: command.OwnerCore}); err != nil {
		t.Fatalf("register command: %v", err)
	}
	journal := &fakeJournal{}
	snapshotErr := errors.New("snapshot save boom")
	checkpoints := &fakeCheckpointStore{}
	handler := Handler{
		Commands:    cmdRegistry,
		Decider:     fixedDecider{decision: command.Accept(event.Event{CampaignID: "camp-1", Type: event.Type("action.tested"), Timestamp: time.Unix(0, 0).UTC(), ActorType: event.ActorTypeSystem, PayloadJSON: []byte(`{}`)})},
		Journal:     journal,
		Checkpoints: checkpoints,
		Snapshots: &trackingSnapshotStore{
			err: snapshotErr,
		},
	}

	_, err := handler.Execute(context.Background(), command.Command{CampaignID: "camp-1", Type: command.Type("action.test"), ActorType: command.ActorTypeSystem})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrPostPersistSnapshotFailed) {
		t.Fatalf("expected ErrPostPersistSnapshotFailed, got %v", err)
	}
	if !errors.Is(err, snapshotErr) {
		t.Fatalf("expected snapshot error, got %v", err)
	}
	if !IsNonRetryable(err) {
		t.Fatal("expected snapshot save failure to be non-retryable")
	}
	meta, ok := AsPostPersistError(err)
	if !ok {
		t.Fatal("expected post-persist metadata")
	}
	if meta.Stage != PostPersistStageSnapshot {
		t.Fatalf("stage = %q, want %q", meta.Stage, PostPersistStageSnapshot)
	}
	if meta.LastSeq != 1 {
		t.Fatalf("last seq = %d, want 1", meta.LastSeq)
	}
	if checkpoints.calls != 0 {
		t.Fatalf("checkpoint calls = %d, want 0 when snapshot save fails", checkpoints.calls)
	}
}
