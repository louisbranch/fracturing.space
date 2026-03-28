package app

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	playprotocol "github.com/louisbranch/fracturing.space/internal/services/play/protocol"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRealtimeSessionResetTypingTimerUsesInjectedRuntimeTimer(t *testing.T) {
	t.Parallel()

	var (
		created []*fakeRealtimeTimer
		delays  []time.Duration
	)
	runtime := realtimeRuntime{
		typingTTL: 42 * time.Millisecond,
		afterFunc: func(delay time.Duration, callback func()) realtimeTimer {
			timer := &fakeRealtimeTimer{callback: callback}
			created = append(created, timer)
			delays = append(delays, delay)
			return timer
		},
	}
	server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
	hub := newRealtimeHubWithRuntime(hubDepsFromServer(server), runtime)
	server.realtime = hub

	room := &campaignRoom{
		hub:        hub,
		campaignID: "c1",
		ctx:        context.Background(),
		cancel:     func() {},
		sessions:   map[*realtimeSession]struct{}{},
	}

	var buffer syncedFrameBuffer
	session := &realtimeSession{
		userID: "user-1",
		peer:   &wsPeer{encoder: json.NewEncoder(&buffer)},
	}
	session.attach(room, playprotocol.InteractionStateFromGameState(playTestState()))
	room.add(session)

	session.resetTypingTimer(true)
	if len(delays) != 1 || delays[0] != 42*time.Millisecond {
		t.Fatalf("afterFunc delays = %#v", delays)
	}

	session.resetTypingTimer(true)
	if len(created) != 2 {
		t.Fatalf("timers = %d, want %d", len(created), 2)
	}
	if !created[0].stopped {
		t.Fatal("expected previous timer to stop before replacement")
	}

	created[1].Fire()
	frames := drainWSFrames(t, &buffer)
	if len(frames) != 1 || frames[0].Type != "play.typing" {
		t.Fatalf("typing frames = %#v", frames)
	}
	var payload struct {
		Active bool `json:"active"`
	}
	if err := json.Unmarshal(frames[0].Payload, &payload); err != nil {
		t.Fatalf("decode typing frame: %v", err)
	}
	if payload.Active {
		t.Fatal("typing expiry should broadcast inactive state")
	}
}

func TestCampaignRoomProjectionSubscriptionUsesConfiguredRetryDelay(t *testing.T) {
	t.Parallel()

	var delays []time.Duration
	runtime := realtimeRuntime{
		projectionRetryTTL: 250 * time.Millisecond,
		sleepUntilRetry: func(_ context.Context, delay time.Duration) bool {
			delays = append(delays, delay)
			return false
		},
	}

	server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
	events := &failingEventClient{err: errors.New("subscribe failed")}
	server.deps.CampaignUpdates = events
	hub := newRealtimeHubWithRuntime(hubDepsFromServer(server), runtime)
	server.realtime = hub

	room := &campaignRoom{
		hub:        hub,
		campaignID: "c1",
		ctx:        context.Background(),
		cancel:     func() {},
		sessions:   map[*realtimeSession]struct{}{},
		authUserID: "user-1",
	}

	room.runProjectionSubscription()

	if events.calls != 1 {
		t.Fatalf("SubscribeCampaignUpdates calls = %d, want %d", events.calls, 1)
	}
	if len(delays) != 1 || delays[0] != 250*time.Millisecond {
		t.Fatalf("retry delays = %#v", delays)
	}
}

