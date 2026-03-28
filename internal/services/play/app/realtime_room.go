package app

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	playprotocol "github.com/louisbranch/fracturing.space/internal/services/play/protocol"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// campaignRoom owns one campaign's session set and projection-driven fanout.
type campaignRoom struct {
	hub        *realtimeHub
	campaignID string

	ctx    context.Context
	cancel context.CancelFunc

	mu          sync.Mutex
	sessions    map[*realtimeSession]struct{}
	lastGameSeq uint64
	authUserID  string

	subscriptionStarted bool
	aiDebugSessionID    string
	aiDebugCancel       context.CancelFunc
}

func (r *campaignRoom) runProjectionSubscription() {
	retryDelay := r.hub.runtime.projectionRetryTTL
	for {
		authCtx, userID, ok := r.subscriptionContext()
		if !ok {
			if !r.hub.runtime.retryWithDelay(r.ctx, retryDelay) {
				return
			}
			retryDelay = r.hub.runtime.backoff(retryDelay)
			continue
		}
		afterSeq := r.latestGameSequence()
		r.hub.log().InfoContext(r.ctx, "play realtime: subscribing to campaign updates",
			"campaign_id", r.campaignID,
			"user_id", userID,
			"after_seq", afterSeq,
			"projection_scopes", []string{"campaign_sessions", "campaign_scenes"},
		)
		stream, err := r.hub.deps.events.SubscribeCampaignUpdates(authCtx, &gamev1.SubscribeCampaignUpdatesRequest{
			CampaignId:       r.campaignID,
			AfterSeq:         afterSeq,
			Kinds:            []gamev1.CampaignUpdateKind{gamev1.CampaignUpdateKind_CAMPAIGN_UPDATE_KIND_PROJECTION_APPLIED},
			ProjectionScopes: []string{"campaign_sessions", "campaign_scenes"},
		})
		if err != nil {
			r.hub.log().WarnContext(r.ctx, "play realtime: subscribe failed",
				"campaign_id", r.campaignID,
				"user_id", userID,
				"after_seq", afterSeq,
				"grpc_code", status.Code(err).String(),
				"error", err,
			)
			if !r.hub.runtime.retryWithDelay(r.ctx, retryDelay) {
				return
			}
			retryDelay = r.hub.runtime.backoff(retryDelay)
			continue
		}
		r.hub.log().InfoContext(r.ctx, "play realtime: campaign update stream connected",
			"campaign_id", r.campaignID,
			"user_id", userID,
			"after_seq", afterSeq,
		)
		// A successful stream connection resets the backoff.
		retryDelay = r.hub.runtime.projectionRetryTTL
		if !r.consumeProjectionStream(stream) {
			return
		}
	}
}

func (r *campaignRoom) consumeProjectionStream(stream gogrpc.ServerStreamingClient[gamev1.CampaignUpdate]) bool {
	for {
		update, recvErr := stream.Recv()
		if recvErr != nil {
			if errors.Is(recvErr, io.EOF) {
				r.hub.log().InfoContext(r.ctx, "play realtime: campaign update stream closed by server",
					"campaign_id", r.campaignID,
					"latest_game_seq", r.latestGameSequence(),
				)
				return true
			}
			r.hub.log().WarnContext(r.ctx, "play realtime: campaign update stream recv failed",
				"campaign_id", r.campaignID,
				"latest_game_seq", r.latestGameSequence(),
				"grpc_code", status.Code(recvErr).String(),
				"error", recvErr,
			)
			return r.hub.runtime.retryWithDelay(r.ctx, r.hub.runtime.projectionRetryTTL)
		}
		if update == nil || update.GetProjectionApplied() == nil {
			continue
		}
		r.hub.log().InfoContext(r.ctx, "play realtime: projection update received",
			"campaign_id", r.campaignID,
			"seq", update.GetSeq(),
			"event_type", update.GetEventType(),
			"scopes", update.GetProjectionApplied().GetScopes(),
		)
		r.setLatestGameSequence(update.GetSeq())
		r.broadcastCurrent()
	}
}

func (r *campaignRoom) add(session *realtimeSession) {
	r.mu.Lock()
	r.sessions[session] = struct{}{}
	if r.authUserID == "" {
		r.authUserID = session.userID
	}
	r.mu.Unlock()
}

