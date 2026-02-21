package web

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	webstorage "github.com/louisbranch/fracturing.space/internal/services/web/storage"
	"google.golang.org/grpc"
)

type fakeEventHeadClient struct {
	headByCampaign   map[string]uint64
	eventsByCampaign map[string][]*statev1.Event
	listRequests     []*statev1.ListEventsRequest
	listErr          error
	listErrByOrder   map[string]error
}

func (f *fakeEventHeadClient) AppendEvent(context.Context, *statev1.AppendEventRequest, ...grpc.CallOption) (*statev1.AppendEventResponse, error) {
	return nil, nil
}

func (f *fakeEventHeadClient) ListEvents(_ context.Context, req *statev1.ListEventsRequest, _ ...grpc.CallOption) (*statev1.ListEventsResponse, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listErrByOrder != nil {
		if err := f.listErrByOrder[strings.TrimSpace(req.GetOrderBy())]; err != nil {
			return nil, err
		}
	}
	f.listRequests = append(f.listRequests, req)

	campaignID := strings.TrimSpace(req.GetCampaignId())
	if strings.TrimSpace(req.GetOrderBy()) == "seq desc" && req.GetPageSize() == 1 && strings.TrimSpace(req.GetPageToken()) == "" {
		head := f.headByCampaign[campaignID]
		if head == 0 {
			return &statev1.ListEventsResponse{}, nil
		}
		return &statev1.ListEventsResponse{
			Events: []*statev1.Event{
				{CampaignId: campaignID, Seq: head},
			},
		}, nil
	}

	if events, ok := f.eventsByCampaign[campaignID]; ok {
		afterSeq := req.GetAfterSeq()
		filtered := make([]*statev1.Event, 0, len(events))
		for _, evt := range events {
			if evt == nil || evt.GetSeq() <= afterSeq {
				continue
			}
			filtered = append(filtered, evt)
		}

		start := 0
		if token := strings.TrimSpace(req.GetPageToken()); token != "" {
			parsed, err := strconv.Atoi(token)
			if err == nil && parsed >= 0 && parsed <= len(filtered) {
				start = parsed
			}
		}

		pageSize := int(req.GetPageSize())
		if pageSize <= 0 {
			pageSize = 50
		}
		end := start + pageSize
		if end > len(filtered) {
			end = len(filtered)
		}
		nextToken := ""
		if end < len(filtered) {
			nextToken = strconv.Itoa(end)
		}

		return &statev1.ListEventsResponse{
			Events:        filtered[start:end],
			NextPageToken: nextToken,
		}, nil
	}
	return &statev1.ListEventsResponse{}, nil
}

func (f *fakeEventHeadClient) ListTimelineEntries(context.Context, *statev1.ListTimelineEntriesRequest, ...grpc.CallOption) (*statev1.ListTimelineEntriesResponse, error) {
	return nil, nil
}

func (f *fakeEventHeadClient) SubscribeCampaignUpdates(context.Context, *statev1.SubscribeCampaignUpdatesRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[statev1.CampaignUpdate], error) {
	return nil, errors.New("not implemented")
}

type fakeCampaignScopeStaleMark struct {
	campaignID string
	scope      string
	headSeq    uint64
}

type fakeInvalidationCacheStore struct {
	trackedCampaignIDs []string
	cursors            map[string]webstorage.CampaignEventCursor
	staleMarks         []fakeCampaignScopeStaleMark
	listErr            error
	getCursorErr       error
	putCursorErr       error
	markErr            error
}

func (f *fakeInvalidationCacheStore) Close() error { return nil }

func (f *fakeInvalidationCacheStore) GetCacheEntry(context.Context, string) (webstorage.CacheEntry, bool, error) {
	return webstorage.CacheEntry{}, false, nil
}

func (f *fakeInvalidationCacheStore) PutCacheEntry(context.Context, webstorage.CacheEntry) error {
	return nil
}

func (f *fakeInvalidationCacheStore) DeleteCacheEntry(context.Context, string) error {
	return nil
}