func TestCampaignRoomEnsureProjectionSubscriptionUsesAuthenticatedCursor(t *testing.T) {
	t.Parallel()

	server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
	events := &fakeEventClient{
		stream:      &fakeCampaignUpdateStream{},
		subscribeCh: make(chan struct{}, 1),
	}
	server.deps.CampaignUpdates = events
	hub := newRealtimeHub(server)
	server.realtime = hub

	room := &campaignRoom{
		hub:        hub,
		campaignID: "c1",
		ctx:        context.Background(),
		cancel:     func() {},
		sessions:   map[*realtimeSession]struct{}{},
		authUserID: "user-1",
	}
	room.setLatestGameSequence(9)

	room.ensureProjectionSubscription()
	events.awaitSubscribe(t)
	room.cancel()

	if events.lastUserID != "user-1" {
		t.Fatalf("event auth metadata = %q, want %q", events.lastUserID, "user-1")
	}
	if events.lastRequest == nil {
		t.Fatal("SubscribeCampaignUpdates request = nil")
	}
	if events.lastRequest.GetAfterSeq() != 9 {
		t.Fatalf("SubscribeCampaignUpdates after_seq = %d, want %d", events.lastRequest.GetAfterSeq(), 9)
	}
	if got := events.lastRequest.GetProjectionScopes(); len(got) != 2 || got[0] != "campaign_sessions" || got[1] != "campaign_scenes" {
		t.Fatalf("SubscribeCampaignUpdates projection scopes = %#v", got)
	}
}

type fakeRealtimeTimer struct {
	callback func()
	stopped  bool
}

func (t *fakeRealtimeTimer) Stop() bool {
	t.stopped = true
	return true
}

func (t *fakeRealtimeTimer) Fire() {
	if t.callback != nil {
		t.callback()
	}
}

type failingEventClient struct {
	err   error
	calls int
}

func (f *failingEventClient) SubscribeCampaignUpdates(context.Context, *gamev1.SubscribeCampaignUpdatesRequest, ...gogrpc.CallOption) (gogrpc.ServerStreamingClient[gamev1.CampaignUpdate], error) {
	f.calls++
	return nil, f.err
}

func TestCampaignRoomConsumeProjectionStreamBroadcastsAndTracksSequence(t *testing.T) {
	t.Parallel()

	server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
	participants := &authSensitivePlayParticipantClient{response: enrichedParticipantResponse()}
	characters := &authSensitivePlayCharacterClient{
		listResponse:  enrichedCharacterResponse(),
		sheetResponse: enrichedCharacterSheetResponse(),
	}
	server.deps.Participants = participants
	server.deps.Characters = characters
	hub := newRealtimeHub(server)
	server.realtime = hub

	room := &campaignRoom{
		hub:        hub,
		campaignID: "c1",
		ctx:        context.Background(),
		cancel:     func() {},
		sessions:   map[*realtimeSession]struct{}{},
	}

	var buffer syncedFrameBuffer
	session := &realtimeSession{
		userID: "user-1",
		peer:   &wsPeer{encoder: json.NewEncoder(&buffer)},
	}
	session.attach(room, playprotocol.InteractionStateFromGameState(playTestState()))
	room.add(session)

	stream := &fakeCampaignUpdateStream{
		updates: make(chan *gamev1.CampaignUpdate, 2),
	}
	stream.updates <- nil
	stream.updates <- &gamev1.CampaignUpdate{
		Seq:       12,
		EventType: "projection.applied",
		Update: &gamev1.CampaignUpdate_ProjectionApplied{
			ProjectionApplied: &gamev1.ProjectionApplied{
				Scopes: []string{"campaign_sessions"},
			},
		},
	}
	close(stream.updates)

	runtime := hub.runtime
	runtime.sleepUntilRetry = func(context.Context, time.Duration) bool { return false }
	hub.runtime = runtime

	if ok := room.consumeProjectionStream(stream); !ok {
		t.Fatal("consumeProjectionStream returned false on EOF, want reconnect")
	}
	if got := room.latestGameSequence(); got != 12 {
		t.Fatalf("latestGameSequence() = %d, want 12", got)
	}

	frames := drainWSFrames(t, &buffer)
	if len(frames) != 1 || frames[0].Type != "play.interaction.updated" {
		t.Fatalf("frames = %#v", frames)
	}
	if participants.lastUserID != "user-1" || characters.lastUserID != "user-1" {
		t.Fatalf("auth metadata = participant:%q character:%q, want user-1", participants.lastUserID, characters.lastUserID)
	}
}