// remove deletes a session from the room. If the room becomes empty, it
// cancels the room context and removes itself from the hub.
// Lock ordering: r.mu must be released before acquiring r.hub.mu to prevent deadlock.
func (r *campaignRoom) remove(session *realtimeSession) {
	r.mu.Lock()
	delete(r.sessions, session)
	if r.authUserID == session.userID {
		r.authUserID = r.firstSessionUserIDLocked()
	}
	if len(r.sessions) != 0 {
		r.mu.Unlock()
		return
	}
	r.mu.Unlock()
	r.cancel()
	r.hub.mu.Lock()
	delete(r.hub.rooms, r.campaignID)
	r.hub.mu.Unlock()
}

func (r *campaignRoom) setLatestGameSequence(seq uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if seq > r.lastGameSeq {
		r.lastGameSeq = seq
	}
}

func (r *campaignRoom) latestGameSequence() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastGameSeq
}

func (r *campaignRoom) sessionsSnapshot() []*realtimeSession {
	r.mu.Lock()
	defer r.mu.Unlock()
	values := make([]*realtimeSession, 0, len(r.sessions))
	for session := range r.sessions {
		values = append(values, session)
	}
	return values
}

func (r *campaignRoom) currentAIDebugSession() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.aiDebugSessionID
}

// ensureProjectionSubscription starts a single projection subscription
// goroutine per room. The goroutine exits when the room context is cancelled,
// which happens when the last session disconnects via remove().
func (r *campaignRoom) ensureProjectionSubscription() {
	start := false
	r.mu.Lock()
	if !r.subscriptionStarted {
		r.subscriptionStarted = true
		start = true
	}
	r.mu.Unlock()
	if start {
		go r.runProjectionSubscription()
	}
}

func (r *campaignRoom) reconcileAIDebugSubscription(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)

	var cancel context.CancelFunc
	var startCtx context.Context
	start := false

	r.mu.Lock()
	if r.aiDebugSessionID == sessionID {
		r.mu.Unlock()
		return
	}
	cancel = r.aiDebugCancel
	r.aiDebugCancel = nil
	r.aiDebugSessionID = sessionID
	if sessionID != "" {
		startCtx, r.aiDebugCancel = context.WithCancel(r.ctx)
		start = true
	}
	r.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if start {
		go r.runAIDebugSubscription(startCtx, sessionID)
	}
}

func (r *campaignRoom) subscriptionContextWithBase(base context.Context) (context.Context, string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	userID := r.authUserID
	if userID == "" {
		userID = r.firstSessionUserIDLocked()
		r.authUserID = userID
	}
	if userID == "" {
		return nil, "", false
	}
	return grpcauthctx.WithUserID(base, userID), userID, true
}

func (r *campaignRoom) subscriptionContext() (context.Context, string, bool) {
	return r.subscriptionContextWithBase(r.ctx)
}

func (r *campaignRoom) firstSessionUserIDLocked() string {
	for session := range r.sessions {
		if session != nil && session.userID != "" {
			return session.userID
		}
	}
	return ""
}

func (r *campaignRoom) broadcastCurrent() {
	sessions := r.sessionsSnapshot()
	if len(sessions) == 0 {
		return
	}

	// Invariant: GetInteractionState returns campaign-level state that is
	// participant-independent. Any authenticated user in the room produces the
	// same result, so we use the first session's auth context.
	app := r.hub.deps.application()
	req := playRequest{
		campaignRequest: campaignRequest{CampaignID: r.campaignID},
		UserID:          sessions[0].userID,
	}
	state, err := app.interactionState(r.ctx, req)
	if err != nil {
		r.broadcastResync(sessions)
		return
	}

	// Build enrichment data and chat cursor once for the campaign.
	snapshot, err := app.roomSnapshotFromState(r.ctx, req, state, r.latestGameSequence())
	if err != nil {
		r.hub.log().WarnContext(r.ctx, "play realtime: broadcast current failed; requesting resync",
			"campaign_id", r.campaignID,
			"error", err,
		)
		r.broadcastResync(sessions)
		return
	}
	activeSceneID := ""
	if snapshot.InteractionState.ActiveScene != nil {
		activeSceneID = snapshot.InteractionState.ActiveScene.SceneID
	}
	activeSessionID := ""
	if snapshot.InteractionState.ActiveSession != nil {
		activeSessionID = snapshot.InteractionState.ActiveSession.SessionID
	}
	aiTurnStatus := ""
	if snapshot.InteractionState.AITurn != nil {
		aiTurnStatus = snapshot.InteractionState.AITurn.Status
	}
	r.hub.log().InfoContext(r.ctx, "play realtime: broadcasting interaction update",
		"campaign_id", r.campaignID,
		"sessions", len(sessions),
		"latest_game_seq", snapshot.LatestGameSeq,
		"active_session_id", activeSessionID,
		"active_scene_id", activeSceneID,
		"ai_turn_status", aiTurnStatus,
	)
	r.reconcileAIDebugSubscription(activeSessionID)

	payload := mustJSON(snapshot)
	for _, session := range sessions {
		session.refreshCampaignState(snapshot.InteractionState)
		_ = session.peer.writeFrame(wsFrame{Type: FrameInteractionUpdated, Payload: payload})
	}
}

