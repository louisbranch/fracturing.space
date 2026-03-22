package ai

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/debugtrace"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeCampaignDebugTraceStore struct {
	page    debugtrace.Page
	turns   map[string]debugtrace.Turn
	entries map[string][]debugtrace.Entry
	listErr error
	getErr  error
}

func (f *fakeCampaignDebugTraceStore) PutCampaignDebugTurn(context.Context, debugtrace.Turn) error {
	return nil
}

func (f *fakeCampaignDebugTraceStore) PutCampaignDebugTurnEntry(context.Context, debugtrace.Entry) error {
	return nil
}

func (f *fakeCampaignDebugTraceStore) ListCampaignDebugTurns(context.Context, string, string, int, string) (debugtrace.Page, error) {
	if f.listErr != nil {
		return debugtrace.Page{}, f.listErr
	}
	return f.page, nil
}

func (f *fakeCampaignDebugTraceStore) GetCampaignDebugTurn(_ context.Context, _ string, turnID string) (debugtrace.Turn, error) {
	if f.getErr != nil {
		return debugtrace.Turn{}, f.getErr
	}
	turn, ok := f.turns[turnID]
	if !ok {
		return debugtrace.Turn{}, storage.ErrNotFound
	}
	return turn, nil
}

func (f *fakeCampaignDebugTraceStore) ListCampaignDebugTurnEntries(_ context.Context, turnID string) ([]debugtrace.Entry, error) {
	return append([]debugtrace.Entry(nil), f.entries[turnID]...), nil
}

func TestCampaignDebugHandlersListAndGet(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC)
	completedAt := startedAt.Add(3 * time.Second)
	store := &fakeCampaignDebugTraceStore{
		page: debugtrace.Page{
			Turns: []debugtrace.Turn{{
				ID:          "turn-1",
				CampaignID:  "campaign-1",
				SessionID:   "session-1",
				Provider:    provider.OpenAI,
				Model:       "gpt-4.1-mini",
				Status:      debugtrace.StatusSucceeded,
				Usage:       provider.Usage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15},
				StartedAt:   startedAt,
				UpdatedAt:   completedAt,
				CompletedAt: &completedAt,
				EntryCount:  2,
			}},
			NextPageToken: "next-1",
		},
		turns: map[string]debugtrace.Turn{
			"turn-1": {
				ID:          "turn-1",
				CampaignID:  "campaign-1",
				SessionID:   "session-1",
				Provider:    provider.OpenAI,
				Model:       "gpt-4.1-mini",
				Status:      debugtrace.StatusSucceeded,
				Usage:       provider.Usage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15},
				StartedAt:   startedAt,
				UpdatedAt:   completedAt,
				CompletedAt: &completedAt,
				EntryCount:  2,
			},
		},
		entries: map[string][]debugtrace.Entry{
			"turn-1": {
				{
					TurnID:    "turn-1",
					Sequence:  1,
					Kind:      debugtrace.EntryKindToolCall,
					ToolName:  "scene_create",
					Payload:   `{"name":"Arrival"}`,
					CreatedAt: startedAt,
				},
				{
					TurnID:     "turn-1",
					Sequence:   2,
					Kind:       debugtrace.EntryKindModelResponse,
					Payload:    "The fog opens over the harbor.",
					ResponseID: "resp-1",
					CreatedAt:  completedAt,
				},
			},
		},
	}
	debugService, err := service.NewCampaignDebugService(service.CampaignDebugServiceConfig{
		DebugTraceStore: store,
	})
	if err != nil {
		t.Fatalf("NewCampaignDebugService: %v", err)
	}
	authz := &fakeGameAuthorizationClient{canResp: &gamev1.CanResponse{Allowed: true}}
	handlers, err := NewCampaignDebugHandlers(CampaignDebugHandlersConfig{
		CampaignDebugService: debugService,
		AuthorizationClient:  authz,
	})
	if err != nil {
		t.Fatalf("NewCampaignDebugHandlers: %v", err)
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	listResp, err := handlers.ListCampaignDebugTurns(ctx, &aiv1.ListCampaignDebugTurnsRequest{
		CampaignId: "campaign-1",
		SessionId:  "session-1",
	})
	if err != nil {
		t.Fatalf("ListCampaignDebugTurns: %v", err)
	}
	if len(listResp.GetTurns()) != 1 {
		t.Fatalf("turn count = %d, want 1", len(listResp.GetTurns()))
	}
	if got := listResp.GetTurns()[0].GetUsage().GetTotalTokens(); got != 15 {
		t.Fatalf("list usage total = %d, want 15", got)
	}
	if authz.lastReq == nil || authz.lastReq.GetAction() != gamev1.AuthorizationAction_AUTHORIZATION_ACTION_READ {
		t.Fatalf("authorization request = %#v", authz.lastReq)
	}

	getResp, err := handlers.GetCampaignDebugTurn(ctx, &aiv1.GetCampaignDebugTurnRequest{
		CampaignId: "campaign-1",
		TurnId:     "turn-1",
	})
	if err != nil {
		t.Fatalf("GetCampaignDebugTurn: %v", err)
	}
	if len(getResp.GetTurn().GetEntries()) != 2 {
		t.Fatalf("entry count = %d, want 2", len(getResp.GetTurn().GetEntries()))
	}
	if got := getResp.GetTurn().GetEntries()[0].GetToolName(); got != "scene_create" {
		t.Fatalf("tool_name = %q, want %q", got, "scene_create")
	}
}