func (f *fakeInvalidationCacheStore) ListTrackedCampaignIDs(context.Context) ([]string, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return append([]string(nil), f.trackedCampaignIDs...), nil
}

func (f *fakeInvalidationCacheStore) GetCampaignEventCursor(_ context.Context, campaignID string) (webstorage.CampaignEventCursor, bool, error) {
	if f.getCursorErr != nil {
		return webstorage.CampaignEventCursor{}, false, f.getCursorErr
	}
	if f.cursors == nil {
		return webstorage.CampaignEventCursor{}, false, nil
	}
	cursor, ok := f.cursors[campaignID]
	return cursor, ok, nil
}

func (f *fakeInvalidationCacheStore) PutCampaignEventCursor(_ context.Context, cursor webstorage.CampaignEventCursor) error {
	if f.putCursorErr != nil {
		return f.putCursorErr
	}
	if f.cursors == nil {
		f.cursors = make(map[string]webstorage.CampaignEventCursor)
	}
	f.cursors[cursor.CampaignID] = cursor
	return nil
}

func (f *fakeInvalidationCacheStore) MarkCampaignScopeStale(_ context.Context, campaignID, scope string, headSeq uint64, _ time.Time) error {
	if f.markErr != nil {
		return f.markErr
	}
	f.staleMarks = append(f.staleMarks, fakeCampaignScopeStaleMark{
		campaignID: campaignID,
		scope:      scope,
		headSeq:    headSeq,
	})
	return nil
}

func TestSyncCampaignEventHeadsMarksCampaignSummaryStaleWhenHeadAdvances(t *testing.T) {
	cacheStore := &fakeInvalidationCacheStore{
		trackedCampaignIDs: []string{"camp-1"},
		cursors: map[string]webstorage.CampaignEventCursor{
			"camp-1": {CampaignID: "camp-1", LatestSeq: 5},
		},
	}
	eventClient := &fakeEventHeadClient{
		headByCampaign: map[string]uint64{"camp-1": 6},
	}
	h := &handler{
		cacheStore:  cacheStore,
		eventClient: eventClient,
	}

	if err := h.syncCampaignEventHeads(context.Background()); err != nil {
		t.Fatalf("sync campaign event heads: %v", err)
	}

	expectedScopes := map[string]bool{
		cacheScopeCampaignSummary:      true,
		cacheScopeCampaignParticipants: true,
		cacheScopeCampaignSessions:     true,
		cacheScopeCampaignCharacters:   true,
		cacheScopeCampaignInvites:      true,
	}
	if len(cacheStore.staleMarks) != len(expectedScopes) {
		t.Fatalf("stale marks = %d, want %d", len(cacheStore.staleMarks), len(expectedScopes))
	}
	for _, staleMark := range cacheStore.staleMarks {
		if staleMark.campaignID != "camp-1" {
			t.Fatalf("campaign id = %q, want %q", staleMark.campaignID, "camp-1")
		}
		if staleMark.headSeq != 6 {
			t.Fatalf("head seq = %d, want %d", staleMark.headSeq, 6)
		}
		if !expectedScopes[staleMark.scope] {
			t.Fatalf("unexpected stale scope %q", staleMark.scope)
		}
	}

	cursor := cacheStore.cursors["camp-1"]
	if cursor.LatestSeq != 6 {
		t.Fatalf("cursor seq = %d, want %d", cursor.LatestSeq, 6)
	}

	if len(eventClient.listRequests) != 2 {
		t.Fatalf("list requests = %d, want %d", len(eventClient.listRequests), 2)
	}

	headRequest := eventClient.listRequests[0]
	if headRequest.GetCampaignId() != "camp-1" {
		t.Fatalf("campaign id = %q, want %q", headRequest.GetCampaignId(), "camp-1")
	}
	if headRequest.GetPageSize() != 1 {
		t.Fatalf("head page size = %d, want %d", headRequest.GetPageSize(), 1)
	}
	if headRequest.GetOrderBy() != "seq desc" {
		t.Fatalf("head order by = %q, want %q", headRequest.GetOrderBy(), "seq desc")
	}

	deltaRequest := eventClient.listRequests[1]
	if deltaRequest.GetCampaignId() != "camp-1" {
		t.Fatalf("delta campaign id = %q, want %q", deltaRequest.GetCampaignId(), "camp-1")
	}
	if deltaRequest.GetAfterSeq() != 5 {
		t.Fatalf("delta after seq = %d, want %d", deltaRequest.GetAfterSeq(), 5)
	}
	if deltaRequest.GetOrderBy() != "seq" {
		t.Fatalf("delta order by = %q, want %q", deltaRequest.GetOrderBy(), "seq")
	}
}

