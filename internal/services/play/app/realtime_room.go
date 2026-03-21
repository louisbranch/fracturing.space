package app

import (
	"context"
	"errors"
	"io"
	"sync"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	gogrpc "google.golang.org/grpc"
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
}

func (r *campaignRoom) runProjectionSubscription() {
	retryDelay := r.hub.runtime.projectionRetryTTL
	for {
		stream, err := r.hub.deps.events.SubscribeCampaignUpdates(r.ctx, &gamev1.SubscribeCampaignUpdatesRequest{
			CampaignId:       r.campaignID,
			Kinds:            []gamev1.CampaignUpdateKind{gamev1.CampaignUpdateKind_CAMPAIGN_UPDATE_KIND_PROJECTION_APPLIED},
			ProjectionScopes: []string{"campaign_sessions", "campaign_scenes"},
		})
		if err != nil {
			if !r.hub.runtime.retryWithDelay(r.ctx, retryDelay) {
				return
			}
			retryDelay = r.hub.runtime.backoff(retryDelay)
			continue
		}
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
				return true
			}
			return r.hub.runtime.retryWithDelay(r.ctx, r.hub.runtime.projectionRetryTTL)
		}
		if update == nil || update.GetProjectionApplied() == nil {
			continue
		}
		r.setLatestGameSequence(update.GetSeq())
		r.broadcastCurrent()
	}
}

func (r *campaignRoom) add(session *realtimeSession) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session] = struct{}{}
}

func (r *campaignRoom) remove(session *realtimeSession) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, session)
	if len(r.sessions) != 0 {
		return
	}
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

func (r *campaignRoom) broadcastCurrent() {
	sessions := r.sessionsSnapshot()
	if len(sessions) == 0 {
		return
	}

	// Fetch interaction state once — campaign-level fields (active session,
	// scene, player phase, OOC) are the same for every participant. We use the
	// first session's auth context because GetInteractionState requires one.
	app := r.hub.deps.application()
	state, err := app.interactionState(r.ctx, playRequest{
		campaignRequest: campaignRequest{CampaignID: r.campaignID},
		UserID:          sessions[0].userID,
	})
	if err != nil {
		r.broadcastResync(sessions)
		return
	}

	// Build enrichment data and chat cursor once for the campaign.
	snapshot, err := app.roomSnapshotFromState(r.ctx, r.campaignID, state, r.latestGameSequence())
	if err != nil {
		r.broadcastResync(sessions)
		return
	}

	payload := mustJSON(snapshot)
	for _, session := range sessions {
		session.refreshCampaignState(snapshot.InteractionState)
		_ = session.peer.writeFrame(wsFrame{Type: "play.interaction.updated", Payload: payload})
	}
}

func (r *campaignRoom) broadcastResync(sessions []*realtimeSession) {
	frame := wsFrame{Type: "play.resync", Payload: mustJSON(map[string]string{"reason": "interaction state changed; reload required"})}
	for _, session := range sessions {
		_ = session.peer.writeFrame(frame)
	}
}

func (r *campaignRoom) broadcastFrame(frame wsFrame) {
	for _, session := range r.sessionsSnapshot() {
		_ = session.peer.writeFrame(frame)
	}
}