func TestCampaignDebugHandlersSubscribe(t *testing.T) {
	t.Parallel()

	broker := service.NewCampaignDebugUpdateBroker()
	debugService, err := service.NewCampaignDebugService(service.CampaignDebugServiceConfig{
		DebugTraceStore: &fakeCampaignDebugTraceStore{},
		UpdateBroker:    broker,
	})
	if err != nil {
		t.Fatalf("NewCampaignDebugService: %v", err)
	}
	authz := &fakeGameAuthorizationClient{canResp: &gamev1.CanResponse{Allowed: true}}
	handlers, err := NewCampaignDebugHandlers(CampaignDebugHandlersConfig{
		CampaignDebugService: debugService,
		AuthorizationClient:  authz,
	})
	if err != nil {
		t.Fatalf("NewCampaignDebugHandlers: %v", err)
	}

	ctx, cancel := context.WithCancel(metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1")))
	defer cancel()
	stream := &campaignDebugSubscribeStream{ctx: ctx}

	errCh := make(chan error, 1)
	go func() {
		errCh <- handlers.SubscribeCampaignDebugUpdates(&aiv1.SubscribeCampaignDebugUpdatesRequest{
			CampaignId: "campaign-1",
			SessionId:  "session-1",
		}, stream)
	}()
	time.Sleep(10 * time.Millisecond)

	broker.Publish("campaign-1", "session-1", service.CampaignDebugTurnUpdate{
		Turn: debugtrace.Turn{
			ID:         "turn-1",
			CampaignID: "campaign-1",
			SessionID:  "session-1",
			Status:     debugtrace.StatusRunning,
			StartedAt:  time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC),
			UpdatedAt:  time.Date(2026, 3, 22, 12, 0, 1, 0, time.UTC),
			EntryCount: 1,
		},
		AppendedEntries: []debugtrace.Entry{{
			TurnID:    "turn-1",
			Sequence:  1,
			Kind:      debugtrace.EntryKindToolCall,
			ToolName:  "scene_create",
			Payload:   `{"name":"Harbor"}`,
			CreatedAt: time.Date(2026, 3, 22, 12, 0, 1, 0, time.UTC),
		}},
	})

	stream.awaitSend(t)
	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("SubscribeCampaignDebugUpdates: %v", err)
	}
	if len(stream.sent) != 1 {
		t.Fatalf("sent updates = %d, want 1", len(stream.sent))
	}
	if got := stream.sent[0].GetTurn().GetId(); got != "turn-1" {
		t.Fatalf("turn id = %q, want %q", got, "turn-1")
	}
	if got := stream.sent[0].GetAppendedEntries()[0].GetToolName(); got != "scene_create" {
		t.Fatalf("tool name = %q, want %q", got, "scene_create")
	}
}