func TestSyncCampaignEventHeadsMarksScopesFromDeltaEvents(t *testing.T) {
	cacheStore := &fakeInvalidationCacheStore{
		trackedCampaignIDs: []string{"camp-1"},
		cursors: map[string]webstorage.CampaignEventCursor{
			"camp-1": {CampaignID: "camp-1", LatestSeq: 5},
		},
	}
	eventClient := &fakeEventHeadClient{
		headByCampaign: map[string]uint64{"camp-1": 7},
		eventsByCampaign: map[string][]*statev1.Event{
			"camp-1": {
				{CampaignId: "camp-1", Seq: 6, Type: "session.started"},
				{CampaignId: "camp-1", Seq: 7, Type: "invite.created"},
			},
		},
	}
	h := &handler{
		cacheStore:  cacheStore,
		eventClient: eventClient,
	}

	if err := h.syncCampaignEventHeads(context.Background()); err != nil {
		t.Fatalf("sync campaign event heads: %v", err)
	}

	expectedScopes := map[string]bool{
		cacheScopeCampaignSessions: true,
		cacheScopeCampaignInvites:  true,
	}
	if len(cacheStore.staleMarks) != len(expectedScopes) {
		t.Fatalf("stale marks = %d, want %d", len(cacheStore.staleMarks), len(expectedScopes))
	}
	for _, staleMark := range cacheStore.staleMarks {
		if staleMark.campaignID != "camp-1" {
			t.Fatalf("campaign id = %q, want %q", staleMark.campaignID, "camp-1")
		}
		if staleMark.headSeq != 7 {
			t.Fatalf("head seq = %d, want %d", staleMark.headSeq, 7)
		}
		if !expectedScopes[staleMark.scope] {
			t.Fatalf("unexpected stale scope %q", staleMark.scope)
		}
	}

	cursor := cacheStore.cursors["camp-1"]
	if cursor.LatestSeq != 7 {
		t.Fatalf("cursor seq = %d, want %d", cursor.LatestSeq, 7)
	}
}

func TestSyncCampaignEventHeadsSkipsStaleMarkWhenHeadUnchanged(t *testing.T) {
	cacheStore := &fakeInvalidationCacheStore{
		trackedCampaignIDs: []string{"camp-1"},
		cursors: map[string]webstorage.CampaignEventCursor{
			"camp-1": {CampaignID: "camp-1", LatestSeq: 6},
		},
	}
	eventClient := &fakeEventHeadClient{
		headByCampaign: map[string]uint64{"camp-1": 6},
	}
	h := &handler{
		cacheStore:  cacheStore,
		eventClient: eventClient,
	}

	if err := h.syncCampaignEventHeads(context.Background()); err != nil {
		t.Fatalf("sync campaign event heads: %v", err)
	}

	if len(cacheStore.staleMarks) != 0 {
		t.Fatalf("stale marks = %d, want %d", len(cacheStore.staleMarks), 0)
	}
	cursor := cacheStore.cursors["camp-1"]
	if cursor.LatestSeq != 6 {
		t.Fatalf("cursor seq = %d, want %d", cursor.LatestSeq, 6)
	}
}