func (r *campaignRoom) broadcastResync(sessions []*realtimeSession) {
	r.hub.log().WarnContext(r.ctx, "play realtime: broadcasting resync",
		"campaign_id", r.campaignID,
		"sessions", len(sessions),
	)
	frame := wsFrame{Type: FrameResync, Payload: mustJSON(map[string]string{"reason": "interaction state changed; reload required"})}
	for _, session := range sessions {
		_ = session.peer.writeFrame(frame)
	}
}

func (r *campaignRoom) broadcastFrame(frame wsFrame) {
	for _, session := range r.sessionsSnapshot() {
		_ = session.peer.writeFrame(frame)
	}
}

func (r *campaignRoom) runAIDebugSubscription(ctx context.Context, sessionID string) {
	retryDelay := r.hub.runtime.projectionRetryTTL
	for {
		authCtx, userID, ok := r.subscriptionContextWithBase(ctx)
		if !ok {
			if !r.hub.runtime.retryWithDelay(ctx, retryDelay) {
				return
			}
			retryDelay = r.hub.runtime.backoff(retryDelay)
			continue
		}
		r.hub.log().InfoContext(ctx, "play realtime: subscribing to ai debug updates",
			"campaign_id", r.campaignID,
			"session_id", sessionID,
			"user_id", userID,
		)
		stream, err := r.hub.deps.aiDebug.SubscribeCampaignDebugUpdates(authCtx, &aiv1.SubscribeCampaignDebugUpdatesRequest{
			CampaignId: r.campaignID,
			SessionId:  sessionID,
		})
		if err != nil {
			r.hub.log().WarnContext(ctx, "play realtime: ai debug subscribe failed",
				"campaign_id", r.campaignID,
				"session_id", sessionID,
				"user_id", userID,
				"grpc_code", status.Code(err).String(),
				"error", err,
			)
			if !r.hub.runtime.retryWithDelay(ctx, retryDelay) {
				return
			}
			retryDelay = r.hub.runtime.backoff(retryDelay)
			continue
		}
		r.hub.log().InfoContext(ctx, "play realtime: ai debug stream connected",
			"campaign_id", r.campaignID,
			"session_id", sessionID,
			"user_id", userID,
		)
		retryDelay = r.hub.runtime.projectionRetryTTL
		if !r.consumeAIDebugStream(ctx, sessionID, stream) {
			return
		}
	}
}

func (r *campaignRoom) consumeAIDebugStream(
	ctx context.Context,
	sessionID string,
	stream gogrpc.ServerStreamingClient[aiv1.CampaignDebugTurnUpdate],
) bool {
	for {
		update, recvErr := stream.Recv()
		if recvErr != nil {
			if errors.Is(recvErr, io.EOF) {
				r.hub.log().InfoContext(ctx, "play realtime: ai debug stream closed by server",
					"campaign_id", r.campaignID,
					"session_id", sessionID,
				)
			} else {
				r.hub.log().WarnContext(ctx, "play realtime: ai debug stream recv failed",
					"campaign_id", r.campaignID,
					"session_id", sessionID,
					"grpc_code", status.Code(recvErr).String(),
					"error", recvErr,
				)
			}
			return r.hub.runtime.retryWithDelay(ctx, r.hub.runtime.projectionRetryTTL)
		}
		if update == nil {
			continue
		}
		if r.currentAIDebugSession() != sessionID {
			return true
		}
		payload := playprotocol.AIDebugTurnUpdateFromProto(update)
		if strings.TrimSpace(payload.Turn.ID) == "" {
			continue
		}
		r.hub.log().InfoContext(ctx, "play realtime: ai debug update received",
			"campaign_id", r.campaignID,
			"session_id", sessionID,
			"turn_id", payload.Turn.ID,
			"status", payload.Turn.Status,
			"appended_entries", len(payload.AppendedEntries),
		)
		r.broadcastFrame(wsFrame{Type: FrameAIDebugTurnUpdated, Payload: mustJSON(payload)})
	}
}
