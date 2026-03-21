package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	playprotocol "github.com/louisbranch/fracturing.space/internal/services/play/protocol"
	gogrpc "google.golang.org/grpc"
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

	var buffer bytes.Buffer
	session := &realtimeSession{
		userID: "user-1",
		peer:   &wsPeer{encoder: json.NewEncoder(&buffer)},
	}
	session.attach(room, playprotocol.InteractionStateFromGameState(playTestState()))
	room.add(session)

	session.resetTypingTimer("play.chat.typing", true)
	if len(delays) != 1 || delays[0] != 42*time.Millisecond {
		t.Fatalf("afterFunc delays = %#v", delays)
	}

	session.resetTypingTimer("play.chat.typing", true)
	if len(created) != 2 {
		t.Fatalf("timers = %d, want %d", len(created), 2)
	}
	if !created[0].stopped {
		t.Fatal("expected previous timer to stop before replacement")
	}

	created[1].Fire()
	frames := drainWSFrames(t, &buffer)
	if len(frames) != 1 || frames[0].Type != "play.chat.typing" {
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
	server.events = events
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
	server.events = events
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
