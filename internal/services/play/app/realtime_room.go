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
	for {
		stream, err := r.hub.server.events.SubscribeCampaignUpdates(r.ctx, &gamev1.SubscribeCampaignUpdatesRequest{
			CampaignId:       r.campaignID,
			Kinds:            []gamev1.CampaignUpdateKind{gamev1.CampaignUpdateKind_CAMPAIGN_UPDATE_KIND_PROJECTION_APPLIED},
			ProjectionScopes: []string{"campaign_sessions", "campaign_scenes"},
		})
		if err != nil {
			if !r.hub.runtime.retry(r.ctx) {
				return
			}
			continue
		}
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
			return r.hub.runtime.retry(r.ctx)
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
	app := r.hub.server.application()
	for _, session := range r.sessionsSnapshot() {
		state, err := app.interactionState(r.ctx, playRequest{
			campaignRequest: campaignRequest{CampaignID: r.campaignID},
			UserID:          session.userID,
		})
		if err != nil {
			_ = session.peer.writeFrame(wsFrame{Type: "play.resync", Payload: mustJSON(map[string]string{"reason": "interaction state changed; reload required"})})
			continue
		}
		session.attach(r, state)
		snapshot, err := app.roomSnapshotFromState(r.ctx, r.campaignID, state, r.latestGameSequence())
		if err != nil {
			_ = session.peer.writeFrame(wsFrame{Type: "play.resync", Payload: mustJSON(map[string]string{"reason": "interaction state changed; reload required"})})
			continue
		}
		_ = session.peer.writeFrame(wsFrame{Type: "play.interaction.updated", Payload: mustJSON(snapshot)})
	}
}

func (r *campaignRoom) broadcastFrame(frame wsFrame) {
	for _, session := range r.sessionsSnapshot() {
		_ = session.peer.writeFrame(frame)
	}
}