func TestCampaignRoomConsumeProjectionStreamRetriesOnRecvError(t *testing.T) {
	t.Parallel()

	var delays []time.Duration
	runtime := realtimeRuntime{
		projectionRetryTTL: 175 * time.Millisecond,
		sleepUntilRetry: func(_ context.Context, delay time.Duration) bool {
			delays = append(delays, delay)
			return false
		},
	}
	server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
	hub := newRealtimeHubWithRuntime(hubDepsFromServer(server), runtime)
	server.realtime = hub

	room := &campaignRoom{
		hub:        hub,
		campaignID: "c1",
		ctx:        context.Background(),
		cancel:     func() {},
		sessions:   map[*realtimeSession]struct{}{},
	}

	if ok := room.consumeProjectionStream(&fakeCampaignUpdateStream{recvErr: errors.New("stream failed")}); ok {
		t.Fatal("consumeProjectionStream returned true, want false")
	}
	if len(delays) != 1 || delays[0] != 175*time.Millisecond {
		t.Fatalf("retry delays = %#v", delays)
	}
}

func TestCampaignRoomBroadcastResyncWritesFrame(t *testing.T) {
	t.Parallel()

	var buffer syncedFrameBuffer
	session := &realtimeSession{
		userID: "user-1",
		peer:   &wsPeer{encoder: json.NewEncoder(&buffer)},
	}
	room := &campaignRoom{campaignID: "c1"}

	room.broadcastResync([]*realtimeSession{session})

	frames := drainWSFrames(t, &buffer)
	if len(frames) != 1 || frames[0].Type != "play.resync" {
		t.Fatalf("frames = %#v", frames)
	}
}

func TestCampaignRoomRunAIDebugSubscriptionUsesConfiguredRetryDelay(t *testing.T) {
	t.Parallel()

	var delays []time.Duration
	runtime := realtimeRuntime{
		projectionRetryTTL: 225 * time.Millisecond,
		sleepUntilRetry: func(_ context.Context, delay time.Duration) bool {
			delays = append(delays, delay)
			return false
		},
	}

	server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
	aiDebug := &fakePlayAIDebugClient{subscribeErr: errors.New("subscribe failed")}
	server.deps.AIDebug = aiDebug
	hub := newRealtimeHubWithRuntime(hubDepsFromServer(server), runtime)
	server.realtime = hub

	room := &campaignRoom{
		hub:        hub,
		campaignID: "c1",
		ctx:        context.Background(),
		cancel:     func() {},
		sessions:   map[*realtimeSession]struct{}{},
		authUserID: "user-1",
	}

	room.runAIDebugSubscription(context.Background(), "s1")

	if aiDebug.subscribeReq == nil {
		t.Fatal("SubscribeCampaignDebugUpdates request = nil")
	}
	if aiDebug.subscribeReq.GetCampaignId() != "c1" || aiDebug.subscribeReq.GetSessionId() != "s1" {
		t.Fatalf("SubscribeCampaignDebugUpdates request = %#v", aiDebug.subscribeReq)
	}
	if len(delays) != 1 || delays[0] != 225*time.Millisecond {
		t.Fatalf("retry delays = %#v", delays)
	}
}

