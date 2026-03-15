package countdowntransport

import (
	"context"
	"errors"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testCampaignStore struct {
	record storage.CampaignRecord
	err    error
}

func (s testCampaignStore) Get(context.Context, string) (storage.CampaignRecord, error) {
	if s.err != nil {
		return storage.CampaignRecord{}, s.err
	}
	return s.record, nil
}

type testSessionStore struct {
	record storage.SessionRecord
	err    error
}

func (s testSessionStore) GetSession(context.Context, string, string) (storage.SessionRecord, error) {
	if s.err != nil {
		return storage.SessionRecord{}, s.err
	}
	return s.record, nil
}

type testDaggerheartStore struct {
	countdowns map[string]projectionstore.DaggerheartCountdown
	getErr     error
}

func (s testDaggerheartStore) GetDaggerheartCountdown(context.Context, string, string) (projectionstore.DaggerheartCountdown, error) {
	if s.getErr != nil {
		return projectionstore.DaggerheartCountdown{}, s.getErr
	}
	if len(s.countdowns) == 0 {
		return projectionstore.DaggerheartCountdown{}, storage.ErrNotFound
	}
	for _, countdown := range s.countdowns {
		return countdown, nil
	}
	return projectionstore.DaggerheartCountdown{}, storage.ErrNotFound
}

func newTestHandler(deps Dependencies) *Handler {
	if deps.Campaign == nil {
		deps.Campaign = testCampaignStore{record: storage.CampaignRecord{
			ID:     "camp-1",
			System: systembridge.SystemIDDaggerheart,
			Status: campaign.StatusActive,
		}}
	}
	if deps.Session == nil {
		deps.Session = testSessionStore{record: storage.SessionRecord{
			ID:         "sess-1",
			CampaignID: "camp-1",
			Status:     session.StatusActive,
		}}
	}
	if deps.SessionGate == nil {
		deps.SessionGate = testGateStore{err: storage.ErrNotFound}
	}
	if deps.Daggerheart == nil {
		deps.Daggerheart = testDaggerheartStore{}
	}
	if deps.NewID == nil {
		deps.NewID = func() (string, error) { return "generated-id", nil }
	}
	return NewHandler(deps)
}

func testContext() context.Context {
	ctx := grpcmeta.WithRequestID(context.Background(), "req-1")
	return grpcmeta.WithInvocationID(ctx, "inv-1")
}

func TestHandlerCreateCountdownRejectsNilRequest(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	_, err := handler.CreateCountdown(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerRequireDependencies(t *testing.T) {
	tests := []struct {
		name string
		deps Dependencies
	}{
		{name: "missing campaign", deps: Dependencies{}},
		{name: "missing session", deps: Dependencies{Campaign: testCampaignStore{}}},
		{name: "missing gate", deps: Dependencies{Campaign: testCampaignStore{}, Session: testSessionStore{}}},
		{name: "missing daggerheart", deps: Dependencies{Campaign: testCampaignStore{}, Session: testSessionStore{}, SessionGate: testGateStore{}}},
		{name: "missing id generator", deps: Dependencies{Campaign: testCampaignStore{}, Session: testSessionStore{}, SessionGate: testGateStore{}, Daggerheart: testDaggerheartStore{}}},
		{name: "missing executor", deps: Dependencies{Campaign: testCampaignStore{}, Session: testSessionStore{}, SessionGate: testGateStore{}, Daggerheart: testDaggerheartStore{}, NewID: func() (string, error) { return "id", nil }}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := NewHandler(tt.deps).requireDependencies(); status.Code(err) != codes.Internal {
				t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
			}
		})
	}
}

func TestHandlerCreateCountdownRequiresExecutor(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	_, err := handler.CreateCountdown(testContext(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		Name:       "Clock",
		Kind:       pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Direction:  pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:        4,
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestHandlerCreateCountdownRejectsInvalidMax(t *testing.T) {
	handler := newTestHandler(Dependencies{
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error { return nil },
	})

	_, err := handler.CreateCountdown(testContext(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		Name:       "Clock",
		Kind:       pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Direction:  pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:        0,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerCreateCountdownRejectsCurrentOutOfRange(t *testing.T) {
	handler := newTestHandler(Dependencies{
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error { return nil },
	})

	_, err := handler.CreateCountdown(testContext(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		Name:       "Clock",
		Kind:       pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Direction:  pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:        4,
		Current:    5,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerCreateCountdownRejectsDuplicate(t *testing.T) {
	handler := newTestHandler(Dependencies{
		Daggerheart: testDaggerheartStore{
			countdowns: map[string]projectionstore.DaggerheartCountdown{
				"camp-1:cd-1": {CampaignID: "camp-1", CountdownID: "cd-1"},
			},
		},
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	_, err := handler.CreateCountdown(testContext(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
		Name:        "Clock",
		Kind:        pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Direction:   pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:         4,
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func TestHandlerCreateCountdownPropagatesIDGenerationFailure(t *testing.T) {
	handler := newTestHandler(Dependencies{
		NewID: func() (string, error) { return "", errors.New("boom") },
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	_, err := handler.CreateCountdown(testContext(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		Name:       "Clock",
		Kind:       pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Direction:  pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:        4,
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestHandlerCreateCountdownSuccess(t *testing.T) {
	store := &testDaggerheartStore{
		countdowns: map[string]projectionstore.DaggerheartCountdown{},
	}
	var commandInput DomainCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			commandInput = in
			store.countdowns["camp-1:generated-id"] = projectionstore.DaggerheartCountdown{
				CampaignID:  "camp-1",
				CountdownID: "generated-id",
				Name:        "Clock",
				Kind:        daggerheart.CountdownKindProgress,
				Current:     0,
				Max:         4,
				Direction:   daggerheart.CountdownDirectionIncrease,
			}
			return nil
		},
	})

	resp, err := handler.CreateCountdown(testContext(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		Name:       "Clock",
		Kind:       pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Direction:  pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:        4,
	})
	if err != nil {
		t.Fatalf("CreateCountdown returned error: %v", err)
	}
	if resp.Countdown.CountdownID != "generated-id" {
		t.Fatalf("countdown_id = %q, want generated-id", resp.Countdown.CountdownID)
	}
	if commandInput.CommandType != commandids.DaggerheartCountdownCreate {
		t.Fatalf("command type = %q, want %q", commandInput.CommandType, commandids.DaggerheartCountdownCreate)
	}
	if commandInput.RequestID != "req-1" || commandInput.InvocationID != "inv-1" {
		t.Fatalf("request metadata = (%q,%q), want (req-1,inv-1)", commandInput.RequestID, commandInput.InvocationID)
	}
}

func TestHandlerUpdateCountdownRejectsMissingMutation(t *testing.T) {
	handler := newTestHandler(Dependencies{
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error { return nil },
	})

	_, err := handler.UpdateCountdown(testContext(), &pb.DaggerheartUpdateCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerUpdateCountdownRejectsNilRequest(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	_, err := handler.UpdateCountdown(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerUpdateCountdownSuccess(t *testing.T) {
	store := &testDaggerheartStore{
		countdowns: map[string]projectionstore.DaggerheartCountdown{
			"camp-1:cd-1": {
				CampaignID:  "camp-1",
				CountdownID: "cd-1",
				Name:        "Clock",
				Kind:        daggerheart.CountdownKindProgress,
				Current:     1,
				Max:         4,
				Direction:   daggerheart.CountdownDirectionIncrease,
			},
		},
	}
	var commandInput DomainCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			commandInput = in
			store.countdowns["camp-1:cd-1"] = projectionstore.DaggerheartCountdown{
				CampaignID:  "camp-1",
				CountdownID: "cd-1",
				Name:        "Clock",
				Kind:        daggerheart.CountdownKindProgress,
				Current:     2,
				Max:         4,
				Direction:   daggerheart.CountdownDirectionIncrease,
			}
			return nil
		},
	})

	resp, err := handler.UpdateCountdown(testContext(), &pb.DaggerheartUpdateCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
		Delta:       1,
	})
	if err != nil {
		t.Fatalf("UpdateCountdown returned error: %v", err)
	}
	if commandInput.CommandType != commandids.DaggerheartCountdownUpdate {
		t.Fatalf("command type = %q, want %q", commandInput.CommandType, commandids.DaggerheartCountdownUpdate)
	}
	if resp.After != 2 || resp.Delta != 1 || resp.Before != 1 {
		t.Fatalf("update summary = (%d,%d,%d), want (1,2,1)", resp.Before, resp.After, resp.Delta)
	}
	if resp.Countdown.Current != 2 {
		t.Fatalf("countdown current = %d, want 2", resp.Countdown.Current)
	}
}

func TestHandlerUpdateCountdownWrapsCountdownStoreErrors(t *testing.T) {
	handler := newTestHandler(Dependencies{
		Daggerheart: testDaggerheartStore{getErr: errors.New("boom")},
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	_, err := handler.UpdateCountdown(testContext(), &pb.DaggerheartUpdateCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
		Delta:       1,
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestHandlerDeleteCountdownRequiresExecutor(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	_, err := handler.DeleteCountdown(testContext(), &pb.DaggerheartDeleteCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestHandlerDeleteCountdownRejectsNilRequest(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	_, err := handler.DeleteCountdown(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerDeleteCountdownSuccess(t *testing.T) {
	store := &testDaggerheartStore{
		countdowns: map[string]projectionstore.DaggerheartCountdown{
			"camp-1:cd-1": {
				CampaignID:  "camp-1",
				CountdownID: "cd-1",
			},
		},
	}
	var commandInput DomainCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			commandInput = in
			delete(store.countdowns, "camp-1:cd-1")
			return nil
		},
	})

	resp, err := handler.DeleteCountdown(testContext(), &pb.DaggerheartDeleteCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
	})
	if err != nil {
		t.Fatalf("DeleteCountdown returned error: %v", err)
	}
	if commandInput.CommandType != commandids.DaggerheartCountdownDelete {
		t.Fatalf("command type = %q, want %q", commandInput.CommandType, commandids.DaggerheartCountdownDelete)
	}
	if resp.CountdownID != "cd-1" {
		t.Fatalf("countdown_id = %q, want cd-1", resp.CountdownID)
	}
}

func TestHandlerDeleteCountdownRejectsMissingCountdown(t *testing.T) {
	handler := newTestHandler(Dependencies{
		Daggerheart: testDaggerheartStore{},
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	_, err := handler.DeleteCountdown(testContext(), &pb.DaggerheartDeleteCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.NotFound)
	}
}

func TestHandlerUpdateCountdownRejectsInactiveSession(t *testing.T) {
	handler := newTestHandler(Dependencies{
		Session: testSessionStore{record: storage.SessionRecord{
			ID:         "sess-1",
			CampaignID: "camp-1",
			Status:     session.StatusEnded,
		}},
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	_, err := handler.UpdateCountdown(testContext(), &pb.DaggerheartUpdateCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
		Delta:       1,
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func TestHandlerDeleteCountdownRejectsOpenSessionGate(t *testing.T) {
	handler := newTestHandler(Dependencies{
		SessionGate: testGateStore{gate: storage.SessionGate{GateID: "gate-1"}},
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	_, err := handler.DeleteCountdown(testContext(), &pb.DaggerheartDeleteCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func TestHandlerUpdateCountdownRejectsUnsupportedSystem(t *testing.T) {
	handler := newTestHandler(Dependencies{
		Campaign: testCampaignStore{record: storage.CampaignRecord{
			ID:     "camp-1",
			System: systembridge.SystemIDUnspecified,
			Status: campaign.StatusActive,
		}},
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	_, err := handler.UpdateCountdown(testContext(), &pb.DaggerheartUpdateCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
		Delta:       1,
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}