func TestCampaignDebugHandlersValidationAndErrorMapping(t *testing.T) {
	t.Parallel()

	if _, err := NewCampaignDebugHandlers(CampaignDebugHandlersConfig{}); err == nil {
		t.Fatal("NewCampaignDebugHandlers error = nil, want validation failure")
	}

	store := &fakeCampaignDebugTraceStore{}
	debugService, err := service.NewCampaignDebugService(service.CampaignDebugServiceConfig{
		DebugTraceStore: store,
		UpdateBroker:    service.NewCampaignDebugUpdateBroker(),
	})
	if err != nil {
		t.Fatalf("NewCampaignDebugService: %v", err)
	}
	authz := &fakeGameAuthorizationClient{canResp: &gamev1.CanResponse{Allowed: true}}
	handlers, err := NewCampaignDebugHandlers(CampaignDebugHandlersConfig{
		CampaignDebugService: debugService,
		AuthorizationClient:  authz,
	})
	if err != nil {
		t.Fatalf("NewCampaignDebugHandlers: %v", err)
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(userIDHeader, "user-1"))

	if _, err := handlers.ListCampaignDebugTurns(ctx, nil); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("ListCampaignDebugTurns(nil) code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
	if _, err := handlers.GetCampaignDebugTurn(ctx, nil); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("GetCampaignDebugTurn(nil) code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
	if err := handlers.SubscribeCampaignDebugUpdates(nil, &campaignDebugSubscribeStream{ctx: ctx}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("SubscribeCampaignDebugUpdates(nil) code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}

	if _, err := handlers.ListCampaignDebugTurns(ctx, &aiv1.ListCampaignDebugTurnsRequest{
		CampaignId: "campaign-1",
	}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("ListCampaignDebugTurns invalid code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}

	store.listErr = errors.New("boom")
	if _, err := handlers.ListCampaignDebugTurns(ctx, &aiv1.ListCampaignDebugTurnsRequest{
		CampaignId: "campaign-1",
		SessionId:  "session-1",
	}); status.Code(err) != codes.Internal {
		t.Fatalf("ListCampaignDebugTurns internal code = %v, want %v", status.Code(err), codes.Internal)
	}
	store.listErr = nil

	if _, err := handlers.GetCampaignDebugTurn(ctx, &aiv1.GetCampaignDebugTurnRequest{
		CampaignId: "campaign-1",
		TurnId:     "missing",
	}); status.Code(err) != codes.NotFound {
		t.Fatalf("GetCampaignDebugTurn missing code = %v, want %v", status.Code(err), codes.NotFound)
	}

	store.getErr = errors.New("boom")
	if _, err := handlers.GetCampaignDebugTurn(ctx, &aiv1.GetCampaignDebugTurnRequest{
		CampaignId: "campaign-1",
		TurnId:     "turn-1",
	}); status.Code(err) != codes.Internal {
		t.Fatalf("GetCampaignDebugTurn internal code = %v, want %v", status.Code(err), codes.Internal)
	}
	store.getErr = nil

	sendErrStream := &campaignDebugSubscribeStream{
		ctx:     ctx,
		sendErr: status.Error(codes.Unavailable, "send failed"),
	}
	broker := service.NewCampaignDebugUpdateBroker()
	subscribeService, err := service.NewCampaignDebugService(service.CampaignDebugServiceConfig{
		DebugTraceStore: store,
		UpdateBroker:    broker,
	})
	if err != nil {
		t.Fatalf("NewCampaignDebugService subscribe: %v", err)
	}
	handlers.svc = subscribeService

	errCh := make(chan error, 1)
	go func() {
		errCh <- handlers.SubscribeCampaignDebugUpdates(&aiv1.SubscribeCampaignDebugUpdatesRequest{
			CampaignId: "campaign-1",
			SessionId:  "session-1",
		}, sendErrStream)
	}()
	time.Sleep(10 * time.Millisecond)
	broker.Publish("campaign-1", "session-1", service.CampaignDebugTurnUpdate{
		Turn: debugtrace.Turn{
			ID:         "turn-1",
			CampaignID: "campaign-1",
			SessionID:  "session-1",
			Status:     debugtrace.StatusRunning,
			StartedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		},
	})

	if err := <-errCh; status.Code(err) != codes.Unavailable {
		t.Fatalf("SubscribeCampaignDebugUpdates send error code = %v, want %v", status.Code(err), codes.Unavailable)
	}
}

type campaignDebugSubscribeStream struct {
	ctx context.Context

	mu      sync.Mutex
	sent    []*aiv1.CampaignDebugTurnUpdate
	sendCh  chan struct{}
	sendErr error
}

func (s *campaignDebugSubscribeStream) Send(update *aiv1.CampaignDebugTurnUpdate) error {
	s.mu.Lock()
	s.sent = append(s.sent, update)
	if s.sendCh == nil {
		s.sendCh = make(chan struct{}, 1)
	}
	sendCh := s.sendCh
	s.mu.Unlock()
	select {
	case sendCh <- struct{}{}:
	default:
	}
	return s.sendErr
}

func (s *campaignDebugSubscribeStream) awaitSend(t *testing.T) {
	t.Helper()
	s.mu.Lock()
	if s.sendCh == nil {
		s.sendCh = make(chan struct{}, 1)
	}
	sendCh := s.sendCh
	s.mu.Unlock()
	select {
	case <-sendCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Send")
	}
}

func (s *campaignDebugSubscribeStream) SetHeader(metadata.MD) error  { return nil }
func (s *campaignDebugSubscribeStream) SendHeader(metadata.MD) error { return nil }
func (s *campaignDebugSubscribeStream) SetTrailer(metadata.MD)       {}
func (s *campaignDebugSubscribeStream) Context() context.Context     { return s.ctx }
func (s *campaignDebugSubscribeStream) SendMsg(any) error            { return nil }
func (s *campaignDebugSubscribeStream) RecvMsg(any) error            { return nil }