func TestCampaignRoomConsumeAIDebugStreamSkipsInvalidUpdatesAndBroadcastsValidOnes(t *testing.T) {
	t.Parallel()

	server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
	hub := newRealtimeHub(server)
	server.realtime = hub

	room := &campaignRoom{
		hub:              hub,
		campaignID:       "c1",
		ctx:              context.Background(),
		cancel:           func() {},
		sessions:         map[*realtimeSession]struct{}{},
		aiDebugSessionID: "s1",
	}

	var buffer syncedFrameBuffer
	session := &realtimeSession{
		userID: "user-1",
		peer:   &wsPeer{encoder: json.NewEncoder(&buffer)},
	}
	room.add(session)

	stream := &fakeCampaignDebugUpdateStream{
		updates: make(chan *aiv1.CampaignDebugTurnUpdate, 3),
	}
	stream.updates <- nil
	stream.updates <- &aiv1.CampaignDebugTurnUpdate{
		Turn: &aiv1.CampaignDebugTurnSummary{
			Id: "",
		},
	}
	stream.updates <- &aiv1.CampaignDebugTurnUpdate{
		Turn: &aiv1.CampaignDebugTurnSummary{
			Id:         "turn-1",
			CampaignId: "c1",
			SessionId:  "s1",
			Status:     aiv1.CampaignDebugTurnStatus_CAMPAIGN_DEBUG_TURN_STATUS_RUNNING,
			StartedAt:  timestamppb.Now(),
			UpdatedAt:  timestamppb.Now(),
			EntryCount: 1,
		},
		AppendedEntries: []*aiv1.CampaignDebugEntry{{
			Sequence:  1,
			Kind:      aiv1.CampaignDebugEntryKind_CAMPAIGN_DEBUG_ENTRY_KIND_TOOL_CALL,
			ToolName:  "scene_create",
			Payload:   `{"name":"Harbor"}`,
			CreatedAt: timestamppb.Now(),
		}},
	}
	close(stream.updates)

	runtime := hub.runtime
	runtime.sleepUntilRetry = func(context.Context, time.Duration) bool { return false }
	hub.runtime = runtime

	if ok := room.consumeAIDebugStream(context.Background(), "s1", stream); ok {
		t.Fatal("consumeAIDebugStream returned true after EOF retry-disabled, want false")
	}

	frames := drainWSFrames(t, &buffer)
	if len(frames) != 1 || frames[0].Type != "play.ai_debug.turn.updated" {
		t.Fatalf("frames = %#v", frames)
	}
}

func TestCampaignRoomConsumeAIDebugStreamStopsOnSessionSwitch(t *testing.T) {
	t.Parallel()

	server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
	hub := newRealtimeHub(server)
	server.realtime = hub

	room := &campaignRoom{
		hub:              hub,
		campaignID:       "c1",
		ctx:              context.Background(),
		cancel:           func() {},
		sessions:         map[*realtimeSession]struct{}{},
		aiDebugSessionID: "s2",
	}

	var buffer syncedFrameBuffer
	session := &realtimeSession{
		userID: "user-1",
		peer:   &wsPeer{encoder: json.NewEncoder(&buffer)},
	}
	room.add(session)

	stream := &fakeCampaignDebugUpdateStream{
		updates: make(chan *aiv1.CampaignDebugTurnUpdate, 1),
	}
	stream.updates <- &aiv1.CampaignDebugTurnUpdate{
		Turn: &aiv1.CampaignDebugTurnSummary{
			Id:         "turn-1",
			CampaignId: "c1",
			SessionId:  "s1",
			Status:     aiv1.CampaignDebugTurnStatus_CAMPAIGN_DEBUG_TURN_STATUS_RUNNING,
			StartedAt:  timestamppb.Now(),
			UpdatedAt:  timestamppb.Now(),
			EntryCount: 1,
		},
	}

	if ok := room.consumeAIDebugStream(context.Background(), "s1", stream); !ok {
		t.Fatal("consumeAIDebugStream returned false on session switch, want true")
	}
	if frames := drainWSFrames(t, &buffer); len(frames) != 0 {
		t.Fatalf("frames = %#v, want no broadcast", frames)
	}
}

// --- Pure function and structural tests for realtimeRuntime ---

func TestDefaultRealtimeRuntime_HasAllFields(t *testing.T) {
	t.Parallel()
	rt := defaultRealtimeRuntime()
	if rt.now == nil {
		t.Fatal("expected now to be set")
	}
	if rt.afterFunc == nil {
		t.Fatal("expected afterFunc to be set")
	}
	if rt.sleepUntilRetry == nil {
		t.Fatal("expected sleepUntilRetry to be set")
	}
	if rt.typingTTL != defaultTypingTTL {
		t.Fatalf("typingTTL = %v, want %v", rt.typingTTL, defaultTypingTTL)
	}
	if rt.projectionRetryTTL != defaultProjectionRetryTTL {
		t.Fatalf("projectionRetryTTL = %v, want %v", rt.projectionRetryTTL, defaultProjectionRetryTTL)
	}
}