func TestSyncCampaignEventHeadsTracksNewCampaignCursorWithoutStaleMark(t *testing.T) {
	cacheStore := &fakeInvalidationCacheStore{
		trackedCampaignIDs: []string{"camp-1"},
	}
	eventClient := &fakeEventHeadClient{
		headByCampaign: map[string]uint64{"camp-1": 3},
	}
	h := &handler{
		cacheStore:  cacheStore,
		eventClient: eventClient,
	}

	if err := h.syncCampaignEventHeads(context.Background()); err != nil {
		t.Fatalf("sync campaign event heads: %v", err)
	}

	if len(cacheStore.staleMarks) != 0 {
		t.Fatalf("stale marks = %d, want %d", len(cacheStore.staleMarks), 0)
	}
	cursor := cacheStore.cursors["camp-1"]
	if cursor.LatestSeq != 3 {
		t.Fatalf("cursor seq = %d, want %d", cursor.LatestSeq, 3)
	}
}

func TestSyncCampaignEventHeadsReturnsStoreError(t *testing.T) {
	h := &handler{
		cacheStore: &fakeInvalidationCacheStore{
			listErr: errors.New("boom"),
		},
		eventClient: &fakeEventHeadClient{},
	}

	if err := h.syncCampaignEventHeads(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
}

func TestSyncCampaignEventHeadsReturnsEventHeadError(t *testing.T) {
	h := &handler{
		cacheStore: &fakeInvalidationCacheStore{
			trackedCampaignIDs: []string{"camp-1"},
		},
		eventClient: &fakeEventHeadClient{listErr: errors.New("event service unavailable")},
	}

	err := h.syncCampaignEventHeads(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "read campaign head") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSyncCampaignEventHeadsReturnsCursorReadError(t *testing.T) {
	h := &handler{
		cacheStore: &fakeInvalidationCacheStore{
			trackedCampaignIDs: []string{"camp-1"},
			getCursorErr:       errors.New("cursor read failed"),
		},
		eventClient: &fakeEventHeadClient{
			headByCampaign: map[string]uint64{"camp-1": 1},
		},
	}

	err := h.syncCampaignEventHeads(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "read campaign cursor") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSyncCampaignEventHeadsReturnsStaleMarkError(t *testing.T) {
	h := &handler{
		cacheStore: &fakeInvalidationCacheStore{
			trackedCampaignIDs: []string{"camp-1"},
			cursors: map[string]webstorage.CampaignEventCursor{
				"camp-1": {CampaignID: "camp-1", LatestSeq: 1},
			},
			markErr: errors.New("stale mark failed"),
		},
		eventClient: &fakeEventHeadClient{
			headByCampaign: map[string]uint64{"camp-1": 2},
		},
	}

	err := h.syncCampaignEventHeads(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "mark stale campaign scope") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSyncCampaignEventHeadsReturnsCursorPersistError(t *testing.T) {
	h := &handler{
		cacheStore: &fakeInvalidationCacheStore{
			trackedCampaignIDs: []string{"camp-1"},
			putCursorErr:       errors.New("cursor persist failed"),
		},
		eventClient: &fakeEventHeadClient{
			headByCampaign: map[string]uint64{"camp-1": 1},
		},
	}

	err := h.syncCampaignEventHeads(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "persist campaign cursor") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSyncCampaignEventHeadsNoopWhenDependenciesMissing(t *testing.T) {
	var nilHandler *handler
	if err := nilHandler.syncCampaignEventHeads(context.Background()); err != nil {
		t.Fatalf("nil handler sync should not fail: %v", err)
	}

	if err := (&handler{}).syncCampaignEventHeads(context.Background()); err != nil {
		t.Fatalf("handler without dependencies should not fail: %v", err)
	}

	if err := (&handler{cacheStore: &fakeInvalidationCacheStore{}}).syncCampaignEventHeads(context.Background()); err != nil {
		t.Fatalf("handler without event client should not fail: %v", err)
	}
}

func TestSyncCampaignEventHeadsAcceptsNilContext(t *testing.T) {
	cacheStore := &fakeInvalidationCacheStore{
		trackedCampaignIDs: []string{"camp-1"},
	}
	h := &handler{
		cacheStore:  cacheStore,
		eventClient: &fakeEventHeadClient{headByCampaign: map[string]uint64{"camp-1": 2}},
	}

	if err := h.syncCampaignEventHeads(nil); err != nil {
		t.Fatalf("sync campaign event heads with nil context: %v", err)
	}
	if got := cacheStore.cursors["camp-1"].LatestSeq; got != 2 {
		t.Fatalf("cursor seq = %d, want %d", got, 2)
	}
}

func TestSyncCampaignEventHeadsSkipsBlankCampaignID(t *testing.T) {
	eventClient := &fakeEventHeadClient{headByCampaign: map[string]uint64{"camp-1": 1}}
	cacheStore := &fakeInvalidationCacheStore{
		trackedCampaignIDs: []string{" ", "camp-1", ""},
	}
	h := &handler{
		cacheStore:  cacheStore,
		eventClient: eventClient,
	}

	if err := h.syncCampaignEventHeads(context.Background()); err != nil {
		t.Fatalf("sync campaign event heads: %v", err)
	}
	if len(eventClient.listRequests) != 1 {
		t.Fatalf("list requests = %d, want %d", len(eventClient.listRequests), 1)
	}
	if eventClient.listRequests[0].GetCampaignId() != "camp-1" {
		t.Fatalf("campaign id = %q, want %q", eventClient.listRequests[0].GetCampaignId(), "camp-1")
	}
}

func TestSyncCampaignEventHeadsReturnsDeltaScopeListError(t *testing.T) {
	h := &handler{
		cacheStore: &fakeInvalidationCacheStore{
			trackedCampaignIDs: []string{"camp-1"},
			cursors: map[string]webstorage.CampaignEventCursor{
				"camp-1": {CampaignID: "camp-1", LatestSeq: 1},
			},
		},
		eventClient: &fakeEventHeadClient{
			headByCampaign: map[string]uint64{"camp-1": 2},
			listErrByOrder: map[string]error{
				"seq": errors.New("delta list failed"),
			},
		},
	}

	err := h.syncCampaignEventHeads(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "list campaign events for stale scopes") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartCacheInvalidationWorkerStopsOnCancel(t *testing.T) {
	cacheStore := &fakeInvalidationCacheStore{
		trackedCampaignIDs: []string{"camp-1"},
		cursors: map[string]webstorage.CampaignEventCursor{
			"camp-1": {CampaignID: "camp-1", LatestSeq: 1},
		},
	}
	eventClient := &fakeEventHeadClient{
		headByCampaign: map[string]uint64{"camp-1": 2},
	}

	stop, done := startCacheInvalidationWorker(cacheStore, eventClient)
	if stop == nil || done == nil {
		t.Fatalf("expected stop and done handles")
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for len(eventClient.listRequests) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if len(eventClient.listRequests) == 0 {
		t.Fatalf("expected worker to request campaign event head")
	}

	stop()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timeout waiting for worker stop")
	}
}

func TestStartCacheInvalidationWorkerSkipsWhenDependenciesMissing(t *testing.T) {
	stop, done := startCacheInvalidationWorker(nil, &fakeEventHeadClient{})
	if stop != nil || done != nil {
		t.Fatalf("expected nil handles for missing store")
	}
	stop, done = startCacheInvalidationWorker(&fakeInvalidationCacheStore{}, nil)
	if stop != nil || done != nil {
		t.Fatalf("expected nil handles for missing event client")
	}
}

func TestNormalizeInvalidationLoopInput(t *testing.T) {
	t.Run("nil handler is ignored", func(t *testing.T) {
		normalized, ok := normalizeInvalidationLoopInput(nil, context.Background(), time.Second)
		if ok {
			t.Fatalf("expected nil handler to return not ok")
		}
		if normalized.ctx != nil {
			t.Fatalf("normalized ctx = %v, want nil", normalized.ctx)
		}
		if normalized.interval != 0 {
			t.Fatalf("normalized interval = %v, want %v", normalized.interval, 0*time.Second)
		}
	})

	t.Run("defaults nil context and non-positive interval", func(t *testing.T) {
		h := &handler{}
		normalized, ok := normalizeInvalidationLoopInput(h, nil, 0)
		if !ok {
			t.Fatalf("expected normalization ok")
		}
		if normalized.ctx == nil {
			t.Fatalf("normalized ctx = nil, want non-nil")
		}
		if normalized.interval != cacheInvalidationInterval {
			t.Fatalf("normalized interval = %v, want %v", normalized.interval, cacheInvalidationInterval)
		}
	})

	t.Run("keeps explicit context and interval", func(t *testing.T) {
		h := &handler{}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		normalized, ok := normalizeInvalidationLoopInput(h, ctx, 3*time.Second)
		if !ok {
			t.Fatalf("expected normalization ok")
		}
		if normalized.ctx != ctx {
			t.Fatalf("normalized ctx mismatch")
		}
		if normalized.interval != 3*time.Second {
			t.Fatalf("normalized interval = %v, want %v", normalized.interval, 3*time.Second)
		}
	})
}

func TestCampaignEventHeadSeqHandlesEmptyEvents(t *testing.T) {
	seq, err := campaignEventHeadSeq(context.Background(), &fakeEventHeadClient{}, "camp-1")
	if err != nil {
		t.Fatalf("campaign event head seq: %v", err)
	}
	if seq != 0 {
		t.Fatalf("seq = %d, want %d", seq, 0)
	}
}

func TestCampaignEventHeadSeqHandlesNilClient(t *testing.T) {
	seq, err := campaignEventHeadSeq(context.Background(), nil, "camp-1")
	if err != nil {
		t.Fatalf("campaign event head seq: %v", err)
	}
	if seq != 0 {
		t.Fatalf("seq = %d, want %d", seq, 0)
	}
}

func TestCampaignEventHeadSeqHandlesNilContext(t *testing.T) {
	seq, err := campaignEventHeadSeq(nil, &fakeEventHeadClient{headByCampaign: map[string]uint64{"camp-1": 8}}, "camp-1")
	if err != nil {
		t.Fatalf("campaign event head seq: %v", err)
	}
	if seq != 8 {
		t.Fatalf("seq = %d, want %d", seq, 8)
	}
}

func TestCampaignEventHeadSeqReturnsListError(t *testing.T) {
	_, err := campaignEventHeadSeq(context.Background(), &fakeEventHeadClient{listErr: errors.New("boom")}, "camp-1")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestCampaignScopesForEventType(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		want      map[string]bool
	}{
		{
			name:      "campaign",
			eventType: "campaign.updated",
			want: map[string]bool{
				cacheScopeCampaignSummary: true,
			},
		},
		{
			name:      "participant",
			eventType: "participant.updated",
			want: map[string]bool{
				cacheScopeCampaignParticipants: true,
				cacheScopeCampaignSummary:      true,
			},
		},
		{
			name:      "seat reassigned trimmed",
			eventType: " seat.reassigned ",
			want: map[string]bool{
				cacheScopeCampaignParticipants: true,
				cacheScopeCampaignSummary:      true,
			},
		},
		{
			name:      "session",
			eventType: "session.started",
			want: map[string]bool{
				cacheScopeCampaignSessions: true,
			},
		},
		{
			name:      "character",
			eventType: "character.updated",
			want: map[string]bool{
				cacheScopeCampaignCharacters: true,
				cacheScopeCampaignSummary:    true,
			},
		},
		{
			name:      "invite",
			eventType: "invite.created",
			want: map[string]bool{
				cacheScopeCampaignInvites: true,
			},
		},
		{
			name:      "unknown falls back to all scopes",
			eventType: "action.outcome_applied",
			want: map[string]bool{
				cacheScopeCampaignSummary:      true,
				cacheScopeCampaignParticipants: true,
				cacheScopeCampaignSessions:     true,
				cacheScopeCampaignCharacters:   true,
				cacheScopeCampaignInvites:      true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := campaignScopesForEventType(tc.eventType)
			if len(got) != len(tc.want) {
				t.Fatalf("scope count = %d, want %d (%v)", len(got), len(tc.want), got)
			}
			for _, scope := range got {
				if !tc.want[scope] {
					t.Fatalf("unexpected scope %q for event %q", scope, tc.eventType)
				}
			}
		})
	}
}

func TestCampaignInvalidationScopesSincePaginatesAndDeduplicates(t *testing.T) {
	events := make([]*statev1.Event, 0, 202)
	for i := 1; i <= 200; i++ {
		events = append(events, &statev1.Event{
			CampaignId: "camp-1",
			Seq:        uint64(i),
			Type:       "session.started",
		})
	}
	events = append(events,
		&statev1.Event{CampaignId: "camp-1", Seq: 201, Type: "character.created"},
		&statev1.Event{CampaignId: "camp-1", Seq: 202, Type: "session.ended"},
	)
	eventClient := &fakeEventHeadClient{
		eventsByCampaign: map[string][]*statev1.Event{
			"camp-1": events,
		},
	}

	scopes, err := campaignInvalidationScopesSince(context.Background(), eventClient, "camp-1", 0)
	if err != nil {
		t.Fatalf("campaign invalidation scopes since: %v", err)
	}

	expectedScopes := map[string]bool{
		cacheScopeCampaignSessions:   true,
		cacheScopeCampaignCharacters: true,
		cacheScopeCampaignSummary:    true,
	}
	if len(scopes) != len(expectedScopes) {
		t.Fatalf("scopes = %d, want %d (%v)", len(scopes), len(expectedScopes), scopes)
	}
	for _, scope := range scopes {
		if !expectedScopes[scope] {
			t.Fatalf("unexpected scope %q", scope)
		}
	}

	if len(eventClient.listRequests) != 2 {
		t.Fatalf("list requests = %d, want %d", len(eventClient.listRequests), 2)
	}
	firstReq := eventClient.listRequests[0]
	if firstReq.GetOrderBy() != "seq" {
		t.Fatalf("first order by = %q, want %q", firstReq.GetOrderBy(), "seq")
	}
	if firstReq.GetPageSize() != 200 {
		t.Fatalf("first page size = %d, want %d", firstReq.GetPageSize(), 200)
	}
	if firstReq.GetAfterSeq() != 0 {
		t.Fatalf("first after seq = %d, want %d", firstReq.GetAfterSeq(), 0)
	}
	if strings.TrimSpace(firstReq.GetPageToken()) != "" {
		t.Fatalf("first page token = %q, want empty", firstReq.GetPageToken())
	}
	secondReq := eventClient.listRequests[1]
	if strings.TrimSpace(secondReq.GetPageToken()) == "" {
		t.Fatalf("second page token = empty, want non-empty")
	}
}

func TestCampaignInvalidationScopesSinceNilClient(t *testing.T) {
	scopes, err := campaignInvalidationScopesSince(context.Background(), nil, "camp-1", 0)
	if err != nil {
		t.Fatalf("campaign invalidation scopes since: %v", err)
	}
	if len(scopes) != 0 {
		t.Fatalf("scopes = %v, want empty", scopes)
	}
}

func TestCampaignInvalidationScopesSinceNilContext(t *testing.T) {
	eventClient := &fakeEventHeadClient{
		eventsByCampaign: map[string][]*statev1.Event{
			"camp-1": {
				{CampaignId: "camp-1", Seq: 1, Type: "session.started"},
			},
		},
	}

	scopes, err := campaignInvalidationScopesSince(nil, eventClient, "camp-1", 0)
	if err != nil {
		t.Fatalf("campaign invalidation scopes since: %v", err)
	}
	if len(scopes) != 1 || scopes[0] != cacheScopeCampaignSessions {
		t.Fatalf("scopes = %v, want [%s]", scopes, cacheScopeCampaignSessions)
	}
}

func TestCampaignInvalidationScopesSinceReturnsListError(t *testing.T) {
	_, err := campaignInvalidationScopesSince(context.Background(), &fakeEventHeadClient{listErr: errors.New("boom")}, "camp-1", 0)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "list events") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCampaignInvalidationScopesSinceSkipsNilEvents(t *testing.T) {
	eventClient := &fakeEventHeadClient{
		eventsByCampaign: map[string][]*statev1.Event{
			"camp-1": {
				nil,
				{CampaignId: "camp-1", Seq: 2, Type: "invite.created"},
			},
		},
	}

	scopes, err := campaignInvalidationScopesSince(context.Background(), eventClient, "camp-1", 0)
	if err != nil {
		t.Fatalf("campaign invalidation scopes since: %v", err)
	}
	if len(scopes) != 1 || scopes[0] != cacheScopeCampaignInvites {
		t.Fatalf("scopes = %v, want [%s]", scopes, cacheScopeCampaignInvites)
	}
}

func TestResolveCampaignStaleScopes(t *testing.T) {
	tests := []struct {
		name        string
		cursorKnown bool
		cursorSeq   uint64
		headSeq     uint64
		delta       []string
		want        map[string]bool
	}{
		{
			name:        "no cursor means no stale scopes",
			cursorKnown: false,
			cursorSeq:   0,
			headSeq:     10,
			delta:       []string{cacheScopeCampaignSessions},
			want:        nil,
		},
		{
			name:        "head unchanged means no stale scopes",
			cursorKnown: true,
			cursorSeq:   10,
			headSeq:     10,
			delta:       []string{cacheScopeCampaignSessions},
			want:        nil,
		},
		{
			name:        "head advanced with delta scopes",
			cursorKnown: true,
			cursorSeq:   10,
			headSeq:     11,
			delta:       []string{cacheScopeCampaignSessions, cacheScopeCampaignInvites},
			want: map[string]bool{
				cacheScopeCampaignSessions: true,
				cacheScopeCampaignInvites:  true,
			},
		},
		{
			name:        "head advanced without delta scopes falls back to defaults",
			cursorKnown: true,
			cursorSeq:   10,
			headSeq:     12,
			delta:       nil,
			want: map[string]bool{
				cacheScopeCampaignSummary:      true,
				cacheScopeCampaignParticipants: true,
				cacheScopeCampaignSessions:     true,
				cacheScopeCampaignCharacters:   true,
				cacheScopeCampaignInvites:      true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveCampaignStaleScopes(tc.cursorKnown, tc.cursorSeq, tc.headSeq, tc.delta)
			if tc.want == nil {
				if len(got) != 0 {
					t.Fatalf("stale scopes = %v, want none", got)
				}
				return
			}
			if len(got) != len(tc.want) {
				t.Fatalf("scope count = %d, want %d (%v)", len(got), len(tc.want), got)
			}
			for _, scope := range got {
				if !tc.want[scope] {
					t.Fatalf("unexpected scope %q", scope)
				}
			}
		})
	}
}

func TestLimitCampaignIDsForSync_RotatesAcrossPasses(t *testing.T) {
	resetCampaignSyncRoundRobinState()
	t.Cleanup(resetCampaignSyncRoundRobinState)

	campaignIDs := []string{"camp-1", "camp-2", "camp-3", "camp-4"}

	first := limitCampaignIDsForSync(campaignIDs, 2)
	if len(first) != 2 || first[0] != "camp-1" || first[1] != "camp-2" {
		t.Fatalf("first batch = %v, want [camp-1 camp-2]", first)
	}

	second := limitCampaignIDsForSync(campaignIDs, 2)
	if len(second) != 2 || second[0] != "camp-3" || second[1] != "camp-4" {
		t.Fatalf("second batch = %v, want [camp-3 camp-4]", second)
	}

	third := limitCampaignIDsForSync(campaignIDs, 2)
	if len(third) != 2 || third[0] != "camp-1" || third[1] != "camp-2" {
		t.Fatalf("third batch = %v, want [camp-1 camp-2]", third)
	}
}