func TestNormalize_FillsMissingFields(t *testing.T) {
	t.Parallel()
	rt := realtimeRuntime{}.normalize()
	if rt.now == nil {
		t.Fatal("normalize should fill now")
	}
	if rt.afterFunc == nil {
		t.Fatal("normalize should fill afterFunc")
	}
	if rt.sleepUntilRetry == nil {
		t.Fatal("normalize should fill sleepUntilRetry")
	}
	if rt.typingTTL != defaultTypingTTL {
		t.Fatalf("typingTTL = %v, want %v", rt.typingTTL, defaultTypingTTL)
	}
	if rt.projectionRetryTTL != defaultProjectionRetryTTL {
		t.Fatalf("projectionRetryTTL = %v, want %v", rt.projectionRetryTTL, defaultProjectionRetryTTL)
	}
}

func TestNormalize_PreservesExistingFields(t *testing.T) {
	t.Parallel()
	fixedNow := func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
	rt := realtimeRuntime{
		now:                fixedNow,
		typingTTL:          5 * time.Second,
		projectionRetryTTL: 10 * time.Second,
	}.normalize()
	if got := rt.now(); !got.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("normalize should preserve existing now, got %v", got)
	}
	if rt.typingTTL != 5*time.Second {
		t.Fatalf("typingTTL = %v, want 5s", rt.typingTTL)
	}
	if rt.projectionRetryTTL != 10*time.Second {
		t.Fatalf("projectionRetryTTL = %v, want 10s", rt.projectionRetryTTL)
	}
}

func TestBackoff_DoublesDelay(t *testing.T) {
	t.Parallel()
	rt := defaultRealtimeRuntime()
	if got := rt.backoff(time.Second); got != 2*time.Second {
		t.Fatalf("backoff(1s) = %v, want 2s", got)
	}
}

func TestBackoff_CapsAtMax(t *testing.T) {
	t.Parallel()
	rt := defaultRealtimeRuntime()
	if got := rt.backoff(20 * time.Second); got != maxProjectionRetryTTL {
		t.Fatalf("backoff(20s) = %v, want max %v", got, maxProjectionRetryTTL)
	}
}

func TestNowTime_ReturnsUTC(t *testing.T) {
	t.Parallel()
	fixed := time.Date(2026, 3, 1, 12, 0, 0, 0, time.FixedZone("EST", -5*3600))
	rt := realtimeRuntime{now: func() time.Time { return fixed }}.normalize()
	got := rt.nowTime()
	if got.Location() != time.UTC {
		t.Fatalf("nowTime location = %v, want UTC", got.Location())
	}
}

func TestSleepUntilRetry_ReturnsOnExpiry(t *testing.T) {
	t.Parallel()
	if !sleepUntilRetry(context.Background(), time.Millisecond) {
		t.Fatal("expected true after timer expiry")
	}
}

func TestSleepUntilRetry_ReturnsFalseOnCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if sleepUntilRetry(ctx, time.Hour) {
		t.Fatal("expected false on cancelled context")
	}
}

func TestStdlibRealtimeTimer_StopNilTimer(t *testing.T) {
	t.Parallel()
	timer := stdlibRealtimeTimer{}
	if timer.Stop() {
		t.Fatal("Stop on nil timer should return false")
	}
}

func TestStdlibRealtimeTimer_StopActiveTimer(t *testing.T) {
	t.Parallel()
	timer := newStdlibRealtimeTimer(time.Hour, func() {})
	if !timer.Stop() {
		t.Fatal("Stop on active timer should return true")
	}
}
